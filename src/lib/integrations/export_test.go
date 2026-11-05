package integrations

import "time"

// SetSignedRetryDelay shortens the wait between retries of signed requests and
// returns a function that restores it.
func SetSignedRetryDelay(d time.Duration) func() {
	prev := signedRetryDelay
	signedRetryDelay = d

	return func() { signedRetryDelay = prev }
}
