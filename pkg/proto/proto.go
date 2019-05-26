package proto

import pipe "github.com/lthibault/pipewerks/pkg"

const (
	ConnStateOpen ConnState = iota
	ConnStateClosed

	StreamStateOpen StreamState = iota
	StreamStateIdle
	StreamStateClosed
)

// ConnState tracks the state of a pipe.Conn
type ConnState uint8

func (c ConnState) String() string {
	switch c {
	case ConnStateOpen:
		return "connection open"
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
	case StreamStateIdle:
		return "stream idle"
	case StreamStateClosed:
		return "stream closed"
	}

	panic("unreachable")
}

// ConnStateHandler is called when a change to a connection state occurs
type ConnStateHandler interface {
	OnConnState(*pipe.Conn, ConnState)
}

// StreamStateHandler is called when a change to a stream state occurs
type StreamStateHandler interface {
	OnStreamState(*pipe.Stream, StreamState)
}

// ConnStateHandlerFunc is a function that satisfies ConnStateHandler
type ConnStateHandlerFunc func(*pipe.Conn, ConnState)

// OnConnState calls the function that underpins ConnStateHandlerFunc
func (h ConnStateHandlerFunc) OnConnState(c *pipe.Conn, s ConnState) { h(c, s) }

// StreamStateHandlerFunc is a function that satisfies ConnStateHandler
type StreamStateHandlerFunc func(*pipe.Stream, StreamState)

// OnStreamState calls the function that underpins StreamStateHandlerFunc
func (h StreamStateHandlerFunc) OnStreamState(s *pipe.Stream, state StreamState) {
	h(s, state)
}
