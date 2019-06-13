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

func TestCtrConn(t *testing.T) {
	mc := new(mockConnConnector)
	conn := ctrConn{Conn: mc}

	t.Run("OpenStream", func(t *testing.T) {
		_, _ = conn.OpenStream()
		assert.Equal(t, 1, conn.Ctr.Num())
	})

	t.Run("AcceptStream", func(t *testing.T) {
		_, _ = conn.AcceptStream()
		assert.Equal(t, 2, conn.Ctr.Num())
	})

	t.Run("GarbageCollect", func(t *testing.T) {
		conn.gc()
		conn.gc()
		assert.True(t, mc.closed)
		assert.Zero(t, conn.Ctr.Num())
	})
}

func TestStreamCountStrategy(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	d := inproc.New()
	s := &StreamCountStrategy{}

	l, err := d.Listen(nil, inproc.Addr("/test"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer l.Close()

	go func() {
		for {
			select {
			case <-ctx.Done():
			default:
				l.Accept()
			}
		}
	}()

	var conn pipe.Conn
	t.Run("GetFresh", func(t *testing.T) {
		var err error
		var notCached bool

		s.OnConnOpened = func(pipe.Conn) { notCached = true }

		conn, err = s.GetConn(context.Background(), d, inproc.Addr("/test"))
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		assert.True(t, notCached)
	})

	t.Run("GetCached", func(t *testing.T) {
		var notCached bool
		s.OnConnOpened = func(pipe.Conn) { notCached = true }

		cached, err := s.GetConn(context.Background(), d, inproc.Addr("/test"))
		if !assert.NoError(t, err) {
			t.FailNow()
		}

		assert.False(t, notCached)
		assert.Equal(t, conn, cached)
	})

	t.Run("EvictIdle", func(t *testing.T) {
		conn.(*ctrConn).Incr() // simulate stream creation
		conn.(*ctrConn).gc()   // decr and clear cache

		// assert.NotContains will read the map; lock to avoid triggering race detector
		s.mu.Lock()
		defer s.mu.Unlock()

		assert.NotContains(t, s.cs, conn)
	})
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
