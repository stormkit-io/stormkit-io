package rediscache_test

import (
	"sync"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
)

// blackhole is a routable-looking address that never answers, so dials hang
// until the driver's dial timeout rather than failing fast.
const blackhole = "10.255.255.1:6379"

// Guards against the lockup that discarding the client used to cause. Clearing
// the shared field made the next caller re-run the verified startup path under
// the package mutex; against an unreachable server that took over a minute,
// and every caller behind it paid the same again. Since every request resolves
// the client, that cost lands on the whole edge at once.
func TestResetDoesNotLockOutCallers(t *testing.T) {
	stub, err := newStubRedis(proxyReadOnly)

	if err != nil {
		t.Fatal(err)
	}

	defer stub.Close()

	prev := config.Get().RedisAddr

	// Address first: Reset builds its replacement from the config as it runs,
	// so discarding before pointing at the stub would connect to neither.
	config.SetRedisAddr(stub.Addr())

	if current := rediscache.Client(); current != nil {
		current.Reset()
	}

	client := rediscache.Client()

	if client == nil {
		t.Fatal("expected a client against the stub")
	}

	// The server becomes unreachable, then a connection fault discards the
	// client - the exact sequence the failover fix introduces.
	config.SetRedisAddr(blackhole)

	defer func() {
		// Restore the address before discarding the client: Reset builds the
		// replacement against whatever the config says at that moment.
		config.SetRedisAddr(prev)

		if current := rediscache.Client(); current != nil {
			current.Reset()
		}
	}()

	resetStart := time.Now()
	client.Reset()
	resetCost := time.Since(resetStart)

	const waiters = 50

	var wg sync.WaitGroup
	durations := make([]time.Duration, waiters)

	start := time.Now()

	for i := range waiters {
		wg.Add(1)

		go func() {
			defer wg.Done()

			began := time.Now()
			rediscache.Client()
			durations[i] = time.Since(began)
		}()
	}

	wg.Wait()

	total := time.Since(start)

	var worst time.Duration

	for _, d := range durations {
		if d > worst {
			worst = d
		}
	}

	t.Logf("reset itself took %v", resetCost.Round(time.Microsecond))
	t.Logf("worst of %d concurrent resolutions: %v", waiters, worst.Round(time.Microsecond))
	t.Logf("all %d resolutions drained in %v", waiters, total.Round(time.Microsecond))

	// Generous bounds: the point is that nothing dials while holding the
	// mutex, so these are microseconds rather than minutes.
	if resetCost > time.Second {
		t.Fatalf("reset blocked for %v, it must not dial under the mutex", resetCost)
	}

	if worst > time.Second {
		t.Fatalf("a caller blocked for %v behind a discarded client", worst)
	}
}
