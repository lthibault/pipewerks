package pipe

import (
	"context"
	"net"
	"time"
)

// Transport is a means by which to connect to an listen for connections from
// other peers.
type Transport interface {
	Listen(context.Context, net.Addr) (Listener, error)
	Dial(context.Context, net.Addr) (Conn, error)
}

// Listener can listen for incoming connections
type Listener interface {
	Addr() net.Addr
	Close() error
	Accept(context.Context) (Conn, error)
}

// Conn represents a logical connection between two peers.  Streams are
// multiplexed onto connections
type Conn interface {
	Context() context.Context
	Stream() Streamer
	Endpoint() Edge
	Close() error
}

// Streamer can open and close various types of streams
type Streamer interface {
	Accept() (Stream, error)
	Open() (Stream, error)
}

// Edge identifies a connection between to endpoints
type Edge interface {
	Local() net.Addr
	Remote() net.Addr
}

// Stream is a bidirectional connection between two hosts.
type Stream interface {
	Context() context.Context
	Endpoint() Edge
	Close() error
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}
