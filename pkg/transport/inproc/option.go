package inproc

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
