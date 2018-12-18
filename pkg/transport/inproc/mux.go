package inproc

import (
	"context"
	"net"
	"sync"

	"github.com/pkg/errors"
)

type lmap struct {
	sync.RWMutex
	m map[string]listener
}

func (m *lmap) Set(path string, l listener) {
	m.Lock()
	m.m[path] = l
	m.Unlock()
}

func (m *lmap) Get(path string) (l listener, ok bool) {
	m.RLock()
	l, ok = m.m[path]
	m.RUnlock()
	return
}

func (m *lmap) Rm(path string) {
	m.Lock()
	delete(m.m, path)
	m.Unlock()
}

type mux struct{ m lmap }

func newMux() (m mux) {
	m.m.m = make(map[string]listener)
	return
}

func (x mux) Listen(c context.Context, network, address string) (net.Listener, error) {
	if network != "inproc" {
		return nil, errors.New("invalid network")
	}

	l := listener{
		a:       Addr(address),
		ch:      make(chan net.Conn),
		cq:      make(chan struct{}),
		release: func() { x.m.Rm(address) },
	}

	x.m.Set(address, l)
	return l, nil
}

func (x mux) DialContext(c context.Context, network, addr string) (net.Conn, error) {
	if network != "inproc" {
		return nil, errors.New("invalid network")
	}

	local, remote := net.Pipe()

	l, ok := x.m.Get(addr)
	if !ok {
		return nil, errors.New("connection refused")
	}

	// NOTE: c is the _dial_ context. It is valid for the duration of the Dial
	// 		 operation. The actual connection must be bound to another context.
	o := getDialback(c)

	select {
	case l.ch <- overrideAddrs(remote, Addr(addr), o):
	case <-c.Done():
		return nil, c.Err()
	}

	return overrideAddrs(local, o, Addr(addr)), nil
}

func overrideAddrs(c net.Conn, local, remote net.Addr) addrOverride {
	return addrOverride{
		Conn:   c,
		local:  local,
		remote: remote,
	}
}

type addrOverride struct {
	local, remote net.Addr
	net.Conn
}

func (o addrOverride) LocalAddr() net.Addr  { return o.local }
func (o addrOverride) RemoteAddr() net.Addr { return o.remote }
