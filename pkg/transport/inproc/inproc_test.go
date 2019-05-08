package inproc

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"
	"time"

	pipe "github.com/lthibault/pipewerks/pkg"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
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

func TestItegration(t *testing.T) {
	tp := New()
	c := context.Background()

	l, err := tp.Listen(c, Addr("/test"))
	assert.NoError(t, err)
	assert.NotNil(t, l)
	defer func() { assert.NoError(t, l.Close()) }()

	var wg sync.WaitGroup
	wg.Add(2)
	go listenTest(c, t, &wg, l)
	go dialTest(c, t, &wg, tp)
	wg.Wait()
}

var res = make([]byte, 5)

func BenchmarkTransmission(b *testing.B) {
	t := New(OptNamespace(make(namespace)))

	l, err := t.Listen(context.Background(), Addr("/bench"))
	if err != nil {
		b.Error(err)
	}

	go func() {

		svrConn, err := l.Accept()
		if err != nil {
			b.Error(err)
		}

		svrStream, err := svrConn.AcceptStream()
		if err != nil {
			b.Error(err)
		}

		for {
			if _, err := svrStream.Read(res); err != nil {
				b.Error(err)
			}
		}
	}()

	cltConn, err := t.Dial(context.Background(), Addr("/bench"))
	if err != nil {
		b.Error(err)
	}

	cltStream, err := cltConn.OpenStream()
	if err != nil {
		b.Error(err)
	}

	snd := []byte("hello")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := cltStream.Write(snd); err != nil {
			b.Error(err)
		}
	}
}
