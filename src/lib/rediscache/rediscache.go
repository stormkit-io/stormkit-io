package rediscache

import (
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/redis/go-redis/v9/maintnotifications"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

var mux sync.Mutex

type RedisCache struct {
	*redis.Client
}

var Cache *RedisCache

// dial builds a client without contacting the server. go-redis connects
// lazily, so this cannot block and cannot fail.
func dial() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: config.Get().RedisAddr,

		// Explicitly disable maintenance notifications
		// This prevents the client from sending CLIENT MAINT_NOTIFICATIONS ON
		// See https://github.com/redis/go-redis/issues/3536#issuecomment-3449792377
		MaintNotificationsConfig: &maintnotifications.Config{
			Mode: maintnotifications.ModeDisabled,
		},
	})
}

// newClient builds a client and verifies it answers. Only startup uses this:
// the ping costs a full dial timeout, and the driver retries it internally, so
// an unreachable server makes a single call here take over a minute.
func newClient() (*redis.Client, error) {
	client := dial()

	if _, err := client.Ping(context.Background()).Result(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

// Client returns a new RedisCache instance. If the connection is not
// closed yet, it returns the shared client.
func Client() *RedisCache {
	mux.Lock()
	defer mux.Unlock()

	if Cache == nil {
		var err error
		var client *redis.Client
		client, err = newClient()

		if err != nil {
			// Backoff stays in the low seconds on purpose: the dial happens
			// under the package mutex, so every other caller is blocked for
			// however long this loop runs.
			for attempt := 1; attempt <= 5; attempt++ {
				backoffDuration := time.Duration(200*(1<<(attempt-1))) * time.Millisecond
				slog.Errorf("redis connection attempt %d failed, retrying in %v", attempt, backoffDuration)
				time.Sleep(backoffDuration)

				if client, _ = newClient(); client != nil {
					break
				}
			}

			if client == nil {
				slog.Errorf("failed to establish redis connection after 5 attempts: %v", err)
				return nil
			}
		}

		Cache = &RedisCache{Client: client}
		slog.Info("created new redis client successfully")
	}

	return Cache
}

// Reset discards the shared client so the next Client call redials.
//
// It is a no-op unless r is still the shared client, so that several
// goroutines observing the same failure do not each tear down a healthy
// replacement.
func (r *RedisCache) Reset() {
	mux.Lock()
	defer mux.Unlock()

	if r == nil || Cache != r {
		return
	}

	// Swap in a replacement rather than clearing the field. Clearing it makes
	// the next caller re-run the verified startup path while holding this
	// mutex, which on an unreachable server parks every other caller for over
	// a minute, and the caller after that pays it again because the failure is
	// not remembered. A lazy client cannot block here, and each command then
	// fails on its own dial timeout instead of behind a process-wide lock.
	old := Cache.Client
	Cache = &RedisCache{Client: dial()}

	if err := old.Close(); err != nil {
		slog.Errorf("error while closing redis client: %v", err)
	}

	slog.Info("discarded redis client, connections will be redialled")
}

// Keys returns all keys matching the given pattern using SCAN.
// This is a cluster-safe alternative to the KEYS command, which may be
// disabled on managed Redis instances.
func (r *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	if r == nil || r.Client == nil {
		return nil, errors.New("redis client is not available")
	}

	var keys []string
	var cursor uint64

	for {
		batch, nextCursor, err := r.Scan(ctx, cursor, pattern, 100).Result()

		if err != nil {
			return nil, err
		}

		keys = append(keys, batch...)
		cursor = nextCursor

		if cursor == 0 {
			break
		}
	}

	return keys, nil
}

func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, redis.ErrClosed) || errors.Is(err, io.EOF) {
		return true
	}

	var netErr net.Error

	if errors.As(err, &netErr) {
		return true
	}

	// Check common network error strings
	errStr := err.Error()
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "no route to host") ||
		strings.Contains(errStr, "i/o timeout") ||
		IsReadOnlyError(err)
}

// IsReadOnlyError reports whether err is a READONLY reply, which a managed
// instance returns while it serves a replica - during and after a
// primary-standby switchover.
//
// The reply arrives as a valid Redis error, so the pool keeps handing out the
// connections pinned to the demoted node and every write path stays dead until
// the client is discarded. Callers should treat it as a connection fault and
// Reset, not as an application error.
//
// Tair's proxy emits "ERR READONLY You can't write against a read only
// instance."; native Redis emits "READONLY You can't write against a read only
// replica.".
func IsReadOnlyError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "READONLY")
}
