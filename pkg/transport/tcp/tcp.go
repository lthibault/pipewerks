package tcp

import (
	"context"
	"net"

	pipe "github.com/lthibault/pipewerks/pkg"
	"github.com/lthibault/pipewerks/pkg/transport/generic"
	"github.com/pkg/errors"
)

func checkNetwork(a net.Addr) (ok bool) {
	switch a.Network() {
	case "tcp", "tcp4", "tcp6":
		ok = true
	}

	return
}

// Transport over TCP
type Transport struct{ generic.Transport }

// Listen TCP
func (t Transport) Listen(c context.Context, a net.Addr) (pipe.Listener, error) {
	if !checkNetwork(a) {
		return nil, errors.Errorf("tcp: invalid network %s", a.Network())
	}

	return t.Transport.Listen(c, a)
}

// Dial TCP
func (t Transport) Dial(c context.Context, a net.Addr) (pipe.Conn, error) {
	if !checkNetwork(a) {
		return nil, errors.Errorf("tcp: invalid network %s", a.Network())
	}

	return t.Transport.Dial(c, a)
}

// New TCP Transport
func New(opt ...Option) (t Transport) {
	t.Transport = generic.New()
	t.Transport.NetDialer = new(net.Dialer)
	t.Transport.NetListener = new(net.ListenConfig)

	for _, fn := range opt {
		fn(&t)
	}

	return t
}
