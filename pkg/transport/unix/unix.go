package unix

import (
	"context"
	"net"

	pipe "github.com/lthibault/pipewerks/pkg"
	"github.com/lthibault/pipewerks/pkg/transport/generic"
	"github.com/pkg/errors"
)

// Transport over Unix domain socket
type Transport struct{ generic.Transport }

// Listen Unix
func (t Transport) Listen(c context.Context, a net.Addr) (pipe.Listener, error) {
	if a.Network() != "unix" {
		return nil, errors.Errorf("unix: invalid network %s", a.Network())
	}

	return t.Transport.Listen(c, a)
}

// Dial Unix
func (t Transport) Dial(c context.Context, a net.Addr) (pipe.Conn, error) {
	if a.Network() != "unix" {
		return nil, errors.Errorf("tcp: invalid network %s", a.Network())
	}

	return t.Transport.Dial(c, a)
}

// New Unix Transport
func New(opt ...Option) (t Transport) {
	t.Transport = generic.New()
	t.Transport.NetDialer = new(net.Dialer)
	t.Transport.NetListener = new(net.ListenConfig)

	for _, fn := range opt {
		fn(&t)
	}

	return t
}
