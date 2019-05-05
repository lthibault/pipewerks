package inproc

import (
	"context"

	"github.com/lthibault/pipewerks/pkg/transport/generic"
)

type ctxKey uint8

const (
	keyDialback ctxKey = iota
)

// SetDialback sets an inproc address in the context that will be reported
// to the listener as the addr of the remote endpoint.
func SetDialback(c context.Context, a Addr) context.Context {
	return context.WithValue(c, keyDialback, a)
}

func getDialback(c context.Context) Addr {
	if a, ok := c.Value(keyDialback).(Addr); ok {
		return a
	}
	return ""
}

type namespace struct {
	generic.NetListener
	generic.NetDialer
}

var defaultMux = newMux()

// Option for Transport
type Option func(*Transport) (prev Option)

// OptNameSpace sets the namespace for the the Transport
func OptNameSpace(n NameSpace) Option {
	return func(t *Transport) (prev Option) {
		prev = OptNameSpace(namespace{
			NetListener: t.Transport.NetListener,
			NetDialer:   t.Transport.NetDialer,
		})

		OptGeneric(generic.OptDialer(n))(t)
		OptGeneric(generic.OptListener(n))(t)

		return
	}
}

// OptGeneric sets an option on the underlying generic transport
func OptGeneric(opt generic.Option) Option {
	return func(t *Transport) Option {
		return OptGeneric(opt(&t.Transport))
	}
}
