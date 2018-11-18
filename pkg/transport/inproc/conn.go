package inproc

import (
	"context"
	"io"
	gonet "net"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/lthibault/pipewerks/pkg/net"
	"golang.org/x/sync/errgroup"
)

type atomicErr atomic.Value

func newAtomicErr() *atomicErr { return (*atomicErr)(new(atomic.Value)) }

func (e *atomicErr) Load() (err error) {
	if v := (*atomic.Value)(unsafe.Pointer(e)).Load(); v != nil {
		err = v.(error)
	}
	return
}

func (e *atomicErr) Store(err error) {
	(*atomic.Value)(unsafe.Pointer(e)).Store(err)
}

// goconn allows us to use some net.go niceties
type goconn interface {
	io.ReadWriteCloser
	SetDeadline(t time.Time) error
	SetReadDeadline(t time.Time) error
	SetWriteDeadline(t time.Time) error
}

type connPair struct {
	c      context.Context
	cancel func()

	ep net.Edge

	local, remote goconn
	lae, rae      atomicErr
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

func (p *connPair) Local() net.Conn  { return p.newConn(false) }
func (p *connPair) Remote() net.Conn { return p.newConn(true) }

func (p *connPair) newConn(remote bool) conn {
	var cxn = p.local
	var lae = &p.lae
	var rae = &p.rae
	var edg = p.ep

	if remote {
		cxn = p.remote
		lae, rae = rae, lae
		edg = ep{local: edg.Remote(), remote: edg.Local()}
	}

	return conn{
		c:      p.c,
		cancel: p.cancel,
		in:     make(chan net.Stream),
		out:    make(chan net.Stream),
		ep:     edg,
		lae:    lae,
		rae:    rae,
		goconn: cxn,
	}
}

type conn struct {
	c      context.Context
	cancel func()

	ep      net.Edge
	in, out chan net.Stream

	lae, rae *atomicErr
	goconn
}

func (c conn) Context() context.Context { return c.c }
func (c conn) Endpoint() net.Edge       { return c.ep }
func (c conn) Stream() net.Streamer     { return c }

func (c conn) Close() error { return c.CloseWithError(0, nil) }

func (c conn) CloseWithError(_ net.ErrorCode, err error) error {
	select {
	case <-c.c.Done():
		if err = c.lae.Load(); err == nil {
			err = c.c.Err()
		}
	default:
		if err == nil {
			err = errors.New("closed")
		}

		c.rae.Store(err)
		c.cancel()

		err = c.goconn.Close()
	}

	return err
}

func (c conn) Open() (s net.Stream, err error) {
	sp := newStreamPair(c.c, c.ep)
	select {
	case c.out <- sp.Remote():
		s = sp.Local()
	case <-c.c.Done():
		if err = c.lae.Load(); err == nil {
			err = errors.Wrap(c.c.Err(), "closed")
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
		if err = c.lae.Load(); err == nil {
			err = errors.Wrap(c.c.Err(), "closed")
		}
	}
	return
}
