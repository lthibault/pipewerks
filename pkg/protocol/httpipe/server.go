package httpipe

import (
	"errors"
	"net"
	"net/http"
	"sync"

	"github.com/SentimensRG/ctx"
	pipe "github.com/lthibault/pipewerks/pkg"
)

type listener struct {
	init, shutdown sync.Once
	pipe.Listener
	cq    chan struct{}
	cherr chan error
	chstr chan pipe.Stream
}

func (l *listener) Close() error {
	l.shutdown.Do(func() {
		close(l.cq)
		close(l.cherr)
		close(l.chstr)
	})

	return l.Listener.Close()
}

func (l *listener) Accept() (conn net.Conn, err error) {
	l.init.Do(func() {
		l.cq = make(chan struct{})
		l.cherr = make(chan error)
		l.chstr = make(chan pipe.Stream)

		go ctx.FTick(ctx.C(l.cq), func() {
			var err error
			var pconn pipe.Conn

			if pconn, err = l.Listener.Accept(); err != nil {
				l.reportErr(err)
			}

			go ctx.FTick(pconn.Context(), l.handleConn(pconn))
		})
	})

	select {
	case err = <-l.cherr:
	case conn = <-l.chstr:
	case <-l.cq:
		err = errors.New("closed")
	}

	return
}

func (l *listener) reportErr(err error) {
	select {
	case <-l.cq:
	case l.cherr <- err:
	}
}

func (l *listener) handleConn(pconn pipe.Conn) func() {
	return func() {
		stream, err := pconn.AcceptStream()
		if err != nil {
			l.reportErr(err)
		}

		select {
		case <-l.cq:
		case l.chstr <- stream:
		}
	}
}

// A Server defines parameters for running an HTTP Server.
type Server struct{ *http.Server }

// Serve HTTP from the specified Listener.  See http.Server.Serve
func (s Server) Serve(l pipe.Listener) error {
	return s.Server.Serve(&listener{Listener: l})
}
