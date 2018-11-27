package generic

import (
	"context"
	"net"

	"github.com/hashicorp/yamux"
	pipe "github.com/lthibault/pipewerks/pkg"
	"github.com/pkg/errors"
)

// NetListener can produce a standard library Listener
type NetListener interface {
	Listen(c context.Context, network, address string) (net.Listener, error)
}

// NetDialer can produce a standard library Dialer
type NetDialer interface {
	DialContext(c context.Context, network, address string) (net.Conn, error)
}

type serverMuxAdapter interface {
	AdaptServer(net.Conn) (pipe.Conn, error)
}

// MuxAdapter can adapt a go net.Conn into a pipe.Conn
type MuxAdapter interface {
	AdaptServer(net.Conn) (pipe.Conn, error)
	AdaptClient(net.Conn) (pipe.Conn, error)
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

// Option for TCP transport
type Option func(*Transport) (prev Option)

// OptListener sets the ListenConfig
func OptListener(l NetListener) Option {
	return func(t *Transport) (prev Option) {
		prev = OptListener(t.NetListener)
		t.NetListener = l
		return
	}
}

// OptDialer sets the dialer
func OptDialer(d NetDialer) Option {
	return func(t *Transport) (prev Option) {
		prev = OptDialer(t.NetDialer)
		t.NetDialer = d
		return
	}
}

// OptMuxAdapter sets the muxer
func OptMuxAdapter(x MuxAdapter) Option {
	return func(t *Transport) (prev Option) {
		prev = OptMuxAdapter(t.MuxAdapter)
		t.MuxAdapter = x
		return
	}
}
