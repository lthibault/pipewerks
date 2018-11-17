package net

import (
	"context"
)

// Dial initiates a connection
func Dial(c context.Context, t Transport, a Addr) (Conn, error) {
	return t.Dial(c, a)
}

// Listen for incoming connections
func Listen(c context.Context, t Transport, a Addr) (Listener, error) {
	return t.Listen(c, a)
}
