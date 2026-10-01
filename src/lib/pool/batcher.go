package pool

import (
	"errors"
	"io"
	"time"
)

var (
	// ErrTimeout indicates an operation has timed out.
	ErrTimeout = errors.New("operation timed-out")
	// ErrClosed indicates the buffer is closed and can no longer be used.
	ErrClosed = errors.New("buffer is closed")
)

type (
	// Buffer represents a data buffer that is asynchronously flushed, either manually or automatically.
	Buffer struct {
		io.Closer
		dataCh  chan any
		flushCh chan struct{}
		closeCh chan struct{}
		doneCh  chan struct{}
		items   []any
		options *Options
	}
)

// Push appends an item to the end of the buffer.
//
// It returns an ErrTimeout if if cannot be performed in a timely fashion, and
// an ErrClosed if the buffer has been closed.
func (buffer *Buffer) Push(item any) error {
	if buffer.closed() {
		return ErrClosed
	}

	select {
	case buffer.dataCh <- item:
		return nil
	case <-time.After(buffer.options.PushTimeout):
		return ErrTimeout
	}
}

// Items returns the currently inserted items in the buffer.
func (buffer *Buffer) Items(at int) any {
	if buffer.closed() {
		return nil
	}

	return buffer.items[at]
}

// Flush outputs the buffer to a permanent destination.
//
// It returns an ErrTimeout if if cannot be performed in a timely fashion, and
// an ErrClosed if the buffer has been closed.
func (buffer *Buffer) Flush() error {
	if buffer.closed() {
		return ErrClosed
	}

	select {
	case buffer.flushCh <- struct{}{}:
		return nil
	case <-time.After(buffer.options.FlushTimeout):
		return ErrTimeout
	}
}

// Close flushes the buffer and prevents it from being further used.
//
// It returns an ErrTimeout if if cannot be performed in a timely fashion, and
// an ErrClosed if the buffer has already been closed.
//
// An ErrTimeout can either mean that a flush could not be triggered, or it can
// mean that a flush was triggered but it has not finished yet. In any case it is
// safe to call Close again.
func (buffer *Buffer) Close() error {
	if buffer.closed() {
		return ErrClosed
	}

	select {
	case buffer.closeCh <- struct{}{}:
		// noop
	case <-time.After(buffer.options.CloseTimeout):
		return ErrTimeout
	}

	select {
	case <-buffer.doneCh:
		close(buffer.dataCh)
		close(buffer.flushCh)
		close(buffer.closeCh)
		return nil
	case <-time.After(buffer.options.CloseTimeout):
		return ErrTimeout
	}
}

func (buffer *Buffer) closed() bool {
	select {
	case <-buffer.doneCh:
		return true
	default:
		return false
	}
}

func (buffer *Buffer) consume() {
	count := 0
	mustFlush := false
	ticker, stopTicker := newTicker(buffer.options.FlushInterval)

	isOpen := true
	for isOpen {
		select {
		case item := <-buffer.dataCh:
			buffer.items[count] = item
			count++
			mustFlush = count >= len(buffer.items)
		case <-ticker:
			count = buffer.drain(count)
			mustFlush = count > 0
		case <-buffer.flushCh:
			count = buffer.drain(count)
			mustFlush = count > 0
		case <-buffer.closeCh:
			isOpen = false
			count = buffer.drain(count)
			mustFlush = count > 0
		}

		if mustFlush {
			stopTicker()
			buffer.flush(count)

			count = 0
			mustFlush = false
			ticker, stopTicker = newTicker(buffer.options.FlushInterval)
		}
	}

	stopTicker()
	close(buffer.doneCh)
}

// drain moves every item already accepted into the data channel into the
// batch, writing full batches on the way, and returns the new count. A flush
// or close has to cover everything whose Push has returned, and with a
// buffered channel those items can still be sitting in it.
func (buffer *Buffer) drain(count int) int {
	for {
		select {
		case item := <-buffer.dataCh:
			buffer.items[count] = item
			count++

			if count >= len(buffer.items) {
				buffer.flush(count)
				count = 0
			}
		default:
			return count
		}
	}
}

func (buffer *Buffer) flush(count int) {
	buffer.options.Flusher.Write(buffer.items[:count])
	buffer.items = make([]any, buffer.options.Size)
}

func newTicker(interval time.Duration) (<-chan time.Time, func()) {
	if interval == 0 {
		return nil, func() {}
	}

	ticker := time.NewTicker(interval)
	return ticker.C, ticker.Stop
}

// New creates a new buffer instance with the provided options.
//
// The data channel holds one batch worth of items. A flush runs on the
// consuming goroutine, so with an unbuffered channel every Push made during a
// flush blocks for as long as the flusher takes; under a burst that is a pile
// of blocked callers for every round trip to the destination. The buffer
// absorbs a flush's worth of pushes instead, and a caller only waits once it is
// a full batch ahead of the consumer.
func New(opts ...Option) *Buffer {
	options := resolveOptions(opts...)

	buffer := &Buffer{
		dataCh:  make(chan any, options.Size),
		flushCh: make(chan struct{}),
		closeCh: make(chan struct{}),
		doneCh:  make(chan struct{}),
		options: options,
	}

	buffer.items = make([]any, options.Size)

	go buffer.consume()

	return buffer
}
