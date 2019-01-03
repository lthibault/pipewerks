package inproc

import (
	"fmt"
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

}
