package protocol

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

	"github.com/SentimensRG/ctx"
	synctoolz "github.com/lthibault/toolz/pkg/sync"

	"github.com/jpillora/backoff"
	log "github.com/lthibault/log/pkg"
	pipe "github.com/lthibault/pipewerks/pkg"
	"golang.org/x/sync/errgroup"
)

var (
	// ErrServerClosed indicates that the server is no longer accepting connections.
	ErrServerClosed = errors.New("server closed")
)

// Handler responds to an incoming stream
type Handler interface {
	ServeStream(pipe.Stream)
}

// HandlerFunc is a type-adapter to allow the use of ordinary functions as stream
// handlers.
type HandlerFunc func(pipe.Stream)

// ServeStream calles f(s)
func (f HandlerFunc) ServeStream(s pipe.Stream) { f(s) }

// Server is a generic server that handles incoming streams
type Server struct {
	Handler
	Backoff backoff.Backoff
	Logger  log.Logger

	init sync.Once
	mu   sync.Mutex
	cq   chan struct{}
	ls   *listenerSet
	cs   *connSet

	ConnStateHandler   func(pipe.Conn, ConnState)
	StreamStateHandler func(pipe.Stream, StreamState)
}

// Serve streams.  Serve always returns a non-nil error and closes l.
func (s *Server) Serve(l pipe.Listener) error {
	// Start by doing some initialization.  A Server can listen on multiple interfaces
	// at once, so setup needs to be idempotent.
	s.init.Do(func() {
		s.cq = make(chan struct{})
		s.ls = &listenerSet{ls: make(map[*pipe.Listener]struct{}), mu: &s.mu}
		s.cs = newConnSet()
		if s.Logger == nil {
			s.Logger = log.New(log.OptLevel(log.NullLevel))
		}
		if s.ConnStateHandler == nil {
			s.ConnStateHandler = func(pipe.Conn, ConnState) {}
		}
		if s.StreamStateHandler == nil {
			s.StreamStateHandler = func(pipe.Stream, StreamState) {}
		}
	})

	// There are two paths to closing a listener.  The first is by breaking out of the
	// main server loop below.  The second is via a call to Close()/Shutdown(), which
	// calls listenerSet.CloseAll().
	l = &closeOnceListener{Listener: l} // makes l.Close idempotent
	defer l.Close()

	if !s.ls.Add(&l) {
		return ErrServerClosed
	}
	defer s.ls.Del(&l)

	// Start the main server loop.  This accepts connections in until (a) the server is
	// closed (b) a non-temporary network outage occurs.
	for {
		conn, e := l.Accept()
		if e != nil {
			select {
			case <-s.cq:
				return ErrServerClosed
			default:
			}

			if ne, ok := e.(net.Error); ok && ne.Temporary() {
				s.Logger.WithError(e).
					WithField("addr", l.Addr()).
					WithField("retry", s.Backoff.ForAttempt(s.Backoff.Attempt())).
					Debug("failed to accept connection")
				time.Sleep(s.Backoff.Duration())
				continue
			}
			return e
		}

		go s.serveConn(conn)
	}
}

func (s *Server) serveConn(conn pipe.Conn) {
	s.ConnStateHandler(conn, ConnStateOpen)
	defer s.ConnStateHandler(conn, ConnStateClosed)

	// Start by getting a reference counter for the stream.  Initially, the counter is
	// set to zero.  When it reaches zero again, the stream will be closed.
	ref := s.cs.Add(conn)

	// Ensure the Conn is not closed until the client explicitly closed by the client
	ctx.Defer(conn.Context(), func() { ref.Incr().Decr() })

	// Configure exponential backoff in case we hit temporary netowrk outages.
	// TODO:  make this configurable (?)
	b := backoff.Backoff{Max: time.Minute, Jitter: true}

	// Start the conn loop.  This repeatedly accepts streams from the connection until a
	// non-temporary error is reached.
	for {
		stream, err := conn.AcceptStream()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				s.Logger.WithError(err).
					WithField("addr", conn.RemoteAddr()).
					WithField("retry", b.ForAttempt(b.Attempt())).
					Debug("temporary network outage")
				time.Sleep(b.Duration())
				continue
			}

			s.Logger.WithError(err).Debug("connection closed")
			return
		}

		go s.serveStream(stream, ref.Incr().Decr)
	}
}

func (s *Server) serveStream(stream pipe.Stream, done func() uint32) {
	s.StreamStateHandler(stream, StreamStateOpen)
	defer func() {
		select {
		case <-stream.Context().Done():
			s.StreamStateHandler(stream, StreamStateClosed)
		default:
			if done() == 1 { // last ref is the Conn, else we'd be disconnected.
				s.StreamStateHandler(stream, StreamStateIdle)
			}
		}
	}()

	s.ServeStream(stream)
}

// Close immediately, terminating all active pipe.Listeners and any connections.
// For graceful shutdown, use Shutdown.
func (s *Server) Close() error {
	select {
	case <-s.cq:
		return ErrServerClosed
	default:
		close(s.cq)
		return s.cs.CloseAll()
	}
}

// Shutdown gracefully shuts down the server without interrupting any active connections.
func (s *Server) Shutdown(c context.Context) error {
	var g errgroup.Group
	g.Go(s.ls.CloseAll)
	g.Go(func() error {
		ticker := time.NewTicker(time.Millisecond * 500)
		defer ticker.Stop()

		for {
			select {
			case <-c.Done():
				return c.Err()
			case <-ticker.C:
				if s.cs.quiescent() {
					return nil
				}
			}
		}
	})
	return g.Wait()
}

type closeOnceListener struct {
	sync.Once
	pipe.Listener
	err error
}

func (l *closeOnceListener) Close() error {
	l.Do(func() { l.err = l.Listener.Close() })
	return l.err
}

type listenerSet struct {
	mu     sync.Locker
	ls     map[*pipe.Listener]struct{}
	closed bool
}

func (s *listenerSet) Add(l *pipe.Listener) (active bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.ls[l] = struct{}{}
		active = true
	}

	return
}

func (s *listenerSet) Del(l *pipe.Listener) {
	s.mu.Lock()
	delete(s.ls, l)
	s.mu.Unlock()
}

func (s *listenerSet) CloseAll() error {
	s.mu.Lock()

	s.closed = true

	var g errgroup.Group
	for l := range s.ls {
		g.Go((*l).Close)
	}

	s.mu.Unlock()
	return g.Wait()
}

type refCounter interface {
	Incr() refCounter
	Decr() uint32
}

type ctr struct {
	synctoolz.Ctr
	cleanup func()
}

func (c *ctr) Incr() refCounter {
	c.Ctr.Incr()
	return c
}

func (c *ctr) Decr() (u uint32) {
	if u = c.Ctr.Decr(); u == 0 {
		c.cleanup()
	}
	return
}

type connSet struct {
	mu sync.Mutex
	cs map[io.Closer]*ctr
}

func newConnSet() *connSet {
	return &connSet{cs: make(map[io.Closer]*ctr)}
}

func (set *connSet) quiescent() bool {
	set.mu.Lock()
	defer set.mu.Unlock()
	return len(set.cs) == 0
}

func (set *connSet) Add(conn pipe.Conn) refCounter {
	c := &ctr{cleanup: set.gc(conn)}

	set.mu.Lock()
	set.cs[conn] = c
	set.mu.Unlock()

	return c
}

func (set *connSet) gc(conn pipe.Conn) func() {
	return func() {
		conn.Close()
		defer set.mu.Unlock()

		set.mu.Lock()
		delete(set.cs, conn)
	}
}

func (set *connSet) CloseAll() error {
	set.mu.Lock()
	defer set.mu.Unlock()

	var g errgroup.Group
	for s := range set.cs {
		g.Go(s.Close)
	}
	return g.Wait()
}
