package inproc

import (
	"context"
	"errors"
	"net"
	"sync"

	pipe "github.com/lthibault/pipewerks/pkg"
)

type listener struct {
	o       sync.Once
	cq      chan struct{}
	ch      chan pipe.Conn
	a       Addr
	release func()
}

func newListener(a Addr, gc func()) *listener {
	return &listener{
		a:       a,
		ch:      make(chan pipe.Conn),
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

func (l *listener) Accept() (pipe.Conn, error) {
	select {
	case <-l.cq:
	case conn, ok := <-l.ch:
		if ok {
			return conn, nil
		}
	}

	return nil, errors.New("closed")
}

func (l *listener) Connect(c context.Context, conn pipe.Conn) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("closed")
		}
	}()

	select {
	case <-c.Done():
		err = c.Err()
	case <-l.cq:
		err = errors.New("connection refused")
	case l.ch <- conn:
	}

	return
}
