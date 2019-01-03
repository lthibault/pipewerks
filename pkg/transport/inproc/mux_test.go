package inproc

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMux(t *testing.T) {
	m := newMux()

	t.Run("GC", func(t *testing.T) {
		n := 100

		gc := make([]func(), n)
		for i := 0; i < n; i++ {
			addr := fmt.Sprintf("/%d", i)
			m.m[addr] = &listener{}
			gc[i] = m.gc(addr)
		}

		var wg sync.WaitGroup
		wg.Add(n)

		for _, f := range gc {
			go func(release func()) {
				defer wg.Done()
				release()
			}(f)
		}
		wg.Wait()

		assert.Empty(t, m.m)
	})

	t.Run("Listen", func(t *testing.T) {
		t.Run("CheckNetwork", func(t *testing.T) {
			_, err := m.Listen(context.Background(), "invalid", "")
			assert.Error(t, err)
		})
	})

	t.Run("Dial", func(t *testing.T) {
		t.Run("CheckNetwork", func(t *testing.T) {
			_, err := m.DialContext(context.Background(), "invalid", "")
			assert.Error(t, err)
		})

		t.Run("NoListener", func(t *testing.T) {
			_, err := m.DialContext(context.Background(), "", "/fail")
			assert.Error(t, err)
		})

		t.Run("ContextExpired", func(t *testing.T) {
			c, cancel := context.WithCancel(context.Background())
			cancel()

			m.m["/fail"] = &listener{}
			defer delete(m.m, "/fail")

			_, err := m.DialContext(c, "", "/fail")
			assert.EqualError(t, err, context.Canceled.Error())
		})
	})
}

func TestAddrOverride(t *testing.T) {
	local := Addr("/local")
	remote := Addr("/remote")
	_, conn := net.Pipe()

	c := addrOverride{Conn: conn, local: local, remote: remote}
	assert.Equal(t, remote, c.RemoteAddr())
	assert.Equal(t, local, c.LocalAddr())
}
