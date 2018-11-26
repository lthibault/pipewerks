package main

import (
	"bytes"
	"context"
	"io"
	"log"
	"sync"

	"github.com/lthibault/pipewerks/pkg/transport/inproc"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

var (
	lock sync.Mutex
	t    = inproc.New()
	a    = inproc.Addr("/test")
	c    = context.Background()
)

func main() {
	g, c := errgroup.WithContext(c)

	l, err := t.Listen(c, a)
	if err != nil {
		log.Fatal("listen:", err)
	}
	defer l.Close()

	g.Go(func() error {
		conn, err := l.Accept(c)
		if err != nil {
			return errors.Wrap(err, "accept")
		}
		defer conn.Close()

		s, err := conn.Stream().Accept()
		if err != nil {
			return errors.Wrap(err, "accept stream")
		}
		defer s.Close()

		ch := make(chan error)
		go func() {
			defer close(ch)
			b := new(bytes.Buffer)
			if _, err := io.Copy(b, s); err != nil {
				ch <- errors.Wrap(err, "read bytes")
			}
			log.Println(b.String())
		}()

		select {
		case <-c.Done():
			return c.Err()
		case err := <-ch:
			return err
		}
	})

	g.Go(func() error {
		conn, err := t.Dial(c, a)
		if err != nil {
			return errors.Wrap(err, "dial")
		}
		defer conn.Close()

		s, err := conn.Stream().Open()
		if err != nil {
			return errors.Wrap(err, "open stream")
		}
		defer s.Close()

		if _, err = io.Copy(s, bytes.NewBufferString("hello, world!")); err != nil {
			return errors.Wrap(err, "write bytes")
		}

		return nil
	})

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}
