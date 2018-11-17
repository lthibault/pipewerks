package stream

import (
	"context"

	"github.com/SentimensRG/ctx/mergectx"
	net "github.com/lthibault/pipewerks/pkg/net"
	"golang.org/x/sync/errgroup"
)

// Protocol can bind two conns
type Protocol interface {
	StartAccepting(left, right net.Stream) func() error
	StartOpening(left, right net.Stream) func() error
}

// Bind Streams
func Bind(p Protocol, left, right net.Stream) net.Joint {
	c := mergectx.Link(left.Context(), right.Context())
	g, c := errgroup.WithContext(c)
	c, cancel := context.WithCancel(c)

	g.Go(p.StartAccepting(left, right))
	g.Go(p.StartAccepting(right, left))
	g.Go(p.StartOpening(left, right))
	g.Go(p.StartOpening(right, left))

	return net.Joint{Break: cancel, Wait: g.Wait}
}
