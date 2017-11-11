package supernova

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"
)

// GracefulListener is used as custom listener to watch connections
type GracefulListener struct {
	// inner listener
	ln net.Listener

	// this channel is closed during graceful shutdown on zero open connections.
	done chan struct{}

	// the number of open connections
	connCount uint64

	// becomes non-zero when graceful shutdown starts
	shutdown uint64
}

// NewGracefulListener wraps the given listener into 'graceful shutdown' listener.
func NewGracefulListener(ln net.Listener) net.Listener {
	listener := &GracefulListener{
		ln:   ln,
		done: make(chan struct{}),
	}

	return listener
}

// Accept waits for connection increments count and returns to the listener.
func (ln *GracefulListener) Accept() (net.Conn, error) {
	c, err := ln.ln.Accept()
	if err != nil {
		return nil, err
	}

	atomic.AddUint64(&ln.connCount, 1)
	return &gracefulConn{
		Conn: c,
		ln:   ln,
	}, nil
}

// Close closes the inner listener and waits until all the pending open connections
// are closed before returning.
func (ln *GracefulListener) Close() error {
	err := ln.ln.Close()
	if err != nil {
		return nil
	}

	return ln.waitForZeroConns()
}

// CloseTimeout closes the inner listener and waits for all pending open connections to close
// or timeout before returning
func (ln *GracefulListener) CloseTimeout(t time.Duration) error {
	err := ln.ln.Close()
	if err != nil {
		return nil
	}
	return ln.waitForZeroConnTimeout(t)
}

// Addr returns the listener's network address.
func (ln *GracefulListener) Addr() net.Addr {
	return ln.ln.Addr()
}

// waitForZeroConns will wait forever for the connections to close
func (ln *GracefulListener) waitForZeroConns() error {
	atomic.AddUint64(&ln.shutdown, 1)

	if atomic.LoadUint64(&ln.connCount) == 0 {
		close(ln.done)
		return nil
	}

	fmt.Printf("Waiting on %d connections\n", ln.connCount)
	select {
	case <-ln.done:
		return nil
	}
}

// waitForZerConnTimeout will wait for all connections to close or timeout
func (ln *GracefulListener) waitForZeroConnTimeout(t time.Duration) error {
	atomic.AddUint64(&ln.shutdown, 1)

	if atomic.LoadUint64(&ln.connCount) == 0 {
		close(ln.done)
		return nil
	}

	fmt.Printf("Waiting on %d connections\n", ln.connCount)

	select {
	case <-ln.done:
		return nil
	case <-time.After(t):
		return fmt.Errorf("max timeout reached after %s", t)
	}
}

func (ln *GracefulListener) closeConn() {
	connCount := atomic.AddUint64(&ln.connCount, ^uint64(0))
	if atomic.LoadUint64(&ln.shutdown) != 0 && connCount == 0 {
		close(ln.done)
	}
}

type gracefulConn struct {
	net.Conn
	ln *GracefulListener
}

// Close starts listener shutdown
func (c *gracefulConn) Close() error {
	err := c.Conn.Close()
	if err != nil {
		return err
	}
	c.ln.closeConn()
	return nil
}
