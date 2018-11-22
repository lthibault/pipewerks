package generic

import (
	"context"
	"net"

	"github.com/hashicorp/yamux"
)

// MuxConfig configures the muxer
type MuxConfig = yamux.Config

// NetListener can produce a standard library Listener
type NetListener interface {
	Listen(c context.Context, network, address string) (net.Listener, error)
}

// NetDialer can produce a standard library Dialer
type NetDialer interface {
	DialContext(c context.Context, network, address string) (net.Conn, error)
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

// OptMux sets the muxer
func OptMux(c *MuxConfig) Option {
	return func(t *Transport) (prev Option) {
		prev = OptMux(t.MuxConfig)
		t.MuxConfig = c
		return
	}
}
