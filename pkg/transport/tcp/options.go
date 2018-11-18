package tcp

import (
	"net"

	"github.com/hashicorp/yamux"
)

// Option for TCP transport
type Option func(Transport) (prev Option)

// OptListener sets the ListenConfig
func OptListener(l *net.ListenConfig) Option {
	return func(t *Transport) (prev Option) {
		prev = OptListener(t.Transport.l)
		t.Transport.l = l
		return
	}
}

// OptDialer sets the dialer
func OptDialer(d *net.Dialer) Option {
	return func(t *Transport) (prev Option) {
		prev = OptDialer(t.Transport.d)
		t.Transport.d = d
		return
	}
}

// OptMux sets the muxer
func OptMux(c yamux.Config) Option {
	return func(t *Transport) (prev Option) {
		prev = OptMux(t.Transport.c)
		t.Transport.c = c
		return
	}
}
