package shttp_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type IdleTimeoutReaderSuite struct {
	suite.Suite
}

func (s *IdleTimeoutReaderSuite) Test_Read_ForwardsData() {
	payload := []byte("hello")
	cancelled := make(chan struct{})
	cancel := func() { close(cancelled) }

	r := shttp.NewIdleTimeoutReader(io.NopCloser(bytes.NewReader(payload)), cancel, 5*time.Second)
	defer r.Close()

	buf := make([]byte, len(payload))
	n, err := r.Read(buf)

	s.NoError(err)
	s.Equal(len(payload), n)
	s.Equal(payload, buf[:n])
}

func (s *IdleTimeoutReaderSuite) Test_Read_CallsCancelOnIdleTimeout() {
	cancelled := make(chan struct{}, 1)
	cancel := func() {
		select {
		case cancelled <- struct{}{}:
		default:
		}
	}

	pr, _ := io.Pipe()
	defer pr.Close()

	r := shttp.NewIdleTimeoutReader(pr, cancel, 50*time.Millisecond)
	defer r.Close()

	go r.Read(make([]byte, 4))

	select {
	case <-cancelled:
	case <-time.After(500 * time.Millisecond):
		s.Fail("cancel was not called within the expected timeout window")
	}
}

func (s *IdleTimeoutReaderSuite) Test_Close_StopsTimer() {
	cancelled := make(chan struct{}, 1)
	cancel := func() {
		select {
		case cancelled <- struct{}{}:
		default:
		}
	}

	r := shttp.NewIdleTimeoutReader(io.NopCloser(bytes.NewReader([]byte("x"))), cancel, 100*time.Millisecond)
	r.Close()

	time.Sleep(200 * time.Millisecond)

	select {
	case <-cancelled:
		s.Fail("cancel should not have been called after Close stopped the timer")
	default:
	}
}

func TestIdleTimeoutReaderSuite(t *testing.T) {
	suite.Run(t, new(IdleTimeoutReaderSuite))
}
