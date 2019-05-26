package proto

import (
	"context"
	"net"
	"sync"

	"github.com/SentimensRG/ctx"
	synctoolz "github.com/lthibault/toolz/pkg/sync"

	log "github.com/lthibault/log/pkg"
	pipe "github.com/lthibault/pipewerks/pkg"
)

// DefaultStrategy is a global dial strategy that allows dialers to share a global
// connection & stream pool.
var DefaultStrategy DialStrategy = &StreamCountStrategy{cs: make(map[string]*ctrConn)}

// PipeDialer is the client end of a Pipewerks Transport.
type PipeDialer interface {
	Dial(context.Context, net.Addr) (pipe.Conn, error)
}

// DialStrategy is responsible dialing connections and opening streams.  This is where
// connection/stream reuse is to be implemented.
type DialStrategy interface {
	GetConn(context.Context, PipeDialer, net.Addr) (c pipe.Conn, new bool, err error)
}

// A Client connects to a server
type Client struct {
	Dialer   PipeDialer
	Strategy DialStrategy
	Logger   log.Logger

	o sync.Once
}

func (c *Client) init() {
	if c.Logger == nil {
		c.Strategy = DefaultStrategy
		c.Logger = log.New(log.OptLevel(log.NullLevel))
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

// StreamCountStrategy automatically closes connections when the strea count reaches
// zero.
type StreamCountStrategy struct {
	mu sync.Mutex
	cs map[string]*ctrConn
}

func (ds *StreamCountStrategy) gc(addr string) func() {
	return func() {
		ds.mu.Lock()
		delete(ds.cs, addr)
		ds.mu.Unlock()
	}
}

// GetConn returns an existing conn if one exists for the given address, else dials a
// new connection.
func (ds *StreamCountStrategy) GetConn(c context.Context, d PipeDialer, a net.Addr) (pipe.Conn, bool, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	if conn, ok := ds.cs[a.String()]; ok {
		return conn, false, nil
	}

	// slow path
	conn, err := d.Dial(c, a)
	if err != nil {
		return nil, false, err
	}

	ds.cs[a.String()] = &ctrConn{Conn: conn}
	ctx.Defer(conn.Context(), ds.gc(a.String()))

	return ds.cs[a.String()], true, nil
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
