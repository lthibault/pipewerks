package protocol

import (
	"context"
	"net"
	"sync"

	log "github.com/lthibault/log/pkg"
	pipe "github.com/lthibault/pipewerks/pkg"
	"golang.org/x/sync/errgroup"
)

// DefaultStrategy is a global dial strategy that allows dialers to share a global
// connection & stream pool.
var DefaultStrategy DialStrategy = &defaultStrategy{
	cs: make(map[string]pipe.Conn),
	ss: make(map[pipe.Conn]streamPool),
}

// PipeDialer is the client end of a Pipewerks Transport.
type PipeDialer interface {
	Dial(context.Context, net.Addr) (pipe.Conn, error)
}

// DialStrategy is responsible dialing connections and opening streams.  This is where
// connection/stream reuse is to be implemented.
type DialStrategy interface {
	GetConn(context.Context, PipeDialer, net.Addr) (pipe.Conn, error)
	GetStream(pipe.Conn) (Stream, error)

	// Flush idle connections
	Flush() error
}

// Stream is a pipe.Stream whose Close() method returns it to a resource pool.  A Stream
// can be discarded with a call to Discard() when reuse is not desired.
type Stream interface {
	pipe.Stream
	Discard() error
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
func (c *Client) Connect(ctx context.Context, a net.Addr) (Stream, error) {
	c.o.Do(c.init)

	conn, err := c.Strategy.GetConn(ctx, c.Dialer, a)
	if err != nil {
		return nil, err
	}

	return c.Strategy.GetStream(conn)
}

// CloseIdleStreams closes any connections on its Transport which were previously
// connected from previous requests but are now sitting idle in a "keep-alive" state. It
// does not interrupt any connections currently in use.
func (c *Client) CloseIdleStreams() {
	c.o.Do(c.init)
	if err := c.Strategy.Flush(); err != nil {
		c.Logger.WithError(err).Debug("partial flush")
	}
}

type defaultStrategy struct {
	mu sync.Mutex

	cs map[string]pipe.Conn
	ss map[pipe.Conn]streamPool
}

func (ds *defaultStrategy) GetConn(ctx context.Context, d PipeDialer, a net.Addr) (conn pipe.Conn, err error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	var ok bool
	if conn, ok = ds.cs[a.String()]; !ok {
		if conn, err = d.Dial(ctx, a); err == nil {
			pool := newStreamPool(32)
			ds.cs[a.String()] = conn
			ds.ss[conn] = pool
		}
	}

	return
}

func (ds *defaultStrategy) GetStream(conn pipe.Conn) (Stream, error) {
	// ds.mu.Lock()
	// defer ds.mu.Unlock()
	// pool, ok := ds.ss[conn]
	// if !ok {
	// 	return nil, errors.New("not connected")
	// }

	// var ps pipe.Stream
	// if ps, ok = pool.Get(conn); !ok {
	// 	var err error
	// 	if ps, err = conn.OpenStream(); err != nil {
	// 		return nil, err
	// 	}
	// }

	panic("Stream NOT IMPLEMENTED")
	// return , nil
}

func (ds *defaultStrategy) Flush() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	var g errgroup.Group

	for _, q := range ds.ss {
		g.Go(q.CloseAll)
	}

	return g.Wait()
}
