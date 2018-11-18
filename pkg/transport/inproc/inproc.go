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

type ep struct{ local, remote net.Addr }

func (ep ep) Local() net.Addr  { return ep.local }
func (ep ep) Remote() net.Addr { return ep.remote }

type listener struct {
	ch chan net.Conn
	a  net.Addr
}

func (l listener) Addr() net.Addr { return l.a }

func (l listener) Close() (err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("already closed")
		}
	}()
	close(l.ch)
	return
}

func (l listener) Accept(c context.Context) (net.Conn, error) {
	select {
	case conn, ok := <-l.ch:
		if !ok {
			return nil, errors.New("closed")
		}
		return conn, nil
	case <-c.Done():
		return nil, c.Err()
	}
}

// Transport bytes around the process
type Transport struct {
	mu       sync.RWMutex
	mux      radixMux
	dialback Addr
}

// Listen for incoming connections
func (t *Transport) Listen(c context.Context, a net.Addr) (net.Listener, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if a.Network() != "inproc" {
		return nil, errors.New("invalid network")
	}

	l := listener{a: a, ch: make(chan net.Conn)}

	if prev, exists := t.mux.SetListener(l, a.String()); exists {
		t.mux.SetListener(prev.listener, a.String())
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
	return cp.Local(), t.mux.ServeConn(c, cp.Remote())
}

// New in-process Transport
func New(opt ...Option) *Transport {
	t := new(Transport)
	t.mux = newMux()

	OptDialback("anonymous")(t)

	for _, o := range opt {
		o(t)
	}
	return t
}
