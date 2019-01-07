package generic

import (
	"net"
	"testing"
	"time"

	"github.com/hashicorp/yamux"
	pipe "github.com/lthibault/pipewerks/pkg"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
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

func TestConnection(t *testing.T) {
	yc, c := net.Pipe()

	dsess, err := yamux.Client(c, nil)
	assert.NoError(t, err)

	lsess, err := yamux.Server(yc, nil)
	assert.NoError(t, err)

	conn := connection{lsess}
	assert.NoError(t, conn.Context().Err())

	t.Run("Close", func(t *testing.T) {
		assert.NoError(t, dsess.Close())
		time.Sleep(time.Millisecond)

		assert.Error(t, conn.Context().Err())
		assert.True(t, func() bool {
			select {
			case <-conn.Context().Done():
				return true
			default:
				return false
			}
		}())
	})

}

func mkConn() (pipe.Conn, pipe.Conn, error) {
	ds, ls := net.Pipe()

	dsess, err := yamux.Client(ds, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "client conn")
	}

	lsess, err := yamux.Server(ls, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "server conn")
	}

	return connection{dsess}, connection{lsess}, nil
}

func TestStream(t *testing.T) {
	dc, lc, err := mkConn()
	assert.NoError(t, err, "canary failed")

	var ds, ls pipe.Stream
	var g errgroup.Group
	g.Go(func() (err error) {
		ds, err = dc.OpenStream()
		return
	})
	g.Go(func() (err error) {
		ls, err = lc.AcceptStream()
		return
	})
	assert.NoError(t, g.Wait())

	t.Run("ChanValid", func(t *testing.T) {
		t.Run("DialStream", func(t *testing.T) {
			assert.NoError(t, ds.Context().Err())

			select {
			case <-ds.Context().Done():
				t.Error("channel recved on active stream")
			default:
			}
		})

		t.Run("ListenStream", func(t *testing.T) {
			assert.NoError(t, ls.Context().Err())

			select {
			case <-ls.Context().Done():
				t.Error("channel recved on active stream")
			default:
			}
		})
	})

	t.Run("Close", func(t *testing.T) {
		assert.NoError(t, ds.Close())
		assert.NoError(t, dc.Context().Err())
		assert.NoError(t, lc.Context().Err())

		t.Run("DialStream", func(t *testing.T) {
			assert.Error(t, ds.Context().Err())
			select {
			case <-ds.Context().Done():
			default:
				t.Error("dial closed, but context not expired")
			}
		})

		t.Run("ListenStream", func(t *testing.T) {
			_, err := ls.Read([]byte{}) // ensure context closure is triggered
			assert.Error(t, err)

			assert.Error(t, ls.Context().Err())
			select {
			case <-ls.Context().Done():
			default:
				t.Error("dial closed, but context not expired")
			}
		})
	})
}
