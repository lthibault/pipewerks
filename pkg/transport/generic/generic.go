package generic

import (
	"context"
	"net"

	"github.com/SentimensRG/ctx"
	pipe "github.com/lthibault/pipewerks/pkg"

	"github.com/hashicorp/yamux"
	"github.com/pkg/errors"
)

// ConnectHook is invoked when the Transport successfully opens a raw
// net.Conn. It allows user-defined logic to run on the raw connection before
// the stream muxer starts.
type ConnectHook interface {
	// OnDialConnected is called when a dialer successfully connects to a remote
	// listener.  The context is the same as the one passed to Dial(), and may
	// be expired.  Users SHOULD NOT abort a connection due to an expired context.
	OnDialConnected(context.Context, net.Conn) (net.Conn, error)
	OnListenConnected(net.Conn) (net.Conn, error)
}

type noopConnect struct{}

func (noopConnect) OnDialConnected(_ context.Context, conn net.Conn) (net.Conn, error) {
	return conn, nil
}

func (noopConnect) OnListenConnected(conn net.Conn) (net.Conn, error) {
	return conn, nil
}

type listener struct {
	h ConnectHook
	serverMuxAdapter
	net.Listener
}

func (l listener) Accept() (pipe.Conn, error) {
	raw, err := l.Listener.Accept()
	if err != nil {
		return nil, errors.Wrap(err, "listener")
	}

	if raw, err = l.h.OnListenConnected(raw); err != nil {
		return nil, err
	}

	conn, err := l.AdaptServer(raw)
	if err != nil {
		raw.Close()
		return nil, errors.Wrap(err, "mux")
	}

	return conn, nil
}

type connection struct{ *yamux.Session }

func (c connection) Context() context.Context {
	return ctx.AsContext(ctx.C(c.CloseChan()))
}

func (c connection) OpenStream() (pipe.Stream, error) {
	s, err := c.Session.OpenStream()
	x, cancel := context.WithCancel(c.Context())
	return stream{c: x, cancel: cancel, Stream: s}, err
}

func (c connection) AcceptStream() (pipe.Stream, error) {
	s, err := c.Session.AcceptStream()
	x, cancel := context.WithCancel(c.Context())
	return stream{c: x, cancel: cancel, Stream: s}, err
}

type stream struct {
	c      context.Context
	cancel func()
	*yamux.Stream
}

func (s stream) Context() context.Context { return s.c }
func (s stream) Close() error {
	s.cancel()
	return s.Stream.Close()
}

// Transport for any pipe.Conn
type Transport struct {
	h ConnectHook
	MuxAdapter
	NetListener
	NetDialer
}

// Listen Generic
func (t Transport) Listen(c context.Context, a net.Addr) (pipe.Listener, error) {
	l, err := t.NetListener.Listen(c, a.Network(), a.String())
	return listener{
		h:                t.h,
		serverMuxAdapter: t.MuxAdapter,
		Listener:         l,
	}, err
}

// Dial Generic
func (t Transport) Dial(c context.Context, a net.Addr) (pipe.Conn, error) {
	raw, err := t.NetDialer.DialContext(c, a.Network(), a.String())
	if err != nil {
		return nil, errors.Wrap(err, "dial")
	}

	if raw, err = t.h.OnDialConnected(c, raw); err != nil {
		return nil, err
	}

	return t.AdaptClient(raw)
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
func New(opt ...Option) (t Transport) {

	OptConnectHook(noopConnect{})(&t)
	OptMuxAdapter(MuxConfig{})(&t)

	for _, fn := range opt {
		fn(&t)
	}

	return t
}
