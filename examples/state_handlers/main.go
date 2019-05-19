package main

import (
	"context"
	"encoding/binary"

	"github.com/SentimensRG/ctx/sigctx"
	log "github.com/lthibault/log/pkg"
	pipe "github.com/lthibault/pipewerks/pkg"
	"github.com/lthibault/pipewerks/pkg/proto"
	"github.com/lthibault/pipewerks/pkg/transport/inproc"
)

var (
	t pipe.Transport = inproc.New()
	a                = inproc.Addr("/test")
)

func server(l pipe.Listener) {
	s := proto.Server{
		Handler: proto.HandlerFunc(func(s pipe.Stream) {
			var i int8
			var err error

			if err = binary.Read(s, binary.BigEndian, &i); err != nil {
				log.Fatal(err)
			}

			if err = binary.Write(s, binary.BigEndian, -i); err != nil {
				log.Fatal(err)
			}
		}),
		ConnStateHandler: func(_ pipe.Conn, s proto.ConnState) {
			log.Info(s)
		},
		StreamStateHandler: func(_ pipe.Stream, s proto.StreamState) {
			log.Info(s)
		},
	}

	log.Fatal(s.Serve(l))
}

func client(c *proto.Client, i int8) {
	s, err := c.Connect(context.Background(), a)
	if err != nil {
		log.Fatal(err)
	}
	defer s.Close()

	// log.Printf("sending %d\n", i)
	if err = binary.Write(s, binary.BigEndian, i); err != nil {
		log.Fatal(err)
	}
	if err = binary.Read(s, binary.BigEndian, &i); err != nil {
		log.Fatal(err)
	}
	// log.Printf("recved %d\n", i)
}

func main() {
	l, err := t.Listen(context.Background(), a)
	if err != nil {
		log.Fatal(err)
	}
	defer l.Close()

	go server(l)

	c := &proto.Client{Dialer: t}

	for i := int8(0); i < 5; i++ {
		go client(c, i)
	}

	<-sigctx.New().Done()
}
