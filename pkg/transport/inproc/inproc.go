package inproc

import (
	"context"
	"net"
	"sync"

	pipe "github.com/lthibault/pipewerks/pkg"
	"github.com/pkg/errors"
)

const network = "inproc"

// ReverseDialer can provide a dialback address
type ReverseDialer interface {
	Dialback() Addr
}

// Addr for inproc transport
type Addr string

// Network satisfies net.Addr
func (Addr) Network() string  { return network }
func (a Addr) String() string { return string(a) }

// Transport bytes around the process
type Transport struct {
	mu sync.RWMutex
	ns Namespace
}

func (t *Transport) gc(addr string) func() {
	return func() {
		t.mu.Lock()
		t.ns.Free(addr)
		t.mu.Unlock()
	}
}

// Listen inproc
func (t *Transport) Listen(_ context.Context, a net.Addr) (pipe.Listener, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if a.Network() != network {
		return nil, errors.Errorf("inproc: invalid network %s", a.Network())
	}

	l := newListener(Addr(a.String()), t.gc(a.String()))
	if ok := t.ns.Bind(a.String(), l); !ok {
		return nil, errors.Errorf("%s: address in use", a.String())
	}

	return l, nil
}

// Dial inproc
func (t *Transport) Dial(c context.Context, a net.Addr) (pipe.Conn, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if a.Network() != network {
		return nil, errors.Errorf("inproc: invalid network %s", a.Network())
	}

	var laddr Addr
	if r, ok := a.(ReverseDialer); ok {
		laddr = r.Dialback()
	}

	local, remote := newConn(context.Background(), laddr, a)

	l, ok := t.ns.GetConnector(a.String())
	if !ok {
		return nil, errors.New("connection refused")
	}

	if err := l.Connect(c, remote); err != nil {
		return nil, err
	}

	return local, nil
}

// New in-process Transport
func New(opt ...Option) *Transport {
	t := &Transport{ns: DefaultNamespace}

	for _, f := range opt {
		f(t)
	}

	return t
}
