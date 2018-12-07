package inproc

import (
	"context"
	"net"
	"sync"

	radix "github.com/armon/go-radix"
	"github.com/pkg/errors"
)

type radixMux struct {
	sync.RWMutex
	r *radix.Tree
}

func newMux() *radixMux { return &radixMux{r: radix.New()} }

func (r *radixMux) GetListener(path string) (l listener, ok bool) {
	r.RLock()
	defer r.RUnlock()

	var v interface{}
	if v, ok = r.r.Get(path); ok {
		l = v.(listener)
	}

	return
}

func (r *radixMux) Listen(c context.Context, network, address string) (net.Listener, error) {
	r.Lock()
	defer r.Unlock()

	if network != "inproc" {
		return nil, errors.New("invalid network")
	}

	l := listener{
		a:      Addr(address),
		ch:     make(chan net.Conn),
		cq:     make(chan struct{}),
		unbind: r.Unbind,
	}

	if v, ok := r.r.Insert(address, l); ok {
		r.r.Insert(address, v)
		return nil, errors.New("address already bound")
	}

	return l, nil
}

func (r *radixMux) Unbind(path string) {
	r.Lock()
	r.r.Delete(path)
	r.Unlock()
}

func (r *radixMux) DialContext(c context.Context, network, addr string) (net.Conn, error) {
	if network != "inproc" {
		return nil, errors.New("invalid network")
	}

	local, remote := net.Pipe()

	l, ok := r.GetListener(addr)
	if !ok {
		return nil, errors.New("connection refused")
	}

	// NOTE: c is the _dial_ context. It is valid for the duration of the Dial
	// 		 operation. The actual connection must be bound to another context.
	o := getDialback(context.Background())

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
