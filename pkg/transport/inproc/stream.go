package inproc

import (
	"context"
	"net"
)

type stream struct {
	ctx    context.Context
	cancel func()

	id uint32
	net.Conn
}

func (s stream) Context() context.Context { return s.ctx }
func (s stream) StreamID() uint32         { return s.id }

func (s stream) Close() error {
	s.cancel()
	return s.Conn.Close()
}
