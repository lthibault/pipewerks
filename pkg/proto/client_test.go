package proto

import (
	"bytes"
	"context"
	"io"
	"testing"

	pipe "github.com/lthibault/pipewerks/pkg"
	"github.com/lthibault/pipewerks/pkg/transport/inproc"
	synctoolz "github.com/lthibault/toolz/pkg/sync"
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
	s := StreamCountStrategy{cs: make(map[string]*synctoolz.Var)}

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
		var isCached bool
		conn, isCached, err = s.GetConn(context.Background(), d, inproc.Addr("/test"))
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		assert.False(t, isCached)
	})

	t.Run("GetCached", func(t *testing.T) {
		cached, isCached, err := s.GetConn(context.Background(), d, inproc.Addr("/test"))
		if !assert.NoError(t, err) {
			t.FailNow()
		}

		assert.True(t, isCached)
		assert.Equal(t, conn, cached)
	})

	t.Run("EvictIdle", func(t *testing.T) {
		conn.(*ctrConn).Incr() // simulate stream creation
		conn.(*ctrConn).gc()   // decr and clear cache
		assert.NotContains(t, s.cs, conn)
	})
}

func TestClient(t *testing.T) {
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

	stream, err := c.Connect(context.Background(), inproc.Addr("/test"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}

	b := new(bytes.Buffer)
	io.Copy(b, stream)
	assert.Equal(t, "hello", b.String())
}
