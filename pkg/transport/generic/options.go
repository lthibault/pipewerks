package generic

import (
	"context"
	gonet "net"

	"github.com/hashicorp/yamux"
)

type netConn = gonet.Conn

// NetListener can produce a standard library Listener
type NetListener interface {
	Listen(c context.Context, network, address string) (gonet.Listener, error)
}

// NetDialer can produce a standard library Dialer
type NetDialer interface {
	DialContext(c context.Context, network, address string) (gonet.Conn, error)
}

// Option for TCP transport
type Option func(*Transport) (prev Option)

// OptListener sets the ListenConfig
func OptListener(l NetListener) Option {
	return func(t *Transport) (prev Option) {
		prev = OptListener(t.l)
		t.l = l
		return
	}
}

// OptDialer sets the dialer
func OptDialer(d NetDialer) Option {
	return func(t *Transport) (prev Option) {
		prev = OptDialer(t.d)
		t.d = d
		return
	}
}

// OptMux sets the muxer
func OptMux(c yamux.Config) Option {
	return func(t *Transport) (prev Option) {
		prev = OptMux(t.c)
		t.c = c
		return
	}
}
