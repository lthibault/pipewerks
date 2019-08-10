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

func (s stream) LocalAddr() net.Addr  { return addrWrapper{s.Conn.LocalAddr()} }
func (s stream) RemoteAddr() net.Addr { return addrWrapper{s.Conn.RemoteAddr()} }

func (s stream) Close() error {
	s.cancel()
	return s.Conn.Close()
}

type addrWrapper struct{ net.Addr }

func (addrWrapper) Network() string { return network }

func (a addrWrapper) String() string {
	if a.Addr.String() == "pipe" {
		return ""
	}

	return a.Addr.String()
}
