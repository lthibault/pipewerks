package httpipe

import (
	"context"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	pipe "github.com/lthibault/pipewerks/pkg"
)

// PipeDialer is the client end of a Pipewerks Transport.
type PipeDialer interface {
	Dial(context.Context, net.Addr) (pipe.Conn, error)
}

type address struct{ network, addr string }

func (a address) Network() string { return a.network }
func (a address) String() string  { return a.addr }

type refConn struct {
	pipe.Conn
	ctr uint32
	gc  func()
}

func (c *refConn) OpenStream() (s pipe.Stream, err error) {
	if s, err = c.Conn.OpenStream(); err == nil {
		atomic.AddUint32(&c.ctr, 1)
		s = &refStream{Stream: s, ctr: &c.ctr}
	}
	return
}

type refStream struct {
	pipe.Stream
	ctr *uint32
	gc  func()
}

func (s *refStream) Close() (err error) {
	// Assumes that (a) Stream.Close is thread-safe, (b) it returns errors when called
	// multiple times and (c) that the HTTP client calls Close().
	if err = s.Stream.Close(); err == nil {
		s.gc()
		// TODO:  return s to a sync.Pool ?
	}

	return
}

type dialer struct {
	sync.Mutex
	PipeDialer
	store map[address]*refConn
}

func newDialContext(d PipeDialer) func(context.Context, string, string) (net.Conn, error) {
	return (&dialer{
		PipeDialer: d,
		store:      make(map[address]*refConn),
	}).DialContext
}

func (d *dialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	pconn, err := d.getConn(ctx, address{network, addr})
	if err != nil {
		return nil, err
	}

	return pconn.OpenStream()
}

func (d *dialer) getConn(ctx context.Context, a address) (pipe.Conn, error) {
	d.Lock()
	defer d.Unlock()

	var ok bool
	var pconn pipe.Conn

	// We may already have a connection to the remote host.  If we don't, open one.
	if pconn, ok = d.store[a]; !ok {
		var err error
		if pconn, err = d.Dial(ctx, a); err != nil {
			return nil, err
		}
	}

	return &refConn{
		Conn: pconn,
		gc: func() {
			d.Lock()
			delete(d.store, a)
			d.Unlock()
		},
	}, nil
}

// NewTransport produces an HTTP Transport that uses the specified Pipewerks Transport.
func NewTransport(d PipeDialer) *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           newDialContext(d),
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
}
