package inproc

import (
	"context"
	"errors"

	"github.com/lthibault/pipewerks/pkg/net"
)

// Listener implementation
type Listener struct {
	ch chan net.Conn
	a  Addr
}

// Addr is the listen address
func (l Listener) Addr() net.Addr { return l.a }

// Close the Listener
func (l Listener) Close() (err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("already closed")
		}
	}()
	close(l.ch)
	return
}

// Accept a Connection
func (l Listener) Accept(c context.Context) (net.Conn, error) {
	select {
	case conn, ok := <-l.ch:
		if !ok {
			return nil, errors.New("closed")
		}
		return conn, nil
	case <-c.Done():
		return nil, c.Err()
	}
}
