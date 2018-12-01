package tcp

import (
	"context"
	"net"

	pipe "github.com/lthibault/pipewerks/pkg"
	"github.com/lthibault/pipewerks/pkg/transport/generic"
	"github.com/pkg/errors"
)

// Transport over TCP
type Transport struct{ generic.Transport }

// Listen TCP
func (t Transport) Listen(c context.Context, a net.Addr) (pipe.Listener, error) {
	if a.Network() != "tcp" {
		return nil, errors.New("invalid network")
	}

	return t.Transport.Listen(c, a)
}

// Dial TCP
func (t Transport) Dial(c context.Context, a net.Addr) (pipe.Conn, error) {
	if a.Network() != "tcp" {
		return nil, errors.New("invalid network")
	}

	return t.Transport.Dial(c, a)
}

// New TCP Transport
func New(opt ...Option) (t Transport) {
	t.Transport = generic.New()
	for _, fn := range opt {
		fn(&t)
	}
	return t
}
