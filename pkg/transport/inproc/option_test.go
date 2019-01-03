package inproc

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDialback(t *testing.T) {
	var c context.Context
	a := Addr("/dialback")

	t.Run("SetDialback", func(t *testing.T) {
		c = SetDialback(context.Background(), a)
		assert.Equal(t, a, c.Value(keyDialback).(Addr))
	})

	t.Run("GetDialback", func(t *testing.T) {
		assert.Equal(t, a, getDialback(c))
	})
}
