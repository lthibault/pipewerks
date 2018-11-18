package net

import (
	"context"
	"net"
	"time"
)

// Addr represents a network end point address.
//
// The two methods Network and String conventionally return strings
// that can be passed as the arguments to Dial, but the exact form
// and meaning of the strings is up to the implementation.
type Addr = net.Addr

// An Error represents a network error.
type Error interface {
	error
	Timeout() bool   // Is the error a timeout?
	Temporary() bool // Is the error temporary?
}

// ErrorCode is used to terminate a connection and signal an error
type ErrorCode uint16

// Transport is a means by which to connect to an listen for connections from
// other peers.
type Transport interface {
	Listen(context.Context, Addr) (Listener, error)
	Dial(context.Context, Addr) (Conn, error)
}

// Listener can listen for incoming connections
type Listener interface {
	Addr() Addr
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
	CloseWithError(ErrorCode, error) error
}

// Streamer can open and close various types of streams
type Streamer interface {
	Accept() (Stream, error)
	Open() (Stream, error)
}

// Edge identifies a connection between to endpoints
type Edge interface {
	Local() Addr
	Remote() Addr
}

// Stream is a bidirectional connection between two hosts
type Stream interface {
	Context() context.Context
	Endpoint() Edge
	Close() error
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}
