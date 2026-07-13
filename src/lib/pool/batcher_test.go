package pool_test

import (
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
	recorded := 0
	num := 100
	buf := pool.New(
		pool.WithSize(num),
		pool.WithFlusher(pool.FlusherFunc(func(items []any) {
			recorded = len(items)
		})),
	)
	// ensure the buffer
	defer buf.Close()

	for i := 0; i < num; i++ {
		buf.Push(i)
	}

	// We need to wait a bit for the routine to be launched and processed
	time.Sleep(50 * time.Microsecond)

	s.Equal(recorded, num)
}

func (s *BatcherSuite) Test_BatchByCount_Notfulfilled() {
	recorded := 0
	num := 100
	buf := pool.New(
		pool.WithSize(num),
		pool.WithFlusher(pool.FlusherFunc(func(items []any) {
			recorded = len(items)
		})),
	)
	// ensure the buffer
	defer buf.Close()

	for i := 0; i < num-1; i++ {
		buf.Push(i)
	}

	// We need to wait a bit for the routine to be launched and processed
	time.Sleep(50 * time.Microsecond)

	s.Equal(recorded, 0)
}

func (s *BatcherSuite) Test_BatchByTime_Fulfill() {
	recorded := 0
	num := 100
	buf := pool.New(
		pool.WithSize(num),
		pool.WithFlushInterval(time.Millisecond*500),
		pool.WithFlusher(pool.FlusherFunc(func(items []any) {
			recorded = len(items)
		})),
	)
	// ensure the buffer
	defer buf.Close()

	for i := 0; i < 20; i++ {
		buf.Push(i)
	}

	// We need to wait a bit for the routine to be launched and processed
	time.Sleep(time.Second)

	s.Equal(recorded, 20)
}

func (s *BatcherSuite) Test_BatchByTime_NotFulfill() {
	recorded := 0
	num := 100
	buf := pool.New(
		pool.WithSize(num),
		pool.WithFlushInterval(time.Millisecond*500),
		pool.WithFlusher(pool.FlusherFunc(func(items []any) {
			recorded = len(items)
		})),
	)
	// ensure the buffer
	defer buf.Close()

	for i := 0; i < 20; i++ {
		buf.Push(i)
	}

	s.Equal(recorded, 0)
}

func TestBatcher(t *testing.T) {
	suite.Run(t, &BatcherSuite{})
}
