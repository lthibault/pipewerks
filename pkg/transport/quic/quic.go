package quic

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/SentimensRG/ctx"
	log "github.com/lthibault/log/pkg"
	pipe "github.com/lthibault/pipewerks/pkg"
	quic "github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
)

// Config for QUIC protocol
type Config = quic.Config

type addresser interface {
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

type conn struct{ quic.Session }

func mkConn(s quic.Session) *conn {
	return &conn{Session: s}
}

func (c conn) AcceptStream() (pipe.Stream, error) {
	s, err := c.Session.AcceptStream()
	return stream{Stream: s, addresser: c}, err
}

func (c conn) OpenStream() (pipe.Stream, error) {
	s, err := c.Session.OpenStream()
	return stream{Stream: s, addresser: c}, err
}

type stream struct {
	quic.Stream
	addresser
}

func (s stream) StreamID() uint32 { return uint32(s.Stream.StreamID()) }

type hookSlice []pipe.Hook

func (hs *hookSlice) SetHook(h pipe.Hook) { *hs = append(*hs, h) }
func (hs *hookSlice) RmHook(h pipe.Hook) {
	for i := range *hs {
		if h == (*hs)[i] {
			*hs = append((*hs)[:i], (*hs)[i+1:]...)
		}
	}
}

// Transport over QUIC
type Transport struct {
	q *Config
	t *tls.Config
}

// Dial the specified address
func (t *Transport) Dial(c context.Context, a net.Addr) (pipe.Conn, error) {
	log.Get(c).Debug("dialing")

	sess, err := quic.DialAddrContext(c, a.String(), t.t, t.q)
	if err != nil {
		return nil, errors.Wrap(err, "dial")
	}

	log.Get(c).Debug("negotiating")

	return mkConn(sess), nil
}

// Listen on the specified address
func (t *Transport) Listen(c context.Context, a net.Addr) (pipe.Listener, error) {
	log.Get(c).Debug("listening")

	l, err := quic.ListenAddr(a.String(), t.t, t.q)
	if err != nil {
		return nil, err
	}
	ctx.Defer(c, func() { l.Close() })

	return listener{l}, nil
}

type listener struct{ quic.Listener }

func (l listener) Accept() (conn pipe.Conn, err error) {
	sess, err := l.Listener.Accept()
	if err != nil {
		return nil, errors.Wrap(err, "accept")
	}

	return mkConn(sess), nil
}

// New Transport over QUIC
func New(opt ...Option) *Transport {
	t := new(Transport)
	for _, o := range opt {
		o(t)
	}
	return t
}
