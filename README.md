# Pipewerks

[![Godoc Reference](https://img.shields.io/badge/godoc-reference-blue.svg?style=flat-square)](https://godoc.org/github.com/lthibault/pipewerks/pkg) [![Go Report Card](https://goreportcard.com/badge/github.com/lthibault/pipewerks?style=flat-square)](https://goreportcard.com/report/github.com/lthibault/pipewerks)


A simple library for working with reliable transports.

## Installation

```bash
go get -u github.com/lthibault/pipewerks
```

## Motivation

Go's `net` package is too low-level for most applications.  Pipewerks is motivated from
the desire to write modular networking code, where transports are trivially interchangeable.

`pipewerks` assumes you are looking for reliable delivery semantics (no UDP here, sorry), and that your application is best modeled as logical streams multiplexed on top of sessions.  As such, it provides uniform interfaces (`pipe.Conn` and `pipe.Stream`) for all transports.

That's right! Pipewerks comes with stream mulitplexing out-of-the box for _all_ protocols!

## Ambition

Pipewerks' ambition is to be a standard library for reliable, multiplexed transports.

It aims to compete with the standard library in terms of productivity.  We hope to
see developers reaching for `pipewerks` first, dropping down to Go's `net` package only
if/when needed.

In order to achieve this, `pipewerks` is designed according to the following principles:

1. **Full compatability with Go's standard library**:  `pipewerks` is built on top of Go's `net` package, and offers unbridled access to the underlying standard library objects.

2. **Transport Modularity**:  `pipewerks` makes it easy to swap out transports.  You can prototype your application using `inproc`, and then deploy it using `tcp` or `quic`.

3. **Transport Uniformity**:  `pipewerks` believes application developers shouldn't care whether their bytes are delivered by TCP, QUIC, µTP, or carrier pigeons.  Pipewerks lets you code your application with generic interfaces like `Conn` and `Stream`.

## Example

```go
import (
    "context"

    pipe "github.com/lthibault/pipewerks/pkg/"
    "github.com/lthibault/pipewerks/pkg/transport/inproc"
)


func startServer(c context.Context, l pipe.Listener) {
    for {
        conn, _ := l.Accept()

        go func() {
            for {
                stream, _ := conn.AcceptStream()

                // stream is a standard library net.Conn
                // do something with stream ...
            }
        }()
    }
}

func startClient(c context.Context, t pipe.Transport, a net.Addr) {
    conn, _ := t.Dial(c, a)
    stream, _ := conn.OpenStream()

    // do something with stream ...
}

func main() {

    // Create a new in-process transport.
    // Using a different wire protocol is as simple as replacing this with
    // e.g. tcp.New().
    t := inproc.New()

    // inproc.Addr is a net.Addr ...
    addr := inproc.Addr("/foo")

    // ... which is nice because all pipewerks transports accept net.Addr.
    l, _ := t.Listen(c, a)

    // Business logic ...
    go startServer(context.Background(), l)
    startClient(context.Background(), t, addr)
}
```

## Supported Transports

The following wire protocols are implemented or planned.

- [x] In-Process
- [x] [TCP](https://en.wikipedia.org/wiki/Transmission_Control_Protocol)
- [x] [Unix domain socket](https://en.wikipedia.org/wiki/Unix_domain_socket)
- [x] [QUIC](https://en.wikipedia.org/wiki/QUIC)
- [ ] [µTP](https://en.wikipedia.org/wiki/Micro_Transport_Protocol)
- [ ] [KCP](https://github.com/xtaci/kcp-go)

In addition, a `generic` transport is provided to facilitate the writing of new transport types.
