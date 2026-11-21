package integrations

import (
	"time"

	"github.com/aws/smithy-go/middleware"
)

// SetSignedRetryDelay shortens the wait between retries of signed requests and
// returns a function that restores it.
func SetSignedRetryDelay(d time.Duration) func() {
	prev := signedRetryDelay
	signedRetryDelay = d

	return func() { signedRetryDelay = prev }
}

// IdleTimeout returns how long the service may go without a request before it
// is stopped, or 0 when it is kept warm.
func (s *Service) IdleTimeout() time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.idleTimeout()
}

// HasIdleTimer reports whether the idle timer is armed.
func (s *Service) HasIdleTimer() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.timer != nil
}

// SetLambdaMiddlewares adds middlewares to the clients that publish functions
// to Lambda next to HCloud and returns a function that removes them.
func SetLambdaMiddlewares(m ...func(stack *middleware.Stack) error) func() {
	prev := lambdaMiddlewares
	lambdaMiddlewares = m

	return func() { lambdaMiddlewares = prev }
}
