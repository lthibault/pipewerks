package generic

import (
	"context"
	"net"

	"github.com/SentimensRG/ctx"
	pipe "github.com/lthibault/pipewerks/pkg"

	"github.com/hashicorp/yamux"
	"github.com/pkg/errors"
)

const (
	// DialEndpoint initiated the connection
	DialEndpoint EndpointType = true
	// ListenEndpoint received the connection request
	ListenEndpoint EndpointType = false
)

// EndpointType specifies whether the endpoint is a client (dialer) or server
// (listener).
type EndpointType bool

// ConnectHandler is an interface that is satisfied by generic.Transport. It allows
// users to set callbacks that are invoked after a net.Conn is created, but
// before the stream muxer starts.
type ConnectHandler interface {
	Set(OnConnect)
	Rm(OnConnect)
}

// OnConnect is invoked when the Transport successfully opens a raw
// net.Conn. It allows user-defined logic to run on the raw connection before
// the stream muxer starts.
type OnConnect interface {
	Connected(net.Conn, EndpointType) (net.Conn, error)
}

// OnConnectFunc is a function that satisfies OnConnect
type OnConnectFunc func(net.Conn, EndpointType) (net.Conn, error)

// Connected is a callback that is invoked when a raw connection is established
func (f OnConnectFunc) Connected(c net.Conn, t EndpointType) (net.Conn, error) {
	return f(c, t)
}

type listener struct {
	cb []OnConnect
	serverMuxAdapter
	net.Listener
}

func (l listener) Accept() (pipe.Conn, error) {
	raw, err := l.Listener.Accept()
	if err != nil {
		return nil, errors.Wrap(err, "listener")
	}

	// Call the "connected" cb and run user-defined logic
	for _, h := range l.cb {
		if raw, err = h.Connected(raw, ListenEndpoint); err != nil {
			return nil, err
		}
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

type cbSlice []OnConnect

func (hs *cbSlice) Set(h OnConnect) { *hs = append(*hs, h) }
func (hs *cbSlice) Rm(h OnConnect) {
	for i := range *hs {
		if h == (*hs)[i] {
			*hs = append((*hs)[:i], (*hs)[i+1:]...)
		}
	}
}

// Transport for any pipe.Conn
type Transport struct {
	cbSlice
	MuxAdapter
	NetListener
	NetDialer
}

// Listen Generic
func (t Transport) Listen(c context.Context, a net.Addr) (pipe.Listener, error) {
	l, err := t.NetListener.Listen(c, a.Network(), a.String())
	return listener{
		Listener:         l,
		cb:               t.cbSlice,
		serverMuxAdapter: t.MuxAdapter,
	}, err
}

// Dial Generic
func (t Transport) Dial(c context.Context, a net.Addr) (pipe.Conn, error) {
	raw, err := t.NetDialer.DialContext(c, a.Network(), a.String())
	if err != nil {
		return nil, errors.Wrap(err, "dial")
	}

	// Call the "connected" cb and run user-defined logic.
	for _, cb := range t.cbSlice {
		if raw, err = cb.Connected(raw, DialEndpoint); err != nil {
			return nil, err
		}
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
	t.cbSlice = []OnConnect{}
	t.MuxAdapter = MuxConfig{}

	for _, fn := range opt {
		fn(&t)
	}

	return t
}
