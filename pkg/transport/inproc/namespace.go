package inproc

import (
	"context"

	pipe "github.com/lthibault/pipewerks/pkg"
)

// DefaultNamespace is a global namespace that is used by default
var DefaultNamespace Namespace = make(namespace)

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

type namespace map[string]Connector

func (n namespace) Bind(addr string, c Connector) bool {
	if _, ok := n[addr]; ok {
		return false
	}

	n[addr] = c
	return true
}

func (n namespace) GetConnector(addr string) (c Connector, ok bool) {
	c, ok = n[addr]
	return
}

func (n namespace) Free(addr string) { delete(n, addr) }
