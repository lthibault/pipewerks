package inproc

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"
	"time"

	pipe "github.com/lthibault/pipewerks/pkg"
	"github.com/stretchr/testify/assert"
)

const (
	dialerSends    = "dialer"
	dialerSendSize = int64(len(dialerSends))

	listenerSends    = "listener"
	listenerSendSize = int64(len(listenerSends))
)

func runListenerTest(cx context.Context, t *testing.T, inproc pipe.Transport) {
	l, err := inproc.Listen(cx, Addr("/test"))
	assert.NoError(t, err)
	assert.NotNil(t, l)

	conn, err := l.Accept()
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	s, err := conn.OpenStream()
	assert.NoError(t, err)

	_, err = io.Copy(s, bytes.NewBuffer([]byte(listenerSends)))
	assert.NoError(t, err)

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, io.LimitReader(s, dialerSendSize))
	assert.NoError(t, err)

	assert.Equal(t, dialerSends, buf.String())
}

func runDialerTest(cx context.Context, t *testing.T, inproc pipe.Transport) {
	conn, err := inproc.Dial(context.Background(), Addr("/test"))
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	s, err := conn.AcceptStream()
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

		var wg sync.WaitGroup
		wg.Add(2)
		t.Parallel()

		go t.Run("Listen", func(t *testing.T) {
			defer wg.Done()
			runListenerTest(cx, t, inproc)
		})

		go t.Run("Dial", func(t *testing.T) {
			defer wg.Done()
			runDialerTest(cx, t, inproc)
		})

		wg.Wait()
	})

	// t.Run("ConnectHandler", func(t *testing.T) {
	// 	h := (*testHandler)(t)
	// 	inproc.Set(h)
	// 	defer inproc.Rm(h)

	// 	cx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	// 	defer cancel()

	// 	t.Parallel()

	// 	go t.Run("Listen", func(t *testing.T) {
	// 	})

	// 	go t.Run("Dial", func(t *testing.T) {

	// 	})
	// })

}

// type testHandler testing.T

// func (t testHandler) Connected(conn net.Conn, et generic.EndpointType) (net.Conn, error) {
// 	switch et {
// 	case generic.ListenEndpoint:
// 		_, err := io.Copy(s, bytes.NewBuffer([]byte(dialerSends)))
// 		assert.NoError(t, err)

// 	case generic.DialEndpoint:
// 	default:
// 		panic(fmt.Sprintf("unknown endpoint type %d", et))
// 	}

// 	return conn, nil
// }
