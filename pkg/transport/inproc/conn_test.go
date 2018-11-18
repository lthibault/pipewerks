package inproc

import (
	"context"
	"testing"

	"github.com/pkg/errors"

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

func TestConn(t *testing.T) {
	t.Run("Close", func(t *testing.T) {
		t.Run("Local", func(t *testing.T) {
			p := newConnPair(context.Background(), local, remote)
			lc := p.Local()
			// rc := p.Remote()
			assert.NoError(t, lc.Close())
		})

		t.Run("Remote", func(t *testing.T) {
			p := newConnPair(context.Background(), local, remote)
			// lc := p.Local()
			rc := p.Remote()
			assert.NoError(t, rc.Close())
		})
	})

	t.Run("CloseWithError", func(t *testing.T) {
		t.Run("Local", func(t *testing.T) {
			p := newConnPair(context.Background(), local, remote)
			lc := p.Local()
			rc := p.Remote()

			assert.NoError(t, lc.CloseWithError(0, errors.New("close local")))

			t.Run("Open", func(t *testing.T) {
				t.Run("Local", func(t *testing.T) {
					_, err := lc.Stream().Open()
					assert.EqualError(t, err, "closed: context canceled")
				})

				t.Run("Remote", func(t *testing.T) {
					_, err := rc.Stream().Open()
					assert.EqualError(t, err, "close local")
				})
			})

			t.Run("Accept", func(t *testing.T) {
				t.Run("Local", func(t *testing.T) {
					_, err := lc.Stream().Accept()
					assert.EqualError(t, err, "closed: context canceled")
				})

				t.Run("Remote", func(t *testing.T) {
					_, err := rc.Stream().Accept()
					assert.EqualError(t, err, "close local")
				})
			})
		})

		t.Run("Remote", func(t *testing.T) {
			p := newConnPair(context.Background(), local, remote)
			lc := p.Local()
			rc := p.Remote()

			assert.NoError(t, rc.CloseWithError(0, errors.New("close remote")))

			t.Run("Open", func(t *testing.T) {
				t.Run("Local", func(t *testing.T) {
					_, err := lc.Stream().Open()
					assert.EqualError(t, err, "close remote")
				})

				t.Run("Remote", func(t *testing.T) {
					_, err := rc.Stream().Open()
					assert.EqualError(t, err, "closed: context canceled")
				})
			})

			t.Run("Accept", func(t *testing.T) {
				t.Run("Local", func(t *testing.T) {
					_, err := lc.Stream().Accept()
					assert.EqualError(t, err, "close remote")
				})

				t.Run("Remote", func(t *testing.T) {
					_, err := rc.Stream().Accept()
					assert.EqualError(t, err, "closed: context canceled")
				})
			})
		})
	})
}
