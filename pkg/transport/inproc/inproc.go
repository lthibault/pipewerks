package inproc

import (
	"context"
	"net"
	"sync"

	pipe "github.com/lthibault/pipewerks/pkg"
	"github.com/pkg/errors"
)

// ReverseDialer can provide a dialback address
type ReverseDialer interface {
	Dialback() Addr
}

// Addr for inproc transport
type Addr string

// Network satisfies net.Addr
func (Addr) Network() string  { return "inproc" }
func (a Addr) String() string { return string(a) }

// Transport bytes around the process
type Transport struct {
	sync.RWMutex
	m map[string]*listener
}

func (t *Transport) gc(addr string) func() {
	return func() {
		t.Lock()
		delete(t.m, addr)
		t.Unlock()
	}
}

// Listen inproc
func (t *Transport) Listen(c context.Context, a net.Addr) (pipe.Listener, error) {
	t.Lock()
	defer t.Unlock()

	if a.Network() != "inproc" {
		return nil, errors.Errorf("inproc: invalid network %s", a.Network())
	}

	l := newListener(Addr(a.String()), t.gc(a.String()))

	t.m[a.String()] = l
	return l, nil
}

// Dial inproc
func (t *Transport) Dial(c context.Context, a net.Addr) (pipe.Conn, error) {
	t.RLock()
	defer t.RUnlock()

	if a.Network() != "inproc" {
		return nil, errors.Errorf("inproc: invalid network %s", a.Network())
	}

	var laddr Addr
	if r, ok := a.(ReverseDialer); ok {
		laddr = r.Dialback()
	}

	local, remote := newConn(context.Background(), laddr, a)

	l, ok := t.m[a.String()]
	if !ok {
		return nil, errors.New("connection refused")
	}

	if err := l.connect(c, remote); err != nil {
		return nil, err
	}

	return local, nil
}

// New in-process Transport
func New() *Transport { return &Transport{m: make(map[string]*listener)} }
