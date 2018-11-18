package inproc

import (
	"context"
	"errors"
	"sync"

	net "github.com/lthibault/pipewerks/pkg/net"
)

// Addr for inproc transport
type Addr string

// Network satisfies net.Addr
func (Addr) Network() string  { return "inproc" }
func (a Addr) String() string { return string(a) }

// NameSpace is an isolated set of connections
type NameSpace interface {
	Bind(Listener) (prev Listener, overwritten bool)
	Connect(context.Context, net.Conn) error
}

type edge struct{ local, remote net.Addr }

func (e edge) Local() net.Addr  { return e.local }
func (e edge) Remote() net.Addr { return e.remote }

// Transport bytes around the process
type Transport struct {
	mu       sync.RWMutex
	ns       NameSpace
	dialback Addr
}

// Listen for incoming connections
func (t *Transport) Listen(c context.Context, a net.Addr) (net.Listener, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if a.Network() != "inproc" {
		return nil, errors.New("invalid network")
	}

	l := Listener{a: Addr(a.String()), ch: make(chan net.Conn)}

	if prev, exists := t.ns.Bind(l); exists {
		t.ns.Bind(prev)
		return nil, errors.New("address in use")
	}

	return l, nil
}

// Dial opens a conneciton
func (t *Transport) Dial(c context.Context, a net.Addr) (conn net.Conn, err error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if a.Network() != "inproc" {
		err = errors.New("invalid network")
		return
	}

	cp := newConnPair(c, t.dialback, a)
	return cp.Local(), t.ns.Connect(c, cp.Remote())
}

// New in-process Transport
func New(opt ...Option) *Transport {
	t := new(Transport)
	t.ns = newMux()

	OptDialback("anonymous")(t)

	for _, o := range opt {
		o(t)
	}
	return t
}
