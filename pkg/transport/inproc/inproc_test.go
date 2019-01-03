package inproc

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"

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

func TestIntegration(t *testing.T) {
	inproc := New()

	t.Run("Single", func(t *testing.T) {
		integrationTest(context.Background(), t, inproc, "/test")
	})

	// t.Run("Parallel", func(t *testing.T) {
	// 	n := 100
	// 	var wg sync.WaitGroup
	// 	wg.Add(n)

	// 	t.Parallel()

	// 	for i := 0; i < n; i++ {
	// 		func(i int) {
	// 			defer wg.Done()
	// 			t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
	// 				l, err := inproc.Listen(cx, Addr(fmt.Sprintf("/%d", i)))
	// 				assert.NoError(t, err)
	// 				assert.NotNil(t, l)
	// 				defer l.Close()

	// 				var wg sync.WaitGroup
	// 				wg.Add(2)
	// 				go listenTest(cx, t, &wg, l)
	// 				go dialTest(cx, t, &wg, inproc)
	// 				wg.Wait()
	// 			})
	// 		}(i)
	// 	}

	// 	wg.Wait()

	// })
}
