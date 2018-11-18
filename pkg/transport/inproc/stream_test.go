package inproc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

var addrs = ep{local: Addr("/local"), remote: Addr("/remote")}

func TestStream(t *testing.T) {
	p := newStreamPair(context.Background(), addrs)

	t.Run("LocalStream", func(t *testing.T) {
		s := p.Local()
		assert.Equal(t, addrs.Local(), s.Endpoint().Local())
		assert.Equal(t, addrs.Remote(), s.Endpoint().Remote())
	})

	t.Run("RemoteStream", func(t *testing.T) {
		s := p.Remote()
		assert.Equal(t, addrs.Remote(), s.Endpoint().Local())
		assert.Equal(t, addrs.Local(), s.Endpoint().Remote())
	})
}
