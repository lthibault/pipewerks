package inproc

import (
	"context"
	"errors"
	"io"
	"time"
	"unsafe"

	radix "github.com/armon/go-radix"
	net "github.com/lthibault/pipewerks/pkg/net"
	"github.com/satori/uuid"
	"golang.org/x/sync/errgroup"
)

type connPipe struct {
	local, remote net.Addr
}

func newConnPipe(local, remote net.Addr) connPipe {
	return connPipe{local: local, remote: remote}
}

func (c connPipe) LocalConn() net.Conn {
	return newConn(c.local, c.remote)
}
func (c connPipe) RemoteConn() net.Conn {
	return newConn(c.remote, c.local)
}

type stream struct {
	ep
	c context.Context
	io.ReadCloser
	io.WriteCloser

	dl, readDL, writeDL <-chan time.Time
}

func (s stream) Context() context.Context   { return s.c }
func (s stream) Endpoint() net.EndpointPair { return s.ep }

func (s stream) Close() error {
	var g errgroup.Group
	g.Go(s.ReadCloser.Close)
	g.Go(s.WriteCloser.Close)
	return g.Wait()
}

func (s *stream) SetDeadline(t time.Time) error {
	s.dl = time.After(t.Sub(time.Now()))
	return nil
}

func (s *stream) SetReadDeadline(t time.Time) error {
	s.readDL = time.After(t.Sub(time.Now()))
	return nil
}

func (s *stream) SetWriteDeadline(t time.Time) error {
	s.writeDL = time.After(t.Sub(time.Now()))
	return nil
}

type addr string

func (addr) Network() string  { return "inproc" }
func (a addr) String() string { return string(a) }

type streamer struct {
	ep
}

func (s streamer) CloseWithError(err error) error {
	panic("function NOT IMPLEMENTED")
}

func (s streamer) Accept() (net.Stream, error) {
	panic("Accept NOT IMPLEMENTED")
}

func (s streamer) Open() (net.Stream, error) {
	panic("Open NOT IMPLEMENTED")
}

type ep struct{ local, remote net.Addr }

func (ep ep) Endpoint() net.EndpointPair { return ep }
func (ep ep) Local() net.Addr            { return ep.local }
func (ep ep) Remote() net.Addr           { return ep.remote }

type conn struct {
	local, remote net.Addr
	c             context.Context
	s             streamer
	ep
}

func newConn(local, remote net.Addr) conn {
	ept := ep{local: local, remote: remote}
	return conn{
		c:  context.Background(),
		ep: ept,
		s: streamer{
			ep: ept,
		},
	}
}

func (c conn) Context() context.Context   { return c.c }
func (c conn) Stream() net.Streamer       { return c.s }
func (c conn) Endpoint() net.EndpointPair { return c.ep }

func (c conn) Close() error { return c.s.CloseWithError(nil) }

func (c conn) CloseWithError(_ net.ErrorCode, err error) error {
	return c.s.CloseWithError(err)
}

type listener struct {
	a  net.Addr
	ch chan net.Conn
}

func (l listener) Addr() net.Addr { return l.a }

func (l listener) Close() (err error) {
	defer func() {
		if recover() != nil {
			err = errors.New("already closed")
		}
	}()

	close(l.ch)
	return
}

func (l listener) Accept(c context.Context) (net.Conn, error) {
	select {
	case <-c.Done():
		return nil, c.Err()
	case conn := <-l.ch:
		return conn, nil
	}
}

func newListener(a net.Addr) listener {
	return listener{a: a, ch: make(chan net.Conn)}
}

type mux radix.Tree

func (m *mux) Get(addr string) (l listener, ok bool) {
	var v interface{}
	if v, ok = (*radix.Tree)(unsafe.Pointer(m)).Get(addr); ok {
		l = v.(listener)
	}
	return
}

func (m *mux) Put(l listener) (prev listener, ok bool) {
	var v interface{}
	if v, ok = (*radix.Tree)(unsafe.Pointer(m)).Insert(l.Addr().String(), l); ok {
		prev = v.(listener)
	}
	return
}

func (m *mux) Del(addr string) (prev listener, ok bool) {
	var v interface{}
	if v, ok = (*radix.Tree)(unsafe.Pointer(m)).Delete(addr); ok {
		prev = v.(listener)
	}
	return
}

// Transport bytes around the process
type Transport struct{ m *mux }

// Listen for incoming connections
func (t Transport) Listen(c context.Context, a net.Addr) (net.Listener, error) {
	if a.Network() != "inproc" {
		return nil, errors.New("invalid network")
	}

	l := newListener(a)
	if prev, ok := t.m.Put(l); ok {
		t.m.Put(prev) // swap it back
		return nil, errors.New("address unavailable")
	}

	return l, nil
}

// Dial opens a conneciton
func (t Transport) Dial(c context.Context, a net.Addr) (conn net.Conn, err error) {
	if a.Network() != "inproc" {
		err = errors.New("invalid network")
		return
	}

	l, ok := t.m.Get(a.String())
	if !ok {
		err = errors.New("not found")
		return
	}

	p := newConnPipe(addr(uuid.NewV4().String()), a)

	select {
	case l.ch <- p.RemoteConn():
		conn = p.LocalConn()
	case <-c.Done():
		err = c.Err()
	}

	return
}

// New in-process Transport
func New(opt ...Option) *Transport {
	t := new(Transport)
	t.m = (*mux)(radix.New())
	for _, o := range opt {
		o(t)
	}
	return t
}
