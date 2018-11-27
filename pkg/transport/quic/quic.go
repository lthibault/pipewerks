package quic

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"io"
	"io/ioutil"
	"net"

	"github.com/SentimensRG/ctx"
	log "github.com/lthibault/log/pkg"
	pipe "github.com/lthibault/pipewerks/pkg"
	quic "github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
)

// Config for QUIC protocol
type Config = quic.Config

type conn struct{ quic.Session }

func mkConn(s quic.Session) *conn {
	return &conn{Session: s}
}

func (conn *conn) Stream() pipe.Streamer       { return conn }
func (conn conn) Accept() (pipe.Stream, error) { return conn.Accept() }

func (conn conn) Open() (pipe.Stream, error) {
	s, err := conn.OpenStream()
	if err != nil {
		return nil, errors.Wrap(err, "open stream")
	}

	var size uint16
	if err = binary.Read(s, binary.BigEndian, &size); err != nil {
		return nil, errors.Wrap(err, "read pathsize")
	}

	path, err := ioutil.ReadAll(io.LimitReader(s, int64(size)))
	if err != nil {
		return nil, errors.Wrap(err, "read path")
	}

	return &stream{
		path:   string(path),
		Stream: s,
		Edge:   conn,
	}, nil
}

func (conn *conn) Endpoint() pipe.Edge { return conn }
func (conn conn) Local() net.Addr      { return conn.LocalAddr() }
func (conn conn) Remote() net.Addr     { return conn.RemoteAddr() }

type stream struct {
	path string
	quic.Stream
	pipe.Edge
}

func (s stream) Path() string        { return s.path }
func (s stream) Endpoint() pipe.Edge { return s.Edge }
func (s stream) StreamID() uint32    { return uint32(s.Stream.StreamID()) }

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
