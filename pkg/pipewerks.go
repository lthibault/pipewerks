package net

import (
	"context"

	"github.com/SentimensRG/ctx"
	net "github.com/lthibault/pipewerks/pkg/net"
)

// Dialer can dial a specific address
type Dialer interface {
	Dial(context.Context, net.Addr) (net.Conn, error)
}

// NewDialer for a transport
func NewDialer(t net.Transport) Dialer { return dialListener{t} }

// Listener can listen at a specific address
type Listener interface {
	Listen(context.Context, net.Addr) (net.Listener, error)
}

// NewListener for a transport
func NewListener(t net.Transport) Listener { return dialListener{t} }

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

// ConnHandler does something with a conn.  Any errors encountered during conn
// negotiation are passed to the function.
type ConnHandler func(net.Conn, error)

// HandleConns calls l.Accept() in a loop and calls h in a new goroutine.
// When c expires, the loop exits.
func HandleConns(c context.Context, l net.Listener, h ConnHandler) {
	var conn net.Conn
	var err error
	for range ctx.Tick(c) {
		conn, err = l.Accept(c)

		select {
		case <-c.Done():
		default:
			go h(conn, err)
		}
	}
}

// StreamHandler does something with a stream.  Any errors encountered during
// stream negotiation are passed to the function.
type StreamHandler func(net.Stream, error)

// HandleStreams calles s.Accept() in a loop and calls h in a new goroutine.
// When c expires, the loop exits.
func HandleStreams(c context.Context, s net.Streamer, h StreamHandler) {
	var str net.Stream
	var err error
	for range ctx.Tick(c) {
		str, err = s.Accept()

		select {
		case <-c.Done():
		default:
			go h(str, err)
		}
	}
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
