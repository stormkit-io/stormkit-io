package pool_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/pool"
	"github.com/stretchr/testify/suite"
)

type BatcherSuite struct {
	suite.Suite
}

type TestItem struct {
	Name  string
	Value string
}

func (s *BatcherSuite) Test_BatchByCount_Fulfill() {
	var recorded atomic.Int32
	num := 100
	buf := pool.New(
		pool.WithSize(num),
		pool.WithFlusher(pool.FlusherFunc(func(items []any) {
			recorded.Store(int32(len(items)))
		})),
	)
	// ensure the buffer
	defer buf.Close()

	for i := 0; i < num; i++ {
		buf.Push(i)
	}

	s.Eventually(func() bool { return recorded.Load() == int32(num) }, time.Second, time.Millisecond)
}

// Test_PushDoesNotWaitForFlush verifies that a batch's worth of pushes goes
// through while the flusher is busy, instead of each push blocking until the
// flush returns.
func (s *BatcherSuite) Test_PushDoesNotWaitForFlush() {
	const size = 50
	const flushTakes = 300 * time.Millisecond

	var flushes atomic.Int32

	buf := pool.New(
		pool.WithSize(size),
		pool.WithFlusher(pool.FlusherFunc(func(items []any) {
			flushes.Add(1)
			time.Sleep(flushTakes)
		})),
	)
	defer buf.Close()

	// Fill one batch so the flusher starts and goes to sleep.
	for i := 0; i < size; i++ {
		s.NoError(buf.Push(i))
	}

	s.Eventually(func() bool { return flushes.Load() == 1 }, time.Second, time.Millisecond)

	// The next batch must be accepted while that flush is still running.
	start := time.Now()

	for i := 0; i < size; i++ {
		s.NoError(buf.Push(i))
	}

	s.Less(time.Since(start), flushTakes/2, "pushes waited for the flush to finish")
}

func (s *BatcherSuite) Test_BatchByCount_Notfulfilled() {
	var recorded atomic.Int32
	num := 100
	buf := pool.New(
		pool.WithSize(num),
		pool.WithFlusher(pool.FlusherFunc(func(items []any) {
			recorded.Store(int32(len(items)))
		})),
	)
	// ensure the buffer
	defer buf.Close()

	for i := 0; i < num-1; i++ {
		buf.Push(i)
	}

	// We need to wait a bit for the routine to be launched and processed
	time.Sleep(50 * time.Microsecond)

	s.Equal(int32(0), recorded.Load())
}

func (s *BatcherSuite) Test_BatchByTime_Fulfill() {
	var recorded atomic.Int32
	num := 100
	buf := pool.New(
		pool.WithSize(num),
		pool.WithFlushInterval(time.Millisecond*500),
		pool.WithFlusher(pool.FlusherFunc(func(items []any) {
			recorded.Store(int32(len(items)))
		})),
	)
	// ensure the buffer
	defer buf.Close()

	for i := 0; i < 20; i++ {
		buf.Push(i)
	}

	// We need to wait a bit for the routine to be launched and processed
	time.Sleep(time.Second)

	s.Equal(int32(20), recorded.Load())
}

func (s *BatcherSuite) Test_BatchByTime_NotFulfill() {
	var recorded atomic.Int32
	num := 100
	buf := pool.New(
		pool.WithSize(num),
		pool.WithFlushInterval(time.Millisecond*500),
		pool.WithFlusher(pool.FlusherFunc(func(items []any) {
			recorded.Store(int32(len(items)))
		})),
	)
	// ensure the buffer
	defer buf.Close()

	for i := 0; i < 20; i++ {
		buf.Push(i)
	}

	s.Equal(int32(0), recorded.Load())
}

// Test_CloseFlushesQueuedItems verifies that items accepted while a flush was
// running are written on Close, not discarded with the channel.
func (s *BatcherSuite) Test_CloseFlushesQueuedItems() {
	const size = 50
	const queued = 30
	const flushTakes = 200 * time.Millisecond

	var written atomic.Int32

	buf := pool.New(
		pool.WithSize(size),
		pool.WithFlusher(pool.FlusherFunc(func(items []any) {
			written.Add(int32(len(items)))
			time.Sleep(flushTakes)
		})),
	)

	for i := 0; i < size; i++ {
		s.NoError(buf.Push(i))
	}

	// These land in the channel while the first flush is still running.
	for i := 0; i < queued; i++ {
		s.NoError(buf.Push(i))
	}

	s.NoError(buf.Close())
	s.Equal(int32(size+queued), written.Load())
}

func TestBatcher(t *testing.T) {
	suite.Run(t, &BatcherSuite{})
}
