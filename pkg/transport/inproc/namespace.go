package inproc

import (
	"context"

	pipe "github.com/lthibault/pipewerks/pkg"
)

// DefaultNamespace is a global namespace that is used by default
var DefaultNamespace Namespace = make(BasicNamespace)

// Connector can wait for incoming connections
type Connector interface {
	Connect(context.Context, pipe.Conn) error
}

// Namespace is an isolated address space
type Namespace interface {
	Bind(string, Connector) bool
	GetConnector(string) (Connector, bool)
	Free(string)
}

// BasicNamespace is the implementation used by DefaultNamespace.
type BasicNamespace map[string]Connector

// Bind a connector to an address.
func (n BasicNamespace) Bind(addr string, c Connector) bool {
	if _, ok := n[addr]; ok {
		return false
	}

	n[addr] = c
	return true
}

// GetConnector at specified address.  ok is false if no connector is bound.
func (n BasicNamespace) GetConnector(addr string) (c Connector, ok bool) {
	c, ok = n[addr]
	return
}

// Free the address by removing its connector.
func (n BasicNamespace) Free(addr string) { delete(n, addr) }
