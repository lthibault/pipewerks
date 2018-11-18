package inproc

import (
	"context"
	"io"
	gonet "net"
	"time"

	"github.com/pkg/errors"

	"github.com/lthibault/pipewerks/pkg/net"
	"golang.org/x/sync/errgroup"
)

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

	edg      net.Edge
	l2r, r2l chan net.Stream

	local, remote goconn
}

func newConnPair(c context.Context, local, remote net.Addr) (p *connPair) {
	p = new(connPair)
	p.c, p.cancel = context.WithCancel(c)
	p.edg = edge{local: local, remote: remote}
	p.l2r = make(chan net.Stream)
	p.r2l = make(chan net.Stream)
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
	var in = p.r2l
	var out = p.l2r
	var cxn = p.local
	var edg = p.edg

	if remote {
		cxn = p.remote
		in, out = out, in
		edg = edge{local: edg.Remote(), remote: edg.Local()}
	}

	return conn{
		c:      p.c,
		cancel: p.cancel,
		in:     in,
		out:    out,
		edg:    edg,
		goconn: cxn,
	}
}

type conn struct {
	c      context.Context
	cancel func()

	edg     net.Edge
	in, out chan net.Stream

	goconn
}

func (c conn) Context() context.Context { return c.c }
func (c conn) Endpoint() net.Edge       { return c.edg }
func (c conn) Stream() net.Streamer     { return c }

func (c conn) Close() error {
	c.cancel()
	return nil
}

func (c conn) Open() (s net.Stream, err error) {
	sp := newStreamPair(c.c, c.edg)
	select {
	case c.out <- sp.Remote():
		s = sp.Local()
	case <-c.c.Done():
		err = errors.Wrap(c.c.Err(), "closed")
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
		err = errors.Wrap(c.c.Err(), "closed")
	}
	return
}
