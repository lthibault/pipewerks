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

var d = inproc.New()

func TestServerIntegration(t *testing.T) {
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
