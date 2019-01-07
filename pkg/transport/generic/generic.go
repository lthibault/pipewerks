package generic

import (
	"context"
	"net"
	"time"

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

type connection struct{ *yamux.Session }

func (c connection) Context() context.Context {
	return ctx.AsContext(ctx.C(c.CloseChan()))
}

func (c connection) OpenStream() (pipe.Stream, error) {
	s, err := c.Session.OpenStream()
	if err != nil {
		return nil, err
	}

	return mkStream(c.Context(), s), nil
}

func (c connection) AcceptStream() (pipe.Stream, error) {
	s, err := c.Session.AcceptStream()
	if err != nil {
		return nil, err
	}

	return mkStream(c.Context(), s), nil
}

type stream struct {
	c      context.Context
	cancel func()
	s      *yamux.Stream
}

func mkStream(c context.Context, s *yamux.Stream) (strm stream) {
	strm.c, strm.cancel = context.WithCancel(c)
	strm.s = s
	return
}

func (s stream) LocalAddr() net.Addr  { return s.s.LocalAddr() }
func (s stream) RemoteAddr() net.Addr { return s.s.RemoteAddr() }

func (s stream) Read(b []byte) (n int, err error) {
	if n, err = s.s.Read(b); err != nil {
		err = s.chkErr(err)
	}
	return
}

func (s stream) Write(b []byte) (n int, err error) {
	if n, err = s.s.Write(b); err != nil {
		err = s.chkErr(err)
	}
	return
}

func (s stream) SetDeadline(t time.Time) error {
	if err := s.s.SetDeadline(t); err != nil {
		return s.chkErr(err)
	}
	return nil
}

func (s stream) SetReadDeadline(t time.Time) error {
	if err := s.s.SetReadDeadline(t); err != nil {
		return s.chkErr(err)
	}
	return nil
}

func (s stream) SetWriteDeadline(t time.Time) error {
	if err := s.s.SetWriteDeadline(t); err != nil {
		return s.chkErr(err)
	}
	return nil
}

func (s stream) StreamID() uint32 { return s.s.StreamID() }

func (s stream) chkErr(err error) error {
	if e, ok := err.(net.Error); ok && e.Temporary() {
		return err
	}

	s.cancel()
	return err
}

func (s stream) Context() context.Context { return s.c }
func (s stream) Close() error {
	s.cancel()
	return s.s.Close()
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
	return listener{
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

	OptMuxAdapter(MuxConfig{})(&t)
	for _, fn := range opt {
		fn(&t)
	}

	return t
}
