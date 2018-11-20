package kcp

import (
	"context"

	net "github.com/lthibault/pipewerks/pkg"
	"github.com/lthibault/pipewerks/pkg/transport/generic"
	"github.com/pkg/errors"
)

// Option for KCP
type Option = generic.Option

// Transport over KCP
type Transport struct{ *generic.Transport }

// Listen KCP
func (t Transport) Listen(c context.Context, a net.Addr) (net.Listener, error) {
	if a.Network() != "kcp" {
		return nil, errors.New("invalid network")
	}

	return t.Transport.Listen(c, a)
}

// Dial KCP
func (t Transport) Dial(c context.Context, a net.Addr) (net.Conn, error) {
	if a.Network() != "kcp" {
		return nil, errors.New("invalid network")
	}

	return t.Transport.Dial(c, a)
}

// New KCP Transport
func New(opt ...Option) (t Transport) {
	t.Transport = generic.New(opt...)
	return t
}
