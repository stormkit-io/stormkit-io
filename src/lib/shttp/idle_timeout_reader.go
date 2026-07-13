package shttp

import (
	"context"
	"io"
	"sync"
	"time"
)

type idleTimeoutReader struct {
	r       io.ReadCloser
	cancel  context.CancelFunc
	timer   *time.Timer
	timeout time.Duration
	mu      sync.Mutex
	active  bool
}

// NewIdleTimeoutReader wraps r so that a rolling idle deadline is enforced via
// context cancellation. If no bytes arrive within timeout between reads, cancel
// is called and the underlying reader is closed to interrupt any blocked Read.
// Works for both HTTP/1.1 and HTTP/2. Equivalent to nginx's client_body_timeout.
func NewIdleTimeoutReader(r io.ReadCloser, cancel context.CancelFunc, timeout time.Duration) io.ReadCloser {
	itr := &idleTimeoutReader{
		r:       r,
		cancel:  cancel,
		timeout: timeout,
		active:  true,
	}

	itr.timer = time.AfterFunc(timeout, func() {
		itr.mu.Lock()
		defer itr.mu.Unlock()

		if itr.active {
			itr.cancel()
			itr.r.Close()
		}
	})

	return itr
}

func (r *idleTimeoutReader) arm() {
	r.mu.Lock()
	r.active = true
	r.timer.Reset(r.timeout)
	r.mu.Unlock()
}

func (r *idleTimeoutReader) disarm() {
	r.mu.Lock()
	r.active = false
	r.timer.Stop()
	r.mu.Unlock()
}

func (r *idleTimeoutReader) Read(p []byte) (int, error) {
	r.arm()
	n, err := r.r.Read(p)
	r.disarm()
	return n, err
}

func (r *idleTimeoutReader) Close() error {
	r.disarm()
	return r.r.Close()
}
