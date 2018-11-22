package generic

import (
	"context"
	"net"

	"github.com/SentimensRG/ctx"
	pipe "github.com/lthibault/pipewerks/pkg"

	"github.com/hashicorp/yamux"
	"github.com/pkg/errors"
)

type listener struct {
	c *MuxConfig
	net.Listener
}

func (l listener) Accept(c context.Context) (cxn pipe.Conn, err error) {
	var sess *yamux.Session
	ch := make(chan struct{})

	go func() {
		var conn net.Conn
		if conn, err = l.Listener.Accept(); err != nil {
			err = errors.Wrap(err, "listener")
		} else if sess, err = yamux.Server(conn, l.c); err != nil {
			err = errors.Wrap(err, "mux")
		}
		close(ch)
	}()

	select {
	case <-c.Done():
		err = c.Err()
	case <-ch:
		cxn = conn{sess}
	}

	return
}

type addresser interface {
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

type edge struct{ addresser }

func (e edge) Local() net.Addr  { return e.LocalAddr() }
func (e edge) Remote() net.Addr { return e.RemoteAddr() }

type conn struct{ *yamux.Session }

func (c conn) Context() context.Context {
	return ctx.AsContext(ctx.C(c.CloseChan()))
}

func (c conn) Endpoint() pipe.Edge   { return edge{c} }
func (c conn) Stream() pipe.Streamer { return c }

func (c conn) Open() (pipe.Stream, error) {
	s, err := c.OpenStream()
	x, cancel := context.WithCancel(c.Context())
	return stream{c: x, cancel: cancel, Stream: s}, err
}

func (c conn) Accept() (pipe.Stream, error) {
	s, err := c.AcceptStream()
	x, cancel := context.WithCancel(c.Context())
	return stream{c: x, cancel: cancel, Stream: s}, err
}

type stream struct {
	c      context.Context
	cancel func()
	*yamux.Stream
}

func (s stream) Context() context.Context { return s.c }
func (s stream) Endpoint() pipe.Edge      { return edge{s} }
func (s stream) Close() error {
	s.cancel()
	return s.Stream.Close()
}

// Transport for any pipe.Conn
type Transport struct {
	*MuxConfig
	NetListener
	NetDialer
}

// Listen Generic
func (t Transport) Listen(c context.Context, a net.Addr) (pipe.Listener, error) {
	l, err := t.NetListener.Listen(c, a.Network(), a.String())
	return listener{Listener: l, c: t.MuxConfig}, err
}

// Dial Generic
func (t Transport) Dial(c context.Context, a net.Addr) (pipe.Conn, error) {
	cxn, err := t.NetDialer.DialContext(c, a.Network(), a.String())
	if err != nil {
		return nil, errors.Wrap(err, "dial")
	}

	sess, err := yamux.Client(cxn, t.MuxConfig)
	return conn{sess}, errors.Wrap(err, "mux")
}

// New Generic Transport
func New(opt ...Option) *Transport {
	t := new(Transport)
	for _, fn := range opt {
		fn(t)
	}
	return t
}
