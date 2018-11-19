package inproc

import (
	net "github.com/lthibault/pipewerks/pkg/net"
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

type edge struct{ local, remote net.Addr }

func (e edge) Local() net.Addr  { return e.local }
func (e edge) Remote() net.Addr { return e.remote }

// Transport bytes around the process
type Transport struct {
	dialback Addr
	*generic.Transport
}

// New in-process Transport
func New(opt ...Option) *Transport {
	t := new(Transport)
	OptDialback("anonymous")(t)
	OptAddrSpace(defaultMux)(t)

	for _, o := range opt {
		o(t)
	}
	return t
}
