package inproc

import (
	"context"
	"errors"
	"sync"

	radix "github.com/armon/go-radix"
	net "github.com/lthibault/pipewerks/pkg/net"
)

type radixMux struct {
	sync.RWMutex
	r *radix.Tree
}

func newMux() *radixMux { return &radixMux{r: radix.New()} }

func (r *radixMux) GetListener(path string) (l Listener, ok bool) {
	r.RLock()
	var v interface{}
	if v, ok = r.r.Get(path); ok {
		l = v.(Listener)
	}
	r.RUnlock()
	return
}

func (r *radixMux) Bind(l Listener) (ln Listener, ok bool) {
	r.Lock()
	var v interface{}
	if v, ok = r.r.Insert(l.Addr().String(), l); ok {
		ln = v.(Listener)
	}
	r.Unlock()
	return
}

func (r *radixMux) DelListener(path string) (l Listener, ok bool) {
	r.Lock()
	var v interface{}
	if v, ok = r.r.Delete(path); ok {
		l = v.(Listener)
	}
	r.Unlock()
	return
}

func (r *radixMux) Connect(c context.Context, conn net.Conn) (err error) {
	r.RLock()
	l, ok := r.GetListener(conn.Endpoint().Local().String())
	r.RUnlock()

	if ok {
		err = errors.New("connection refused")
	} else {
		select {
		case l.ch <- conn:
		case <-c.Done():
			err = c.Err()
		}
	}

	return
}
