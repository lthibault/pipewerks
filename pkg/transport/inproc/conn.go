package inproc

import (
	"context"
	"errors"
	gonet "net"
	"sync/atomic"
	"unsafe"

	"github.com/lthibault/pipewerks/pkg/net"
	"golang.org/x/sync/errgroup"
)

type atomicErr atomic.Value

func newAtomicErr() *atomicErr { return (*atomicErr)(new(atomic.Value)) }

func (e *atomicErr) Load() (err error) {
	return (*atomic.Value)(unsafe.Pointer(e)).Load().(error)
}

func (e *atomicErr) Store(err error) {
	(*atomic.Value)(unsafe.Pointer(e)).Store(err)
}

type connPair struct {
	c      context.Context
	cancel func()

	ep net.EndpointPair

	local, remote gonet.Conn
	lerr, rerr    atomicErr
}

func newConnPair(c context.Context, local, remote net.Addr) (p *connPair) {
	p = new(connPair)
	p.c, p.cancel = context.WithCancel(c)
	p.ep = ep{local: local, remote: remote}
	p.local, p.remote = gonet.Pipe()
	return
}

func (p *connPair) Close() error {
	defer p.cancel()
	var g errgroup.Group
	g.Go(p.local.Close)
	g.Go(p.remote.Close)
	return g.Wait()
}

func (p *connPair) Local() net.Conn  { return p.newConn(p.local, &p.lerr, &p.rerr) }
func (p *connPair) Remote() net.Conn { return p.newConn(p.remote, &p.rerr, &p.lerr) }

func (p *connPair) newConn(cxn gonet.Conn, lerr, rerr *atomicErr) conn {
	return conn{
		c:      p.c,
		cancel: p.cancel,
		in:     make(chan net.Stream),
		out:    make(chan net.Stream),
		ep:     p.ep,
		lerr:   lerr,
		rerr:   rerr,
		Conn:   cxn,
	}
}

type conn struct {
	c      context.Context
	cancel func()

	ep      net.EndpointPair
	in, out chan net.Stream

	lerr, rerr *atomicErr
	gonet.Conn
}

func (c conn) Context() context.Context   { return c.c }
func (c conn) Endpoint() net.EndpointPair { return c.ep }
func (c conn) Stream() net.Streamer       { return c }

func (c conn) Close() error { return c.CloseWithError(0, nil) }

func (c conn) CloseWithError(_ net.ErrorCode, err error) error {
	select {
	case <-c.c.Done():
		if err = c.lerr.Load(); err == nil {
			err = c.c.Err()
		}
	default:
		if err == nil {
			err = errors.New("closed")
		}

		c.rerr.Store(err)
		c.cancel()

		err = c.Conn.Close()
	}

	return err
}

func (c conn) Open() (s net.Stream, err error) {
	sp := newStreampair(c.c, c.ep)
	select {
	case c.out <- sp.Remote():
		s = sp.Local()
	case <-c.c.Done():
		if err = c.lerr.Load(); err == nil {
			err = c.c.Err()
		}
	}
	return
}

func (c conn) Accept() (s net.Stream, err error) {
	var ok bool
	select {
	case s, ok = <-c.in:
		if !ok {
			err = errors.New("closed")
		}
	case <-c.c.Done():
		if err = c.lerr.Load(); err == nil {
			err = c.c.Err()
		}
	}
	return
}
