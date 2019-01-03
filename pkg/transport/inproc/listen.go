package inproc

import (
	"errors"
	"net"
	"sync"
)

type listener struct {
	o       sync.Once
	cq      chan struct{}
	ch      chan net.Conn
	a       Addr
	release func()
}

func newListener(a Addr, gc func()) *listener {
	return &listener{
		a:       a,
		ch:      make(chan net.Conn),
		cq:      make(chan struct{}),
		release: gc,
	}
}

func (l *listener) Addr() net.Addr { return l.a }

func (l *listener) Close() (err error) {
	err = errors.New("already closed")

	l.o.Do(func() {
		close(l.cq)
		close(l.ch)
		l.release()
		err = nil
	})

	return
}

func (l *listener) Accept() (net.Conn, error) {
	select {
	case <-l.cq:
		break
	default:
		select {
		case conn := <-l.ch:
			return conn, nil
		case <-l.cq:
			break
		}
	}

	return nil, errors.New("closed")
}
