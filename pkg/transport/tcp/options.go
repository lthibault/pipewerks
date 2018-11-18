package tcp

import (
	"net"

	"github.com/lthibault/pipewerks/pkg/transport/generic"
)

// Option for TCP transport
type Option func(Transport) (prev Option)

// OptListener sets the ListenConfig
func OptListener(l *net.ListenConfig) Option {
	return func(t Transport) (prev Option) {
		prev = OptListener(t.Transport.NetListener.(*net.ListenConfig))
		t.Transport.NetListener = l
		return
	}
}

// OptDialer sets the dialer
func OptDialer(d *net.Dialer) Option {
	return func(t Transport) (prev Option) {
		prev = OptDialer(t.Transport.NetDialer.(*net.Dialer))
		t.Transport.NetDialer = d
		return
	}
}

// OptMux sets the muxer
func OptMux(c generic.MuxConfig) Option {
	return func(t Transport) (prev Option) {
		prev = OptMux(t.Transport.MuxConfig)
		t.Transport.MuxConfig = c
		return
	}
}
