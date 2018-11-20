package net

import (
	"context"

	net "github.com/lthibault/pipewerks/pkg/net"
)

// Client is a Dialer that knows where to dial
type Client interface {
	Addr() net.Addr
	Dial(context.Context) (net.Conn, error)
}

// NewClient that connects to a over t
func NewClient(t net.Transport, a net.Addr) Client {
	return peer{dialListener: dialListener{t}, a: a}
}

// Server is a Listener that knows where to listen
type Server interface {
	Addr() net.Addr
	Listen(context.Context) (net.Listener, error)
}

// NewServer that listens on a over t
func NewServer(t net.Transport, a net.Addr) Server {
	return peer{dialListener: dialListener{t}, a: a}
}

type dialListener struct{ net.Transport }

func (d dialListener) Dial(c context.Context, a net.Addr) (net.Conn, error) {
	return d.Transport.Dial(c, a)
}

func (d dialListener) Listen(c context.Context, a net.Addr) (net.Listener, error) {
	return d.Transport.Listen(c, a)
}

type peer struct {
	a net.Addr
	dialListener
}

func (p peer) Addr() net.Addr                                 { return p.a }
func (p peer) Dial(c context.Context) (net.Conn, error)       { return p.dialListener.Dial(c, p.a) }
func (p peer) Listen(c context.Context) (net.Listener, error) { return p.dialListener.Listen(c, p.a) }
