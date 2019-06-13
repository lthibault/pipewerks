package proto

import (
	"context"
	"net"
	"sync"

	pipe "github.com/lthibault/pipewerks/pkg"
)

// DefaultStrategy is a global dial strategy that allows dialers to share a global
// connection & stream pool.
var DefaultStrategy DialStrategy = NewStreamCountStrategy()

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
