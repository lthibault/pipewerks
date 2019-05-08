package protocol

import (
	"context"
	"errors"
	"io"
	"net"
	"sync"
	"time"

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
	cs   connSet
}

// Serve streams.  Serve always returns a non-nil error and closes l.
func (s *Server) Serve(l pipe.Listener) error {
	s.init.Do(func() {
		s.cq = make(chan struct{})
		s.ls = &listenerSet{ls: make(map[*pipe.Listener]struct{}), mu: &s.mu}
		s.cs.cs = make(map[io.Closer]struct{})
		if s.Logger == nil {
			s.Logger = log.New(log.OptLevel(log.NullLevel))
		}
	})

	l = &closeOnceListener{Listener: l}
	defer l.Close()

	if !s.ls.Add(&l) {
		return ErrServerClosed
	}
	defer s.ls.Del(&l)

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
	b := backoff.Backoff{Max: time.Minute, Jitter: true}

	for {
		stream, err := conn.AcceptStream()
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				s.Logger.WithError(err).
					WithField("addr", conn.RemoteAddr()).
					WithField("retry", b.ForAttempt(b.Attempt())).
					Debug("failed to accept stream")
				time.Sleep(b.Duration())
				continue
			}
			return
		}

		go s.serveStream(stream)
	}
}

func (s *Server) serveStream(stream pipe.Stream) {
	s.cs.Add(stream)
	defer s.cs.Del(stream)
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
func (s *Server) Shutdown(ctx context.Context) error {
	var g errgroup.Group
	g.Go(s.ls.CloseAll)
	g.Go(func() error {
		ticker := time.NewTicker(time.Millisecond * 500)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
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

type connSet struct {
	mu sync.Mutex
	cs map[io.Closer]struct{}
}

func (c *connSet) quiescent() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.cs) == 0
}

func (c *connSet) Add(cl io.Closer) {
	c.mu.Lock()
	c.cs[cl] = struct{}{}
	c.mu.Unlock()
}

func (c *connSet) Del(cl io.Closer) {
	c.mu.Lock()
	delete(c.cs, cl)
	c.mu.Unlock()
}

func (c *connSet) CloseAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	var g errgroup.Group
	for conn := range c.cs {
		g.Go(conn.Close)
	}
	return g.Wait()
}
