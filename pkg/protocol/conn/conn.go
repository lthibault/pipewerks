package conn

import (
	"context"

	"github.com/SentimensRG/ctx/mergectx"
	net "github.com/lthibault/pipewerks/pkg"
	"golang.org/x/sync/errgroup"
)

// Protocol can bind two conns
type Protocol interface {
	StartAccepting(left, right net.Conn) func() error
	StartOpening(left, right net.Conn) func() error
}

// BindConn ...
func BindConn(p Protocol, left, right net.Conn) net.Joint {
	c := mergectx.Link(left.Context(), right.Context())
	g, c := errgroup.WithContext(c)
	c, cancel := context.WithCancel(c)

	g.Go(p.StartAccepting(left, right))
	g.Go(p.StartAccepting(right, left))
	g.Go(p.StartOpening(left, right))
	g.Go(p.StartOpening(right, left))

	return net.Joint{Break: cancel, Wait: g.Wait}
}
