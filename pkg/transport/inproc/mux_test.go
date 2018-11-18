package inproc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	ListenerPath = "listener"
	ConnPath     = "conn"
	StreamPath   = "stream"
)

func TestMux(t *testing.T) {
	m := newMux()
	assert.Empty(t, m.r.ToMap())

	t.Run("Listener", func(t *testing.T) {
		var l listener

		t.Run("Store", func(t *testing.T) {
			t.Run("New", func(t *testing.T) {
				v, replace := m.SetListener(l, ListenerPath)
				assert.False(t, replace, "unexpected %v (type %T)", v, v)
			})

			t.Run("Replace", func(t *testing.T) {
				ln, replace := m.SetListener(listener{}, ListenerPath)
				assert.True(t, replace, "no replacee or replacement not reported")
				assert.Equal(t, l, ln.listener, "unexpected replacee")
				l = ln.listener // put it back so we can continue testing
			})
		})

		t.Run("Retrieve", func(t *testing.T) {
			t.Run("Existing", func(t *testing.T) {
				ln, ok := m.GetListener(ListenerPath)
				assert.True(t, ok, "retrieval failed OR false negative")
				assert.Equal(t, l, ln.listener, "unexpected replacee")
			})

			t.Run("Missing", func(t *testing.T) {
				ln, ok := m.GetListener("WRONG")
				assert.False(t, ok, "false positive")
				assert.Nil(t, ln.listener.a, "unexpected %T", ln.listener.a)
				assert.Nil(t, ln.listener.ch, "unexpected %T", ln.listener.ch)
			})
		})

		t.Run("Delete", func(t *testing.T) {
			t.Run("Existing", func(t *testing.T) {
				ln, ok := m.DelListener(ListenerPath)
				assert.True(t, ok, "retrieval failed OR false negative")
				assert.Equal(t, l, ln.listener, "unexpected replacee")
			})

			t.Run("Missing", func(t *testing.T) {
				ln, ok := m.DelListener("WRONG")
				assert.False(t, ok, "false positive")
				assert.Nil(t, ln.listener.a, "unexpected %T", ln.listener.a)
				assert.Nil(t, ln.listener.ch, "unexpected %T", ln.listener.ch)
			})
		})
	})

	t.Run("Conn", func(t *testing.T) {
		var c conn

		t.Run("Store", func(t *testing.T) {
			t.Run("New", func(t *testing.T) {
				v, replace := m.SetConn(c, ListenerPath, ConnPath)
				assert.False(t, replace, "unexpected %v (type %T)", v, v)
			})

			t.Run("Replace", func(t *testing.T) {
				cn, replace := m.SetConn(conn{}, ListenerPath, ConnPath)
				assert.True(t, replace, "no replacee or replacement not reported")
				assert.Equal(t, c, cn.conn, "unexpected replacee")
				c = cn.conn // put it back so we can continue testing
			})
		})

		t.Run("Retrieve", func(t *testing.T) {
			t.Run("Existing", func(t *testing.T) {
				cn, ok := m.GetConn(ListenerPath, ConnPath)
				assert.True(t, ok, "retrieval failed OR false negative")
				assert.Equal(t, c, cn.conn, "unexpected replacee")
			})

			t.Run("Missing", func(t *testing.T) {
				_, ok := m.GetConn("WRONG", "PATH")
				assert.False(t, ok, "false positive")
			})
		})

		t.Run("Delete", func(t *testing.T) {
			t.Run("Existing", func(t *testing.T) {
				cn, ok := m.DelConn(ListenerPath, ConnPath)
				assert.True(t, ok, "retrieval failed OR false negative")
				assert.Equal(t, c, cn.conn, "unexpected replacee")
			})

			t.Run("Missing", func(t *testing.T) {
				_, ok := m.DelConn("WRONG", "PATH")
				assert.False(t, ok, "false positive")
			})
		})
	})

	t.Run("Stream", func(t *testing.T) {
		var s stream

		t.Run("Store", func(t *testing.T) {
			t.Run("New", func(t *testing.T) {
				v, replace := m.SetStream(s, ListenerPath, ConnPath, StreamPath)
				assert.False(t, replace, "unexpected %v (type %T)", v, v)
			})

			t.Run("Replace", func(t *testing.T) {
				sn, replace := m.SetStream(stream{}, ListenerPath, ConnPath, StreamPath)
				assert.True(t, replace, "no replacee or replacement not reported")
				assert.Equal(t, s, sn.stream, "unexpected replacee")
				s = sn.stream // put it back so we can continue testing
			})
		})

		t.Run("Retrieve", func(t *testing.T) {
			t.Run("Existing", func(t *testing.T) {
				sn, ok := m.GetStream(ListenerPath, ConnPath, StreamPath)
				assert.True(t, ok, "retrieval failed OR false negative")
				assert.Equal(t, s, sn.stream, "unexpected replacee")
			})

			t.Run("Missing", func(t *testing.T) {
				_, ok := m.GetStream("WRONG", "PATH", "DUDE")
				assert.False(t, ok, "false positive")
			})
		})

		t.Run("Delete", func(t *testing.T) {
			t.Run("Existing", func(t *testing.T) {
				sn, ok := m.DelStream(ListenerPath, ConnPath, StreamPath)
				assert.True(t, ok, "retrieval failed OR false negative")
				assert.Equal(t, s, sn.stream, "unexpected replacee")
			})

			t.Run("Missing", func(t *testing.T) {
				_, ok := m.DelStream("WRONG", "PATH", "DUDE")
				assert.False(t, ok, "false positive")
			})
		})
	})
}

func TestListenNode(t *testing.T) {

}

func TestConnNode(t *testing.T) {

}
