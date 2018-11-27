package inproc

import (
	"context"
	"net"

	"github.com/lthibault/pipewerks/pkg/transport/generic"
	"github.com/pkg/errors"
)

type ctxKey uint8

const (
	keyDialback ctxKey = iota
)

func setDialback(c context.Context, a Addr) context.Context {
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

// OptDialback sets the dialback addr for a transport.  This is useful when
// dialers are also listening, and need to announce the listen address.
func OptDialback(a net.Addr) Option {
	if a.Network() != "inproc" {
		panic(errors.Errorf("invalid network %s", a.Network()))
	}

	return func(t *Transport) (prev Option) {
		prev = OptDialback(t.dialback)
		t.dialback = Addr(a.String())
		return
	}
}

// OptAddrSpace sets the namespace for the the Transport
func OptAddrSpace(n NameSpace) Option {
	return func(t *Transport) (prev Option) {
		prev = OptAddrSpace(namespace{
			NetListener: t.Transport.NetListener,
			NetDialer:   t.Transport.NetDialer,
		})
		t.Transport.NetListener = n
		t.Transport.NetDialer = n
		return
	}
}

// OptMuxAdapter sets the muxer
func OptMuxAdapter(x generic.MuxAdapter) Option {
	return func(t *Transport) (prev Option) {
		prev = OptMuxAdapter(t.Transport.MuxAdapter)
		t.Transport.MuxAdapter = x
		return
	}
}
