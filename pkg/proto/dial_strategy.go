package proto

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/SentimensRG/ctx"
	pipe "github.com/lthibault/pipewerks/pkg"
	synctoolz "github.com/lthibault/toolz/pkg/sync"
)

// StreamCountStrategy automatically closes connections when the stream count reaches
// zero.
type StreamCountStrategy struct {
	init sync.Once
	mu   sync.Mutex
	cs   map[string]*cacheDialer

	OnConnOpened func(pipe.Conn) error
}

func (s *StreamCountStrategy) gc(addr string) func() {
	return func() {
		s.mu.Lock()
		delete(s.cs, addr)
		s.mu.Unlock()
	}
}

func (s *StreamCountStrategy) setup() {
	if s.cs == nil {
		s.cs = make(map[string]*cacheDialer)
	}

	if s.OnConnOpened == nil {
		s.OnConnOpened = func(pipe.Conn) error { return nil }
	}
}

// Track the connection. n should be set to the number of open streams on the connection.
// It is intended to help DialStrategy implementers extend default behavior.
func (s *StreamCountStrategy) Track(conn pipe.Conn, n uint32) error {
	s.init.Do(s.setup)

	s.mu.Lock()
	defer s.mu.Unlock()

	key := conn.RemoteAddr().String()
	if _, ok := s.cs[key]; ok {
		return errors.New("already tracking")
	}

	d := &cacheDialer{gc: s.gc(key)}
	defer d.Do(func() {}) // disarm

	s.cs[key] = d
	d.conn = &ctrConn{Conn: conn, Ctr: synctoolz.Ctr(n)}
	ctx.Defer(conn.Context(), d.gc)

	return nil
}

func (s *StreamCountStrategy) getDialer(a net.Addr) *cacheDialer {
	s.mu.Lock()
	defer s.mu.Unlock()

	cd, ok := s.cs[a.String()]
	if !ok {
		cd = &cacheDialer{gc: s.gc(a.String()), cb: s.OnConnOpened}
		s.cs[a.String()] = cd
	}

	return cd
}

// GetConn returns an existing conn if one exists for the given address, else dials a
// new connection.
func (s *StreamCountStrategy) GetConn(c context.Context, d PipeDialer, a net.Addr) (pipe.Conn, error) {
	s.init.Do(s.setup)
	return s.getDialer(a).Dial(c, d, a)
}

type cacheDialer struct {
	sync.Once
	p PipeDialer

	gc   func()
	cb   func(pipe.Conn) error
	conn *ctrConn
	err  error
}

func (d *cacheDialer) Dial(c context.Context, p PipeDialer, a net.Addr) (*ctrConn, error) {
	d.Do(func() {
		var conn pipe.Conn
		if conn, d.err = p.Dial(c, a); d.err != nil {
			d.gc()
			return
		}

		if d.err = d.cb(d.conn); d.err != nil {
			d.conn.Close() // will call gc
			d.gc()
			return
		}

		d.conn = &ctrConn{Conn: conn}
		ctx.Defer(conn.Context(), d.gc)
	})

	return d.conn, d.err
}

type ctrConn struct {
	mu sync.RWMutex
	synctoolz.Ctr
	pipe.Conn
}

func (c *ctrConn) gc() {
	c.mu.Lock()
	if c.Ctr.Decr() == 0 {
		c.Close()
	}
	c.mu.Unlock()
}

func (c *ctrConn) wrapStream(s pipe.Stream) pipe.Stream {
	return &ctrStream{Stream: s, done: c.gc}
}

func (c *ctrConn) AcceptStream() (s pipe.Stream, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if s, err = c.Conn.AcceptStream(); err == nil {
		c.Ctr.Incr()
		s = c.wrapStream(s)
	}

	return
}

func (c *ctrConn) OpenStream() (s pipe.Stream, err error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if s, err = c.Conn.OpenStream(); err == nil {
		c.Ctr.Incr()
		s = c.wrapStream(s)
	}

	return
}

type ctrStream struct {
	pipe.Stream
	done func()
}

func (s ctrStream) Close() error {
	defer s.done() // decr-ing before close might cause Close() to report errors
	return s.Stream.Close()
}
