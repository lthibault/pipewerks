package inproc

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	local  Addr = "local"
	remote Addr = "remote"
)

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

	t.Run("Stream", func(t *testing.T) {
		t.Parallel()

		p := newConnPair(context.Background(), local, remote)

		t.Run("Open", func(t *testing.T) {
			lc := p.Local()
			s, err := lc.Stream().Open()
			assert.NoError(t, err)

			_, err = io.Copy(s, bytes.NewBuffer([]byte("testing")))
			assert.NoError(t, err)
		})

		t.Run("Accept", func(t *testing.T) {
			rc := p.Remote()
			s, err := rc.Stream().Accept()
			assert.NoError(t, err)

			b := new(bytes.Buffer)
			_, err = io.Copy(b, s)
			assert.NoError(t, err)
			assert.Equal(t, "testing", b.String())
		})

	})
}
