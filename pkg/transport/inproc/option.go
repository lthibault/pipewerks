package inproc

import "github.com/lthibault/pipewerks/pkg/transport/generic"

type namespace struct {
	generic.NetListener
	generic.NetDialer
}

var defaultMux = newMux()

// Option for Transport
type Option func(*Transport) (prev Option)

// OptDialback sets the dialback addr for a transport.  This is useful when
// dialers are also listening, and need to announce the listen address.
func OptDialback(a Addr) Option {
	return func(t *Transport) (prev Option) {
		prev = OptDialback(t.dialback)
		t.dialback = a
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
