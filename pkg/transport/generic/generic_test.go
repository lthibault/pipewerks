package generic

import (
	"errors"
	"net"
	"testing"

	"github.com/hashicorp/yamux"
	"github.com/stretchr/testify/assert"
)

type mockListener struct {
	c   net.Conn
	err error
}

func newMockListener(err error) listener {
	conn, _ := net.Pipe()
	return listener{
		serverMuxAdapter: MuxConfig{},
		Listener: mockListener{
			err: err,
			c:   conn,
		},
	}
}

func (mockListener) Addr() net.Addr              { return nil }
func (mockListener) Close() error                { return nil }
func (l mockListener) Accept() (net.Conn, error) { return l.c, l.err }

func TestListener(t *testing.T) {

	t.Run("Accept", func(t *testing.T) {
		t.Run("Succeed", func(t *testing.T) {
			_, err := newMockListener(nil).Accept()
			assert.NoError(t, err)
		})

		t.Run("Fail", func(t *testing.T) {
			t.Run("ListenError", func(t *testing.T) {
				_, err := newMockListener(errors.New("")).Accept()
				assert.Error(t, err)
			})

			t.Run("MuxError", func(t *testing.T) {
				l := newMockListener(nil)
				var mx MuxConfig
				mx.Config = new(yamux.Config)
				l.serverMuxAdapter = mx
				_, err := l.Accept()
				assert.Error(t, err)
			})
		})
	})
}

func TestMuxConfig(t *testing.T) {
	var mx MuxConfig
	dconn, lconn := net.Pipe()

	t.Run("ValidConfig", func(t *testing.T) {
		t.Run("AdaptClient", func(t *testing.T) {
			_, err := mx.AdaptClient(dconn)
			assert.NoError(t, err)
		})

		t.Run("AdaptServer", func(t *testing.T) {
			_, err := mx.AdaptServer(lconn)
			assert.NoError(t, err)
		})
	})

	t.Run("InvalidConfig", func(t *testing.T) {
		mx.Config = new(yamux.Config)
		assert.Error( // sanity check
			t,
			yamux.VerifyConfig(mx.Config),
			"YAMUX config is valid. Subsequent tests will FAIL.",
		)

		t.Run("AdaptClient", func(t *testing.T) {
			_, err := mx.AdaptClient(dconn)
			assert.Error(t, err)
		})

		t.Run("AdaptServer", func(t *testing.T) {
			_, err := mx.AdaptServer(lconn)
			assert.Error(t, err)
		})
	})
}
