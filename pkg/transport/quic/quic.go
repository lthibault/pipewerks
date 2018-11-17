package quic

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"io"
	"io/ioutil"

	"github.com/SentimensRG/ctx"
	log "github.com/lthibault/log/pkg"
	net "github.com/lthibault/pipewerks/pkg"
	quic "github.com/lucas-clemente/quic-go"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// Config for QUIC protocol
type Config = quic.Config

type conn struct{ quic.Session }

func mkConn(s quic.Session) *conn {
	return &conn{Session: s}
}

func (conn *conn) Stream() net.Streamer       { return conn }
func (conn conn) Accept() (net.Stream, error) { return conn.Accept() }

func (conn conn) Open() (net.Stream, error) {
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
		path:         string(path),
		Stream:       s,
		EndpointPair: conn,
	}, nil
}

func (conn *conn) Endpoint() net.EndpointPair { return conn }
func (conn conn) Local() net.Addr             { return conn.LocalAddr() }
func (conn conn) Remote() net.Addr            { return conn.RemoteAddr() }

func (conn conn) CloseWithError(c net.ErrorCode, err error) error {
	return conn.Session.CloseWithError(quic.ErrorCode(c), err)
}

type stream struct {
	path string
	quic.Stream
	net.EndpointPair
}

func (s stream) Path() string               { return s.path }
func (s stream) Endpoint() net.EndpointPair { return s.EndpointPair }

// Transport over QUIC
type Transport struct {
	q *Config
	t *tls.Config
}

// Dial the specified address
func (t *Transport) Dial(c context.Context, a net.Addr) (net.Conn, error) {
	log.Get(c).Debug("dialing")

	sess, err := quic.DialAddrContext(c, a.String(), t.t, t.q)
	if err != nil {
		return nil, errors.Wrap(err, "dial")
	}

	log.Get(c).Debug("negotiating")

	return mkConn(sess), nil
}

// Listen on the specified address
func (t *Transport) Listen(c context.Context, a net.Addr) (net.Listener, error) {
	log.Get(c).Debug("listening")

	l, err := quic.ListenAddr(a.String(), t.t, t.q)
	if err != nil {
		return nil, err
	}
	ctx.Defer(c, func() { l.Close() })

	return listener{l}, nil
}

type listener struct{ quic.Listener }

func (l listener) Accept(c context.Context) (conn net.Conn, err error) {
	var sess quic.Session

	var g errgroup.Group
	g.Go(func() error {
		sess, err = l.Listener.Accept()
		return err
	})

	if g.Wait() == nil {
		conn = mkConn(sess)
	}

	return
}

// New Transport over QUIC
func New(opt ...Option) *Transport {
	t := new(Transport)
	for _, o := range opt {
		o(t)
	}
	return t
}
