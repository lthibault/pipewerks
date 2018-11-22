package inproc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

const ListenerPath = "listener"

func TestMux(t *testing.T) {
	m := newMux()
	assert.Empty(t, m.r.ToMap())

	t.Run("Listen", func(t *testing.T) {
		_, err := m.Listen(context.Background(), "inproc", ListenerPath)
		assert.NoError(t, err)

		t.Run("Retrieve", func(t *testing.T) {
			ln, ok := m.GetListener(ListenerPath)
			assert.True(t, ok, "retrieval failed OR false negative")
			assert.Equal(t, ln.Addr().String(), ListenerPath, "wrong listener")
		})

		// t.Run("Dial", func(t *testing.T) {
		// 	_, err := m.DialContext(context.Background(), "inproc", ListenerPath)
		// 	assert.NoError(t, err)
		// })

		t.Run("Delete", func(t *testing.T) {
			t.Run("Existing", func(t *testing.T) {
				m.Unbind(ListenerPath)
				assert.NotContains(t, m.r.ToMap(), ListenerPath)
			})
		})
	})

}
