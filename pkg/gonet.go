package net

import "net"

// Addr represents a network end point address.
//
// The two methods Network and String conventionally return strings
// that can be passed as the arguments to Dial, but the exact form
// and meaning of the strings is up to the implementation.
type Addr = net.Addr

// An Error represents a network error.
type Error interface {
	error
	Timeout() bool   // Is the error a timeout?
	Temporary() bool // Is the error temporary?
}
