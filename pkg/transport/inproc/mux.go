package inproc

import (
	"context"
	"errors"
	"strings"
	"sync"
	"unsafe"

	radix "github.com/armon/go-radix"
	net "github.com/lthibault/pipewerks/pkg/net"
)

const (
	sep    = '/'
	sepLen = 1
)

type listenNode struct {
	path string
	r    *radixMux
	listener
}

func (l listenNode) Close() error {
	_, ok := l.r.DelListener(l.path)
	if !ok {
		return errors.New("not found")
	}

	l.IterConn(func(c net.Conn) (abort bool) {
		c.Close()
		return
	})

	return l.listener.Close()
}

func (l listenNode) GetConn(path string) (connNode, bool) {
	return l.r.GetConn(l.path, path)
}

func (l listenNode) IterConn(fn func(net.Conn) bool) {
	(*radix.Tree)(unsafe.Pointer(l.r)).WalkPrefix(l.path, func(_ string, v interface{}) bool {
		if c, ok := v.(net.Conn); ok {
			return fn(c)
		}
		return true // abort
	})
}

type connNode struct {
	path [2]string
	r    *radixMux
	conn
}

func (c connNode) Close() error {
	_, ok := c.r.DelConn(c.path[0], c.path[1])
	if !ok {
		return errors.New("not found")
	}

	c.IterStreams(func(s net.Stream) (abort bool) {
		s.Close()
		return
	})

	return c.conn.Close()
}

func (c connNode) GetStream(path string) (streamNode, bool) {
	return c.r.GetStream(c.path[0], c.path[1], path)
}

func (c connNode) IterStreams(fn func(net.Stream) bool) {
	path := c.r.join(c.path[0], c.path[1])
	(*radix.Tree)(unsafe.Pointer(c.r)).WalkPrefix(path, func(_ string, v interface{}) bool {
		if s, ok := v.(net.Stream); ok {
			return fn(s)
		}
		return true // abort
	})
}

type streamNode struct {
	path [3]string
	stream
}

type radixMux struct {
	sync.RWMutex
	r *radix.Tree
}

func newMux() radixMux { return radixMux{r: radix.New()} }

func (r *radixMux) GetListener(path string) (l listenNode, ok bool) {
	r.RLock()
	var v interface{}
	if v, ok = r.r.Get(path); ok {
		l = v.(listenNode)
	}
	r.RUnlock()
	return
}

func (r *radixMux) GetConn(lpath, cpath string) (c connNode, ok bool) {
	r.RLock()
	var v interface{}
	if v, ok = r.r.Get(r.join(lpath, cpath)); ok {
		c = v.(connNode)
	}
	r.RUnlock()
	return
}

func (r *radixMux) GetStream(lpath, cpath, spath string) (c streamNode, ok bool) {
	r.RLock()
	var v interface{}
	if v, ok = r.r.Get(r.join(lpath, cpath, spath)); ok {
		c = v.(streamNode)
	}
	r.RUnlock()
	return
}

func (r *radixMux) SetListener(l listener, path string) (ln listenNode, ok bool) {
	r.Lock()
	ln = listenNode{path: path, r: r, listener: l}
	var v interface{}
	if v, ok = r.r.Insert(path, ln); ok {
		ln = v.(listenNode)
	}
	r.Unlock()
	return
}

func (r *radixMux) SetConn(c conn, lpath, cpath string) (cn connNode, ok bool) {
	r.Lock()
	cn = connNode{path: [2]string{lpath, cpath}, r: r, conn: c}
	var v interface{}
	if v, ok = r.r.Insert(r.join(lpath, cpath), cn); ok {
		cn = v.(connNode)
	}
	r.Unlock()
	return
}

func (r *radixMux) SetStream(s stream, lpath, cpath, spath string) (c streamNode, ok bool) {
	r.Lock()
	cn := streamNode{path: [3]string{lpath, cpath, spath}, stream: s}
	var v interface{}
	if v, ok = r.r.Insert(r.join(lpath, cpath, spath), cn); ok {
		c = v.(streamNode)
	}
	r.Unlock()
	return
}

func (r *radixMux) DelListener(path string) (l listenNode, ok bool) {
	r.Lock()
	var v interface{}
	if v, ok = r.r.Delete(path); ok {
		l = v.(listenNode)
	}
	r.Unlock()
	return
}

func (r *radixMux) DelConn(lpath, cpath string) (c connNode, ok bool) {
	r.Lock()
	var v interface{}
	if v, ok = r.r.Delete(r.join(lpath, cpath)); ok {
		c = v.(connNode)
	}
	r.Unlock()
	return
}

func (r *radixMux) DelStream(lpath, cpath, spath string) (c streamNode, ok bool) {
	r.Lock()
	var v interface{}
	if v, ok = r.r.Delete(r.join(lpath, cpath, spath)); ok {
		c = v.(streamNode)
	}
	r.Unlock()
	return
}

func (r *radixMux) ServeConn(c context.Context, conn net.Conn) (err error) {
	r.RLock()
	l, ok := r.GetListener(conn.Endpoint().Local().String())
	r.RUnlock()

	if ok {
		err = errors.New("connection refused")
	} else {
		select {
		case l.listener.ch <- conn:
		case <-c.Done():
			err = c.Err()
		}
	}

	return
}

func (r *radixMux) join(parts ...string) string {
	var sb strings.Builder
	for _, p := range parts {
		sb.Grow(len(p) + sepLen)
		sb.WriteString(p)
		sb.WriteRune(sep)
	}
	return sb.String()
}
