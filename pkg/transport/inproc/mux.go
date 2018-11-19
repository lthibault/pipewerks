package inproc

import (
	"context"
	"errors"
	gonet "net"
	"sync"

	radix "github.com/armon/go-radix"
)

type radixMux struct {
	sync.RWMutex
	r *radix.Tree
}

func newMux() *radixMux { return &radixMux{r: radix.New()} }

func (r *radixMux) GetListener(path string) (l listener, ok bool) {
	var v interface{}

	r.RLock()
	v, ok = r.r.Get(path)
	r.RUnlock()

	if ok {
		l = v.(listener)
	}

	return
}

func (r *radixMux) Listen(c context.Context, network, address string) (gonet.Listener, error) {
	if network != "inproc" {
		return nil, errors.New("invalid network")
	}

	l := listener{
		a:  Addr(address),
		ch: make(chan gonet.Conn),
		cq: make(chan struct{}),
	}

	r.Lock()
	defer r.Unlock()

	if v, ok := r.r.Insert(address, l); ok {
		r.r.Insert(address, v)
		return nil, errors.New("address already bound")
	}

	return l, nil
}

func (r *radixMux) DelListener(path string) (l listener, ok bool) {
	var v interface{}

	r.Lock()
	v, ok = r.r.Delete(path)
	r.Unlock()

	if ok {
		l = v.(listener)
	}

	return
}

func (r *radixMux) DialContext(c context.Context, network, address string) (gonet.Conn, error) {
	if network != "inproc" {
		return nil, errors.New("invalid network")
	}

	local, remote := gonet.Pipe()

	r.RLock()
	defer r.RUnlock()

	l, ok := r.GetListener(address)
	if ok {
		return nil, errors.New("connection refused")
	}

	select {
	case l.ch <- remote:
	case <-c.Done():
		return nil, c.Err()
	}

	return local, nil
}
