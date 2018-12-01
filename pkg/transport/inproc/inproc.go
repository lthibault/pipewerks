package inproc

import (
	"context"
	"net"

	pipe "github.com/lthibault/pipewerks/pkg"
	"github.com/lthibault/pipewerks/pkg/transport/generic"
)

// Addr for inproc transport
type Addr string

// Network satisfies net.Addr
func (Addr) Network() string  { return "inproc" }
func (a Addr) String() string { return string(a) }

// NameSpace is an isolated set of connections
type NameSpace interface {
	generic.NetListener
	generic.NetDialer
}

// Transport bytes around the process
type Transport struct {
	dialback Addr
	generic.Transport
}

// Dial sets the dialback addr before dialing into the connection.
func (t Transport) Dial(c context.Context, a net.Addr) (pipe.Conn, error) {
	if t.dialback != "" {
		c = setDialback(c, t.dialback)
	}
	return t.Transport.Dial(c, a)
}

// New in-process Transport
func New(opt ...Option) (t *Transport) {
	t = new(Transport)
	t.Transport = generic.New()

	OptDialback(Addr("anonymous"))(t)
	OptAddrSpace(defaultMux)(t)

	for _, o := range opt {
		o(t)
	}
	return
}
