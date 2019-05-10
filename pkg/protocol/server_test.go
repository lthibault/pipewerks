package protocol

import (
	"bytes"
	"context"
	"io"
	"testing"

	synctoolz "github.com/lthibault/toolz/pkg/sync"

	pipe "github.com/lthibault/pipewerks/pkg"
	"github.com/lthibault/pipewerks/pkg/transport/inproc"
	"github.com/stretchr/testify/assert"
)

var d = inproc.New()

func TestServer(t *testing.T) {
	l, err := d.Listen(context.Background(), inproc.Addr("/test"))
	if !assert.NoError(t, err) {
		t.FailNow()
	}
	defer l.Close()

	s := Server{
		Handler: HandlerFunc(func(s pipe.Stream) {
			defer s.Close()

			b := make([]byte, 32)
			n, _ := s.Read(b)
			s.Write(b[:n])
		}),
	}

	go s.Serve(l)

	t.Run("Echo", func(t *testing.T) {
		conn, err := d.Dial(context.Background(), inproc.Addr("/test"))
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		defer conn.Close()

		stream, err := conn.OpenStream()
		if !assert.NoError(t, err) {
			t.FailNow()
		}
		defer stream.Close()

		sync := make(chan struct{})
		go func() {
			defer close(sync)
			stream.Write([]byte("hello, world!"))
		}()

		<-sync
		b := new(bytes.Buffer)
		io.Copy(b, stream)
		assert.Equal(t, "hello, world!", b.String())
	})

	t.Run("Close", func(t *testing.T) {
		assert.NoError(t, s.Close())
	})
}

var c synctoolz.Ctr

func BenchmarkOpenStream(b *testing.B) {
	s := Server{
		Handler: HandlerFunc(func(s pipe.Stream) { c.Decr() }),
	}

	addr := inproc.Addr("/benchmark")
	l, err := d.Listen(context.Background(), addr)
	if !assert.NoError(b, err) {
		b.FailNow()
	}
	defer l.Close()

	go s.Serve(l)

	ctx := context.Background()

	conn, err := d.Dial(ctx, addr)
	if !assert.NoError(b, err) {
		b.FailNow()
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Incr()
		conn.OpenStream()
	}

	for c.Num() != 0 {
	} // spin

	b.StopTimer()
}
