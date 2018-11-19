package inproc

import (
	"errors"
	gonet "net"

	"github.com/lthibault/pipewerks/pkg/net"
)

type listener struct {
	cq chan struct{}
	ch chan gonet.Conn
	a  Addr
}

func (l listener) Addr() net.Addr { return l.a }

func (l listener) Close() (err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("already closed")
		}
	}()
	close(l.cq)
	close(l.ch)
	return
}

func (l listener) Accept() (gonet.Conn, error) {
	select {
	case conn, ok := <-l.ch:
		if !ok {
			return nil, errors.New("closed")
		}
		return conn, nil
	case <-l.cq:
		return nil, errors.New("closed")
	}
}
