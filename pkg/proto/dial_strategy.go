package proto

import (
	"context"
	"net"
	"sync"

	"github.com/SentimensRG/ctx"
	pipe "github.com/lthibault/pipewerks/pkg"
	synctoolz "github.com/lthibault/toolz/pkg/sync"
)

// StreamCountStrategy automatically closes connections when the stream count reaches
// zero.
type StreamCountStrategy struct {
	mu sync.Mutex
	cs map[string]*cacheDialer
}

// NewStreamCountStrategy ...
func NewStreamCountStrategy() *StreamCountStrategy {
	return &StreamCountStrategy{cs: make(map[string]*cacheDialer)}
}

func (ds *StreamCountStrategy) gc(addr string) func() {
	return func() {
		ds.mu.Lock()
		delete(ds.cs, addr)
		ds.mu.Unlock()
	}
}

// Track the connection. n should be set to the number of open streams on the connection.
// It is intended to help DialStrategy implementers extend default behavior.
func (ds *StreamCountStrategy) Track(conn pipe.Conn, n uint32) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	key := conn.RemoteAddr().String()

	d := &cacheDialer{gc: ds.gc(key)}
	defer d.Do(func() {}) // disarm

	ds.cs[key] = d
	d.conn = &ctrConn{Conn: conn, Ctr: synctoolz.Ctr(n)}
	ctx.Defer(conn.Context(), d.gc)
}

func (ds *StreamCountStrategy) getDialer(a net.Addr) (*cacheDialer, bool) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	cd, ok := ds.cs[a.String()]
	if !ok {
		cd = &cacheDialer{gc: ds.gc(a.String())}
		ds.cs[a.String()] = cd
	}

	return cd, ok
}

// GetConn returns an existing conn if one exists for the given address, else dials a
// new connection.
func (ds *StreamCountStrategy) GetConn(c context.Context, d PipeDialer, a net.Addr) (pipe.Conn, bool, error) {
	cd, ok := ds.getDialer(a)
	conn, err := cd.Dial(c, d, a)
	return conn, ok, err
}

type cacheDialer struct {
	sync.Once
	p PipeDialer

	gc   func()
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

		ctx.Defer(conn.Context(), d.gc)
		d.conn = &ctrConn{Conn: conn}
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
