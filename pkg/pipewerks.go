package net

import (
	"context"

	net "github.com/lthibault/pipewerks/pkg/net"
)

// Dial initiates a connection
func Dial(c context.Context, t net.Transport, a net.Addr) (net.Conn, error) {
	return t.Dial(c, a)
}

// Listen for incoming connections
func Listen(c context.Context, t net.Transport, a net.Addr) (net.Listener, error) {
	return t.Listen(c, a)
}
