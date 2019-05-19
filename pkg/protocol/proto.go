package protocol

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
		return "open"
	case ConnStateClosed:
		return "closed"
	}

	panic("unreachable")
}

// StreamState tracks the state of a pipe.Stream
type StreamState uint8

func (s StreamState) String() string {
	switch s {
	case StreamStateOpen:
		return "open"
	case StreamStateIdle:
		return "idle"
	case StreamStateClosed:
		return "closed"
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
