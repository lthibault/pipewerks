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
	serverMuxAdapter
	net.Listener
}

func (l listener) Accept() (pipe.Conn, error) {
	raw, err := l.Listener.Accept()
	if err != nil {
		return nil, errors.Wrap(err, "listener")
	}

	conn, err := l.AdaptServer(raw)
	if err != nil {
		raw.Close()
		return nil, errors.Wrap(err, "mux")
	}

	return conn, nil
}

type addresser interface {
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

type edge struct{ addresser }

func (e edge) Local() net.Addr  { return e.LocalAddr() }
func (e edge) Remote() net.Addr { return e.RemoteAddr() }

type connection struct{ *yamux.Session }

func (c connection) Context() context.Context {
	return ctx.AsContext(ctx.C(c.CloseChan()))
}

func (c connection) Endpoint() pipe.Edge   { return edge{c} }
func (c connection) Stream() pipe.Streamer { return c }

func (c connection) Open() (pipe.Stream, error) {
	s, err := c.OpenStream()
	x, cancel := context.WithCancel(c.Context())
	return stream{c: x, cancel: cancel, Stream: s}, err
}

func (c connection) Accept() (pipe.Stream, error) {
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
	MuxAdapter
	NetListener
	NetDialer
}

// Listen Generic
func (t Transport) Listen(c context.Context, a net.Addr) (pipe.Listener, error) {
	l, err := t.NetListener.Listen(c, a.Network(), a.String())
	return listener{Listener: l, serverMuxAdapter: t.MuxAdapter}, err
}

// Dial Generic
func (t Transport) Dial(c context.Context, a net.Addr) (pipe.Conn, error) {
	conn, err := t.NetDialer.DialContext(c, a.Network(), a.String())
	if err != nil {
		return nil, errors.Wrap(err, "dial")
	}

	return t.AdaptClient(conn)
}

// MuxConfig is a MuxAdapter that uses github.com/hashicorp/yamux
type MuxConfig struct{ *yamux.Config }

// AdaptServer is called by the listener
func (c MuxConfig) AdaptServer(conn net.Conn) (pipe.Conn, error) {
	sess, err := yamux.Server(conn, c.Config)
	return connection{Session: sess}, errors.Wrap(err, "yamux")
}

// AdaptClient is called by the dialer
func (c MuxConfig) AdaptClient(conn net.Conn) (pipe.Conn, error) {
	sess, err := yamux.Client(conn, c.Config)
	return connection{Session: sess}, errors.Wrap(err, "yamux")
}

// New Generic Transport
func New(opt ...Option) *Transport {
	t := new(Transport)

	// defaults
	t.MuxAdapter = MuxConfig{}

	// overrides
	for _, fn := range opt {
		fn(t)
	}
	return t
}
