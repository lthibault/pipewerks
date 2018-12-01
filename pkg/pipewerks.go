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
	Accept() (Conn, error)
}

// Conn represents a logical connection between two peers.  Streams are
// multiplexed onto connections
type Conn interface {
	Context() context.Context
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	AcceptStream() (Stream, error)
	OpenStream() (Stream, error)
	Close() error
}

// Stream is a bidirectional connection between two hosts.
type Stream interface {
	Context() context.Context
	StreamID() uint32
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
	Close() error
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}
