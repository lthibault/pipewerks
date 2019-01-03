package inproc

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestListener(t *testing.T) {
	t.Run("Addr", func(t *testing.T) {
		a := Addr("/test")
		l := newListener(a, func() {})
		assert.Equal(t, a, l.Addr())
	})

	t.Run("Close", func(t *testing.T) {
		var called bool
		l := newListener(Addr("/test"), func() { called = true })

		t.Run("FirstCall", func(t *testing.T) {
			assert.NoError(t, l.Close())
			assert.True(t, called, "release function not called")
			assert.Panics(t, func() { l.ch <- nil }) // ch is closed
			assert.Panics(t, func() { close(l.cq) }) // cq is closed
		})

		t.Run("SubsequentCalls", func(t *testing.T) {
			assert.Error(t, l.Close())
		})

	})

	t.Run("Connect", func(t *testing.T) {
		l := newListener(Addr("/test"), func() {})

		t.Run("Success", func(t *testing.T) {
			go func() { <-l.ch }()
			assert.NoError(t, l.connect(context.Background(), nil))
		})

		t.Run("DialContextExpired", func(t *testing.T) {
			c, cancel := context.WithCancel(context.Background())
			cancel()

			cq := make(chan struct{})
			defer close(cq)
			go func() {
				select {
				case <-l.ch:
				case <-cq:
				}
			}()

			assert.EqualError(t, l.connect(c, nil), context.Canceled.Error())
		})

		t.Run("Closed", func(t *testing.T) {
			l.Close()

			cq := make(chan struct{})
			defer close(cq)
			go func() {
				select {
				case <-l.ch:
				case <-cq:
				}
			}()

			assert.EqualError(t, l.connect(context.Background(), nil), "connection refused")
		})
	})

	t.Run("Accept", func(t *testing.T) {
		l := newListener(Addr("/test"), func() {})

		t.Run("Success", func(t *testing.T) {
			go func() { l.ch <- nil }()

			_, err := l.Accept()
			assert.NoError(t, err)
		})

		t.Run("Closed", func(t *testing.T) {
			t.Run("WhileAccepting", func(t *testing.T) {
				ch := make(chan error)
				go func() {
					_, err := l.Accept()
					ch <- err
				}()

				<-time.After(time.Millisecond) // wait for goroutine to block

				l.Close()
				assert.Error(t, <-ch)
			})

			t.Run("BeforeAccepting", func(t *testing.T) {
				_, err := l.Accept()
				assert.Error(t, err)
			})
		})
	})
}
