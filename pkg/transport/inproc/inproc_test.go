package inproc

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/lthibault/pipewerks/pkg"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"github.com/stretchr/testify/assert"
)

const (
	dialerSends    = "dialer"
	dialerSendSize = int64(len(dialerSends))

	listenerSends    = "listener"
	listenerSendSize = int64(len(listenerSends))
)

func listenTest(c context.Context, t *testing.T, wg *sync.WaitGroup, l pipe.Listener) {
	defer wg.Done()

	conn, err := l.Accept()
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	defer func() { assert.NoError(t, conn.Close()) }()

	s, err := conn.OpenStream()
	assert.NoError(t, err)
	defer func() { assert.NoError(t, s.Close()) }()

	var g errgroup.Group
	g.Go(func() error {
		_, err := io.Copy(s, bytes.NewBuffer([]byte(listenerSends)))
		return errors.Wrap(err, "listener send")
	})

	g.Go(func() error {
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, io.LimitReader(s, dialerSendSize)); err != nil {
			return errors.Wrap(err, "listener recv")
		}

		assert.Equal(t, dialerSends, buf.String())
		return nil
	})

	assert.NoError(t, g.Wait())
}

func dialTest(c context.Context, t *testing.T, wg *sync.WaitGroup, tp pipe.Transport) {
	defer wg.Done()

	conn, err := tp.Dial(context.Background(), Addr("/test"))
	assert.NoError(t, err)
	assert.NotNil(t, conn)
	defer func() { assert.NoError(t, conn.Close()) }()

	s, err := conn.AcceptStream()
	assert.NoError(t, err)
	defer func() { assert.NoError(t, s.Close()) }()

	var g errgroup.Group

	g.Go(func() error {
		_, err := io.Copy(s, bytes.NewBuffer([]byte(dialerSends)))
		return errors.Wrap(err, "dialer send")
	})

	g.Go(func() error {
		buf := new(bytes.Buffer)
		if _, err := io.Copy(buf, io.LimitReader(s, listenerSendSize)); err != nil {
			return errors.Wrap(err, "dialer recv")
		}

		assert.Equal(t, listenerSends, buf.String())
		return nil
	})

	assert.NoError(t, g.Wait())
	<-time.After(time.Millisecond) // give the listener time to read
}

func integrationTest(c context.Context, t *testing.T, tp pipe.Transport, addr string) {
	l, err := tp.Listen(c, Addr(addr))
	assert.NoError(t, err)
	assert.NotNil(t, l)
	defer func() { assert.NoError(t, l.Close()) }()

	var wg sync.WaitGroup
	wg.Add(2)
	go listenTest(c, t, &wg, l)
	go dialTest(c, t, &wg, tp)
	wg.Wait()
}
func TestItegration(t *testing.T) {
	integrationTest(context.Background(), t, New(), "/test")
}

func TestIntegrationParallel(t *testing.T) {
	inproc := New()

	t.Run("Parallel", func(t *testing.T) {
		t.Parallel()
		n := 2

		l, err := inproc.Listen(context.Background(), Addr("/test/parallel/listen"))
		assert.NoError(t, err)
		assert.NotNil(t, l)

		go t.Run("Server", func(t *testing.T) {
			defer func() { assert.NoError(t, l.Close()) }()

			var wg sync.WaitGroup
			wg.Add(n)

			for i := 0; i < n; i++ {
				conn, err := l.Accept()
				assert.NoError(t, err)
				defer conn.Close()

				t.Fatal(conn)

				s, err := conn.AcceptStream()
				assert.NoError(t, err)

				go func() {
					defer s.Close()

					var payload int8
					assert.NoError(t, binary.Read(s, binary.BigEndian, &payload))
					assert.NoError(t, binary.Write(s, binary.BigEndian, payload))
				}()

				wg.Wait()
				<-time.After(time.Millisecond * 10) // TODO: try removing when working
			}
		})

		// Clients
		var wg sync.WaitGroup
		wg.Add(n)

		for i := 0; i < n; i++ {
			name := fmt.Sprintf("client-%d", i)

			func(i int8) {
				go t.Run(name, func(t *testing.T) {
					defer wg.Done()

					conn, err := inproc.Dial(context.Background(), Addr("/test/parallel/listen"))
					assert.NoError(t, err)
					defer conn.Close()

					s, err := conn.OpenStream()
					assert.NoError(t, err)
					defer s.Close()

					var echo int8
					assert.NoError(t, binary.Write(s, binary.BigEndian, i))
					assert.NoError(t, binary.Read(s, binary.BigEndian, &echo))
					assert.Equal(t, i, echo)
				})
			}(int8(i))
		}

		wg.Wait()
	})
}
