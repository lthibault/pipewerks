package inproc

import (
	"context"
	"net"
	"sync"

	"github.com/lthibault/pipewerks/pkg/transport/generic"
	"github.com/pkg/errors"
)

// NameSpace is an address space that encapusulates a set of connections.
type NameSpace interface {
	generic.NetListener
	generic.NetDialer
}

type mux struct {
	sync.RWMutex
	m map[string]*listener
}

func newMux() (m mux) {
	m.m = make(map[string]*listener)
	return
}

func (x *mux) gc(addr string) func() {
	return func() {
		x.Lock()
		delete(x.m, addr)
		x.Unlock()
	}
}

func (x *mux) Listen(c context.Context, network, address string) (net.Listener, error) {
	if network != netInproc {
		return nil, errors.Errorf("%s: invalid network %s", netInproc, network)
	}

	l := newListener(Addr(address), x.gc(address))

	x.m[address] = l
	return l, nil
}

func (x *mux) DialContext(c context.Context, network, addr string) (net.Conn, error) {
	if network != netInproc {
		return nil, errors.Errorf("%s: invalid network %s", netInproc, network)
	}

	local, remote := net.Pipe()
	x.RLock()
	defer x.RUnlock()

	l, ok := x.m[addr]
	if !ok {
		return nil, errors.New("connection refused")
	}

	o := getDialback(c)
	if err := l.connect(c, overrideAddrs(remote, Addr(addr), o)); err != nil {
		return nil, err
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
