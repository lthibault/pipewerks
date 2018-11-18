package inproc

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	local  Addr = "local"
	remote Addr = "remote"
)

func TestAtomicError(t *testing.T) {
	var ae atomicErr

	t.Run("DefaultNil", func(t *testing.T) {
		assert.NoError(t, ae.Load())
	})

	t.Run("SetPersists", func(t *testing.T) {
		ae.Store(errors.New(""))
		assert.Error(t, ae.Load())
	})
}

func TestConnPair(t *testing.T) {
	p := newConnPair(context.Background(), local, remote)

	t.Run("ConnAddr", func(t *testing.T) {
		t.Run("Local", func(t *testing.T) {
			assert.Equal(t, local, p.Local().Endpoint().Local())
			assert.Equal(t, remote, p.Local().Endpoint().Remote())
		})

		t.Run("Remote", func(t *testing.T) {
			assert.Equal(t, local, p.Remote().Endpoint().Remote())
			assert.Equal(t, remote, p.Remote().Endpoint().Local())
		})
	})
}
