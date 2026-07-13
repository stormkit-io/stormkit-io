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

func newClient() (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: config.Get().RedisAddr,

		// Explicitly disable maintenance notifications
		// This prevents the client from sending CLIENT MAINT_NOTIFICATIONS ON
		// See https://github.com/redis/go-redis/issues/3536#issuecomment-3449792377
		MaintNotificationsConfig: &maintnotifications.Config{
			Mode: maintnotifications.ModeDisabled,
		},
	})

	_, err := client.Ping(context.Background()).Result()

	if err != nil {
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
			for attempt := 1; attempt <= 5; attempt++ {
				backoffDuration := time.Duration(attempt*attempt) * time.Second
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
		strings.Contains(errStr, "i/o timeout")
}
