package inproc

import (
	"context"
	gonet "net"

	"github.com/lthibault/pipewerks/pkg/net"

	"golang.org/x/sync/errgroup"
)

type streamPair struct {
	c context.Context
	net.Edge
	localConn, remoteConn gonet.Conn
}

func newStreamPair(c context.Context, ep net.Edge) (p streamPair) {
	p.c = c
	p.Edge = ep
	p.localConn, p.remoteConn = gonet.Pipe()
	return
}

func (p streamPair) Close() error {
	var g errgroup.Group
	g.Go(p.localConn.Close)
	g.Go(p.remoteConn.Close)
	return g.Wait()
}

func (p streamPair) Local() net.Stream {
	return stream{
		c:    p.c,
		Edge: p.Edge,
		Conn: p.localConn,
	}
}

func (p streamPair) Remote() net.Stream {
	return stream{
		c:    p.c,
		Edge: ep{local: p.Edge.Remote(), remote: p.Edge.Local()},
		Conn: p.remoteConn,
	}
}

type stream struct {
	c context.Context
	net.Edge
	gonet.Conn
}

func (s stream) Context() context.Context { return s.c }
func (s stream) Endpoint() net.Edge       { return s }
