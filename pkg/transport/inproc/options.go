package inproc

// Option for inproc transport
type Option func(*Transport) Option

// OptNamespace sets the namespace for the transport instance
func OptNamespace(ns Namespace) Option {
	return func(t *Transport) (prev Option) {
		prev = OptNamespace(t.ns)
		t.ns = ns
		return
	}
}
