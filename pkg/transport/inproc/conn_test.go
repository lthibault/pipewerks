package inproc

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
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
