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
	StreamID() uint32
	Context() context.Context
	Endpoint() Edge
	Close() error
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}

// RawConn allows users to take over the underlying connection in a Transport.
// For clarity:  this is the connection onto which Transport multiplexes streams.
// Most transports support RawConn with the notable exception of QUIC.
type RawConn interface {
	// Raw lets the caller take over the underlying net.Conn in a Transport.
	// After a call to Raw, it is the caller's responsibility to manage the
	// connection, including ensuring that no streams are open or otherwise
	// reading/writing to the Conn.
	//
	// The returned net.Conn may have read or write deadlines
	// already set, depending on configuration. It is the caller's
	// responsibility to set or clear those deadlines as needed.
	//
	// The Raw connection may receive unprocessed stream data if streams were
	// previously opened. It is highly recommended to call Raw only on fresh
	// Conns.
	Raw() net.Conn
}
