package quic

import (
	"crypto/tls"

	quic "github.com/lucas-clemente/quic-go"
)

// Option for Transport
type Option func(*Transport) (prev Option)

// OptQuic sets the QUIC configuration
func OptQuic(q *quic.Config) Option {
	return func(t *Transport) (prev Option) {
		prev = OptQuic(t.q)
		t.q = q
		return
	}
}

// OptTLS sets the TLS configuration
func OptTLS(tc *tls.Config) Option {
	return func(t *Transport) (prev Option) {
		prev = OptTLS(t.t)
		t.t = tc
		return
	}
}
