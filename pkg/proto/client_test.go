package proto

import (
	"bytes"
	"context"
	"io"
	"testing"

	pipe "github.com/lthibault/pipewerks/pkg"
	"github.com/lthibault/pipewerks/pkg/transport/inproc"
	"github.com/stretchr/testify/assert"
)

type mockStreamCloser struct {
	pipe.Stream
	closer func() error
}

func (s mockStreamCloser) Close() error { return s.closer() }

func TestCtrStream(t *testing.T) {
	var gcFlag, closerFlag bool
	s := ctrStream{
		Stream: mockStreamCloser{closer: func() error {
			closerFlag = true
			return nil
		}},
		done: func() { gcFlag = true },
	}

	s.Close()
	assert.True(t, gcFlag)
	assert.True(t, closerFlag)
}

type mockConnConnector struct {
	pipe.Conn
	closed bool
}

func (c mockConnConnector) AcceptStream() (pipe.Stream, error) { return nil, nil }
func (c mockConnConnector) OpenStream() (pipe.Stream, error)   { return nil, nil }
func (c *mockConnConnector) Close() error {
	c.closed = true
	return nil
}

func TestClient(t *testing.T) {
	t.Run("Integration", func(t *testing.T) {
		d := inproc.New()
		c := &Client{Dialer: d}

		l, err := d.Listen(nil, inproc.Addr("/test"))
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		defer l.Close()

		var s Server
		s.Handler = HandlerFunc(func(s pipe.Stream) {
			defer s.Close()
			s.Write([]byte("hello"))
		})
		go s.Serve(l)

		t.Run("Sync", func(t *testing.T) {
			stream, err := c.Connect(context.Background(), inproc.Addr("/test"))
			if !assert.NoError(t, err) {
				t.FailNow()
			}

			b := new(bytes.Buffer)
			io.Copy(b, stream)
			assert.Equal(t, "hello", b.String())
		})

		t.Run("Async", func(t *testing.T) {
			t.Parallel()

			go t.Run("0", func(t *testing.T) {
				stream, err := c.Connect(context.Background(), inproc.Addr("/test"))
				if !assert.NoError(t, err) {
					t.FailNow()
				}

				b := new(bytes.Buffer)
				io.Copy(b, stream)
				assert.Equal(t, "hello", b.String())
			})

			go t.Run("1", func(t *testing.T) {
				stream, err := c.Connect(context.Background(), inproc.Addr("/test"))
				if !assert.NoError(t, err) {
					t.FailNow()
				}

				b := new(bytes.Buffer)
				io.Copy(b, stream)
				assert.Equal(t, "hello", b.String())
			})

		})
	})
}
