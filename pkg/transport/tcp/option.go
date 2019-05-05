package tcp

import (
	"net"

	"github.com/lthibault/pipewerks/pkg/transport/generic"
)

// Option for TCP transport
type Option func(*Transport) (prev Option)

// OptListener sets the ListenConfig
func OptListener(l *net.ListenConfig) Option {
	return func(t *Transport) (prev Option) {
		prev = OptListener(t.Transport.NetListener.(*net.ListenConfig))
		OptGeneric(generic.OptListener(l))(t)
		return
	}
}

// OptDialer sets the dialer
func OptDialer(d *net.Dialer) Option {
	return func(t *Transport) (prev Option) {
		prev = OptDialer(t.Transport.NetDialer.(*net.Dialer))
		OptGeneric(generic.OptDialer(d))(t)
		return
	}
}

// OptGeneric sets an option on the underlying generic transport
func OptGeneric(opt generic.Option) Option {
	return func(t *Transport) Option {
		return OptGeneric(opt(&t.Transport))
	}
}
