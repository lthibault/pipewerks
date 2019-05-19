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

type streamPool interface {
	CloseAll() error
	Get(pipe.Conn) (pipe.Stream, bool)
	Put(pipe.Stream)
}

func newStreamPool(cap int) streamPool {
	panic("function NOT IMPLEMENTED")
}
