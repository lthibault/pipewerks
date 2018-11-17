package inproc

import (
	"context"

	net "github.com/lthibault/pipewerks/pkg/net"
)

// Transport bytes around the process
type Transport struct{}

// Listen for incoming connections
func (t Transport) Listen(c context.Context, a net.Addr) (net.Listener, error) {
	panic("Listen NOT IMPLEMENTED")
}

// Dial opens a conneciton
func (t Transport) Dial(c context.Context, a net.Addr) (net.Conn, error) {
	panic("Dial NOT IMPLEMENTED")
}

// New in-process Transport
func New(opt ...Option) *Transport {
	t := new(Transport)
	for _, o := range opt {
		o(t)
	}
	return t
}
