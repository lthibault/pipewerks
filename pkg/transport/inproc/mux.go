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
	var v interface{}

	r.RLock()
	v, ok = r.r.Get(path)
	r.RUnlock()

	if ok {
		l = v.(Listener)
	}

	return
}

func (r *radixMux) Bind(l Listener) (ln Listener, ok bool) {
	var v interface{}

	r.Lock()
	v, ok = r.r.Insert(l.Addr().String(), l)
	r.Unlock()

	if ok {
		ln = v.(Listener)
	}

	return
}

func (r *radixMux) DelListener(path string) (l Listener, ok bool) {
	var v interface{}

	r.Lock()
	v, ok = r.r.Delete(path)
	r.Unlock()

	if ok {
		l = v.(Listener)
	}

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
