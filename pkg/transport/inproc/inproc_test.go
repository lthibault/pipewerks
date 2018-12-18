package inproc

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/lthibault/pipewerks/pkg"

	"github.com/lthibault/pipewerks/pkg/transport/generic"
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
	defer conn.Close()
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	s, err := conn.OpenStream()
	defer s.Close()
	assert.NoError(t, err)

	_, err = io.Copy(s, bytes.NewBuffer([]byte(listenerSends)))
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, io.LimitReader(s, dialerSendSize))
	assert.NoError(t, err)

	assert.Equal(t, dialerSends, buf.String())
}

func dialTest(c context.Context, t *testing.T, wg *sync.WaitGroup, tp pipe.Transport) {
	defer wg.Done()

	conn, err := tp.Dial(context.Background(), Addr("/test"))
	defer conn.Close()
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	s, err := conn.AcceptStream()
	defer s.Close()
	assert.NoError(t, err)

	_, err = io.Copy(s, bytes.NewBuffer([]byte(dialerSends)))
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, io.LimitReader(s, listenerSendSize))
	assert.NoError(t, err)

	assert.Equal(t, listenerSends, buf.String())
}

func TestIntegration(t *testing.T) {
	inproc := New()

	t.Run("Standard", func(t *testing.T) {
		cx, cancel := context.WithTimeout(context.Background(), time.Second*1)
		defer cancel()

		l, err := inproc.Listen(cx, Addr("/test"))
		assert.NoError(t, err)
		assert.NotNil(t, l)
		defer l.Close()

		var wg sync.WaitGroup
		wg.Add(2)
		t.Parallel()

		go t.Run("Listen", func(t *testing.T) {
			listenTest(cx, t, &wg, l)
		})

		go t.Run("Dial", func(t *testing.T) {
			dialTest(cx, t, &wg, inproc)
		})

		wg.Wait()
	})

	t.Run("ConnectHandler", func(t *testing.T) {
		opt := OptGeneric(generic.OptConnectHandler((*testHandler)(t)))
		prev := opt(&inproc)
		defer prev(&inproc)

		cx, cancel := context.WithTimeout(context.Background(), time.Second*1)
		defer cancel()

		l, err := inproc.Listen(cx, Addr("/test"))
		defer l.Close()
		assert.NoError(t, err)
		assert.NotNil(t, l)

		var wg sync.WaitGroup
		wg.Add(2)
		t.Parallel()

		go t.Run("Listen", func(t *testing.T) {
			listenTest(cx, t, &wg, l)
		})

		go t.Run("Dial", func(t *testing.T) {
			dialTest(cx, t, &wg, inproc)
		})

		wg.Wait()
	})

}

type testHandler testing.T

func (t *testHandler) Connected(conn net.Conn, et generic.EndpointType) (net.Conn, error) {
	switch et {
	case generic.ListenEndpoint:
		_, err := io.Copy(conn, bytes.NewBuffer([]byte(dialerSends)))
		assert.NoError(t, err)
	case generic.DialEndpoint:
		b := new(bytes.Buffer)
		_, err := io.Copy(b, io.LimitReader(conn, dialerSendSize))
		assert.NoError(t, err)
		assert.Equal(t, dialerSends, b.String())
	default:
		panic(fmt.Sprintf("unknown endpoint type %v", et))
	}

	return conn, nil
}
