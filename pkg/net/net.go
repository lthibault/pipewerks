package net

import (
	"context"
	"time"
)

// Joint between two Pipes.  Break will stop the participating Pipes.  Wait
// blocks until the Joint is broken.
type Joint struct {
	Break func()
	Wait  func() error
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
	Endpoint() EndpointPair
	Close() error
	CloseWithError(ErrorCode, error) error
}

// Streamer can open and close various types of streams
type Streamer interface {
	Accept() (Stream, error)
	Open() (Stream, error)
}

// EndpointPair identifies the two endpoints
type EndpointPair interface {
	Local() Addr
	Remote() Addr
}

// Stream is a bidirectional connection between two hosts
type Stream interface {
	Path() string
	Context() context.Context
	Endpoint() EndpointPair
	Close() error
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	SetDeadline(time.Time) error
	SetReadDeadline(time.Time) error
	SetWriteDeadline(time.Time) error
}
