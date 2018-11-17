package inproc

// Transport bytes around the process
type Transport struct{}

// New in-process Transport
func New(opt ...Option) *Transport {
	t := new(Transport)
	for _, o := range opt {
		o(t)
	}
	return t
}
