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

var Cache *redis.Client

const ListenerChannel = "redis-listener"

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
func Client() *redis.Client {
	mux.Lock()
	defer mux.Unlock()

	if Cache == nil {
		var err error
		Cache, err = newClient()

		if err != nil {
			for attempt := 1; attempt <= 5; attempt++ {
				backoffDuration := time.Duration(attempt*attempt) * time.Second
				slog.Errorf("redis connection attempt %d failed, retrying in %v", attempt, backoffDuration)
				time.Sleep(backoffDuration)

				if Cache, _ = newClient(); Cache != nil {
					break
				}
			}

			if Cache == nil {
				slog.Errorf("failed to establish redis connection after 5 attempts: %v", err)
			}
		}

		slog.Info("created new redis client successfully")
	}

	return Cache
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
