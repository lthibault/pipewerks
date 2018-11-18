package inproc

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const ListenerPath = "listener"

func TestMux(t *testing.T) {
	m := newMux()
	assert.Empty(t, m.r.ToMap())

	t.Run("Listener", func(t *testing.T) {
		var l Listener
		l.a = ListenerPath

		t.Run("Store", func(t *testing.T) {
			t.Run("New", func(t *testing.T) {
				v, replace := m.Bind(l)
				assert.False(t, replace, "unexpected %v (type %T)", v, v)
			})

			t.Run("Replace", func(t *testing.T) {
				ln, replace := m.Bind(Listener{a: ListenerPath})
				assert.True(t, replace, "no replacee or replacement not reported")
				assert.Equal(t, l, ln, "unexpected replacee")
				l = ln // put it back so we can continue testing
			})
		})

		t.Run("Retrieve", func(t *testing.T) {
			t.Run("Existing", func(t *testing.T) {
				ln, ok := m.GetListener(ListenerPath)
				assert.True(t, ok, "retrieval failed OR false negative")
				assert.Equal(t, l, ln, "unexpected replacee")
			})

			t.Run("Missing", func(t *testing.T) {
				_, ok := m.GetListener("WRONG")
				assert.False(t, ok, "false positive")
			})
		})

		t.Run("Delete", func(t *testing.T) {
			t.Run("Existing", func(t *testing.T) {
				ln, ok := m.DelListener(ListenerPath)
				assert.True(t, ok, "retrieval failed OR false negative")
				assert.Equal(t, l, ln, "unexpected replacee")
			})

			t.Run("Missing", func(t *testing.T) {
				_, ok := m.DelListener("WRONG")
				assert.False(t, ok, "false positive")
			})
		})
	})
}
