package inproc

import (
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
type Transport struct{ generic.Transport }

// New in-process Transport
func New(opt ...Option) (t Transport) {
	t.Transport = generic.New()

	OptNameSpace(defaultMux)(&t)

	for _, o := range opt {
		o(&t)
	}
	return
}
