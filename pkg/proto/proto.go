package proto

import pipe "github.com/lthibault/pipewerks/pkg"

const (
	// ConnStateOpen indicates the connection was opened
	ConnStateOpen ConnState = iota
	// ConnStateIdle indicates there are no active streams for this connection
	ConnStateIdle
	// ConnStateClosed indicates the connection was closed
	ConnStateClosed

	// StreamStateOpen indicates the stream was opened
	StreamStateOpen StreamState = iota
	// StreamStateClosed indicates the stream was closed
	StreamStateClosed
)

// ConnState tracks the state of a pipe.Conn
type ConnState uint8

func (c ConnState) String() string {
	switch c {
	case ConnStateOpen:
		return "connection open"
	case ConnStateIdle:
		return "connection idle"
	case ConnStateClosed:
		return "connection closed"
	}

	panic("unreachable")
}

// StreamState tracks the state of a pipe.Stream
type StreamState uint8

func (s StreamState) String() string {
	switch s {
	case StreamStateOpen:
		return "stream open"
	case StreamStateClosed:
		return "stream closed"
	}

	panic("unreachable")
}

// ConnStateHandler is called when a change to a connection state occurs
type ConnStateHandler interface {
	OnConnState(pipe.Conn, ConnState)
}

// StreamStateHandler is called when a change to a stream state occurs
type StreamStateHandler interface {
	OnStreamState(pipe.Stream, StreamState)
}

// ConnStateHandlerFunc is a function that satisfies ConnStateHandler
type ConnStateHandlerFunc func(pipe.Conn, ConnState)

// OnConnState calls the function that underpins ConnStateHandlerFunc
func (h ConnStateHandlerFunc) OnConnState(c pipe.Conn, s ConnState) { h(c, s) }

// StreamStateHandlerFunc is a function that satisfies ConnStateHandler
type StreamStateHandlerFunc func(pipe.Stream, StreamState)

// OnStreamState calls the function that underpins StreamStateHandlerFunc
func (h StreamStateHandlerFunc) OnStreamState(s pipe.Stream, state StreamState) {
	h(s, state)
}
