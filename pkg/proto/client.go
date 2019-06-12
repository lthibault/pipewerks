package proto

import (
	"context"
	"net"
	"sync"

	"github.com/SentimensRG/ctx"
	pipe "github.com/lthibault/pipewerks/pkg"
	synctoolz "github.com/lthibault/toolz/pkg/sync"
)

// DefaultStrategy is a global dial strategy that allows dialers to share a global
// connection & stream pool.
var DefaultStrategy DialStrategy = &StreamCountStrategy{cs: make(map[string]*synctoolz.Var)}

// PipeDialer is the client end of a Pipewerks Transport.
type PipeDialer interface {
	Dial(context.Context, net.Addr) (pipe.Conn, error)
}

// DialStrategy is responsible dialing connections and opening streams.  This is where
// connection/stream reuse is to be implemented.
type DialStrategy interface {
	GetConn(context.Context, PipeDialer, net.Addr) (c pipe.Conn, cached bool, err error)
}

// A Client connects to a server
type Client struct {
	Dialer   PipeDialer
	Strategy DialStrategy

	o sync.Once
}

func (c *Client) init() {
	if c.Strategy == nil {
		c.Strategy = DefaultStrategy
	}
}

// Connect to the specified server
func (c *Client) Connect(ctx context.Context, a net.Addr) (pipe.Stream, error) {
	c.o.Do(c.init)

	conn, _, err := c.Strategy.GetConn(ctx, c.Dialer, a)
	if err != nil {
		return nil, err
	}

	return conn.OpenStream()
}

// StreamCountStrategy automatically closes connections when the stream count reaches
// zero.
type StreamCountStrategy struct {
	mu sync.Mutex
	cs map[string]*synctoolz.Var
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

	var v synctoolz.Var
	v.Set(&ctrConn{Conn: conn, Ctr: synctoolz.Ctr(n)})
	ds.cs[conn.RemoteAddr().String()] = &v
}

func (ds *StreamCountStrategy) getVar(key string) (v *synctoolz.Var, cached bool) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if v, cached = ds.cs[key]; !cached {
		v = &synctoolz.Var{}
		ds.cs[key] = v
	}

	return
}

// GetConn returns an existing conn if one exists for the given address, else dials a
// new connection.
func (ds *StreamCountStrategy) GetConn(c context.Context, d PipeDialer, a net.Addr) (pipe.Conn, bool, error) {
	v, cached := ds.getVar(a.String())

	if cached {
		select {
		case <-v.Ready():
			cc := v.Get().(*ctrConn)
			return cc, cached, cc.Error
		case <-c.Done():
			return nil, cached, c.Err()
		}
	}

	// slow path
	cxn, err := d.Dial(c, a)
	if err != nil {
		v.Set(&ctrConn{Error: err})
		return nil, false, err
	}

	cc := &ctrConn{Conn: cxn}
	v.Set(cc)
	ctx.Defer(cxn.Context(), ds.gc(cxn.RemoteAddr().String()))

	return cc, false, nil
}

type ctrConn struct {
	mu    sync.RWMutex
	Error error
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
