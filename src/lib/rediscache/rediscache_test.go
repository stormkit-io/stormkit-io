package rediscache_test

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stretchr/testify/suite"
)

type RedisCacheSuite struct {
	suite.Suite
}

func (s *RedisCacheSuite) Test_IsReadOnlyError() {
	// Tair's proxy and native Redis word the same failover differently.
	s.True(rediscache.IsReadOnlyError(errors.New("ERR READONLY You can't write against a read only instance.")))
	s.True(rediscache.IsReadOnlyError(errors.New("READONLY You can't write against a read only replica.")))

	s.False(rediscache.IsReadOnlyError(nil))
	s.False(rediscache.IsReadOnlyError(redis.Nil))
	s.False(rediscache.IsReadOnlyError(errors.New("NOPERM this user has no permissions")))

	// Discarding the pool is expensive, so an error that merely mentions the
	// word must not be mistaken for a failover.
	s.False(rediscache.IsReadOnlyError(errors.New("ERR unknown command 'READONLY'")))
	s.False(rediscache.IsReadOnlyError(errors.New("ERR Error running script: READONLY")))
	s.False(rediscache.IsReadOnlyError(errors.New("READONLYISH something else")))
}

func (s *RedisCacheSuite) Test_IsConnectionError_Readonly() {
	// The regression the 2026-09-09 failover exposed: a READONLY reply is a
	// topology event, not an application error.
	s.True(rediscache.IsConnectionError(errors.New("ERR READONLY You can't write against a read only instance.")))
	s.True(rediscache.IsConnectionError(errors.New("READONLY You can't write against a read only replica.")))
}

func (s *RedisCacheSuite) Test_IsConnectionError_Network() {
	s.True(rediscache.IsConnectionError(redis.ErrClosed))
	s.True(rediscache.IsConnectionError(io.EOF))
	s.True(rediscache.IsConnectionError(errors.New("dial tcp 10.0.0.1:6379: connect: connection refused")))
	s.True(rediscache.IsConnectionError(errors.New("network is unreachable")))
	s.True(rediscache.IsConnectionError(errors.New("no route to host")))
	s.True(rediscache.IsConnectionError(errors.New("lookup r-x.redis.rds.aliyuncs.com: i/o timeout")))
	s.True(rediscache.IsConnectionError(&net.DNSError{Err: "server misbehaving"}))

	s.False(rediscache.IsConnectionError(nil))
	s.False(rediscache.IsConnectionError(redis.Nil))
	s.False(rediscache.IsConnectionError(errors.New("WRONGTYPE Operation against a key holding the wrong kind of value")))
}

func (s *RedisCacheSuite) Test_Reset_RedialsOnNextClient() {
	client := rediscache.Client()
	s.Require().NotNil(client)

	client.Reset()

	next := rediscache.Client()
	s.Require().NotNil(next)
	s.NotSame(client, next, "Reset should force the next call to redial")
	s.NoError(next.Ping(context.Background()).Err(), "the replacement client should be usable")
}

func (s *RedisCacheSuite) Test_Reset_IgnoresStaleClient() {
	stale := rediscache.Client()
	s.Require().NotNil(stale)

	stale.Reset()

	current := rediscache.Client()
	s.Require().NotNil(current)

	// A second goroutine observing the same failure must not tear down the
	// replacement that already redialled.
	stale.Reset()

	s.Same(current, rediscache.Client())
	s.NoError(current.Ping(context.Background()).Err())
}

func (s *RedisCacheSuite) Test_Reset_NilReceiver() {
	var nilCache *rediscache.RedisCache
	s.NotPanics(func() { nilCache.Reset() })

	s.NotNil(rediscache.Client())
}

func TestRedisCache(t *testing.T) {
	suite.Run(t, &RedisCacheSuite{})
}
