package inproc

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"

	pipe "github.com/lthibault/pipewerks/pkg"
)

type remoteConnector interface {
	Connect(*stream) error
}

type conn struct {
	o      sync.Once
	ctx    context.Context
	cancel func()

	ch chan *stream
	rc remoteConnector

	clientSide    bool
	idCtr         uint32
	local, remote net.Addr
}

func newConn(c context.Context, laddr, raddr net.Addr) (local *conn, remote *conn) {
	local = new(conn)
	remote = new(conn)

	ctx, cancel := context.WithCancel(c)

	local.ctx = ctx
	local.cancel = cancel
	local.local = laddr
	local.remote = raddr
	local.ch = make(chan *stream)
	local.rc = remote
	local.clientSide = true // needed to set stream id

	remote.ctx = ctx
	remote.cancel = cancel
	remote.local = raddr
	remote.remote = laddr
	remote.ch = make(chan *stream)
	remote.rc = local

	return
}

func (c *conn) Context() context.Context { return c.ctx }

func (c *conn) LocalAddr() net.Addr  { return c.local }
func (c *conn) RemoteAddr() net.Addr { return c.remote }

func (c *conn) AcceptStream() (pipe.Stream, error) {
	select {
	case <-c.ctx.Done():
	case s, ok := <-c.ch:
		if ok {
			return s, nil
		}
	}

	return nil, errors.New("closed")
}

func (c *conn) OpenStream() (pipe.Stream, error) {
	ctx, cancel := context.WithCancel(c.ctx)
	lp, rp := net.Pipe()

	local := new(stream)
	local.ctx = ctx
	local.cancel = cancel
	local.Conn = lp

	remote := new(stream)
	remote.ctx = ctx
	remote.cancel = cancel
	remote.Conn = rp

	// client-initiated streams have even-numbered IDs.
	// server-initiated streams have odd-numbered IDs.
	if i := atomic.AddUint32(&c.idCtr, 1); c.clientSide {
		local.id = i - 1
		remote.id = i
	} else {
		local.id = i
		remote.id = i - 1

	}

	return local, c.rc.Connect(remote)
}

func (c *conn) Connect(s *stream) (err error) {
	select {
	case <-c.ctx.Done():
		err = errors.New("closed")
	case c.ch <- s:
	}

	return
}

func (c *conn) Close() (err error) {
	err = errors.New("already closed")
	c.o.Do(func() {
		c.cancel()
		close(c.ch)
		err = nil
	})
	return
}
