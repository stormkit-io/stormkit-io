package rediscache_test

import (
	"context"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stretchr/testify/suite"
)

// FailoverSuite reproduces the 2026-09-09 incident: a managed primary-standby
// switchover starts refusing writes, and nothing recovers until the client is
// discarded.
//
// The driver already handles the native wording on its own. What it does not
// handle is a proxy that prefixes the reply with "ERR ", which is what the
// managed instance in front of that fleet emits.
type FailoverSuite struct {
	suite.Suite
	stub     *stubRedis
	prevAddr string
	ctx      context.Context
}

func (s *FailoverSuite) start(wording string) *rediscache.RedisCache {
	stub, err := newStubRedis(wording)
	s.Require().NoError(err)
	s.stub = stub

	// Point the package at the stub, then discard the shared client so its
	// replacement is built against the stub rather than the real instance.
	config.SetRedisAddr(stub.Addr())

	if current := rediscache.Client(); current != nil {
		current.Reset()
	}

	client := rediscache.Client()
	s.Require().NotNil(client)
	s.Require().NoError(client.Set(s.ctx, "k", "v", 0).Err(), "writable before the switchover")

	return client
}

func (s *FailoverSuite) SetupTest() {
	s.ctx = context.Background()
	s.prevAddr = config.Get().RedisAddr
}

func (s *FailoverSuite) TearDownTest() {
	// Restore the address before discarding the client: Reset builds the
	// replacement against whatever the config says at that moment.
	config.SetRedisAddr(s.prevAddr)

	if current := rediscache.Client(); current != nil {
		current.Reset()
	}

	if s.stub != nil {
		s.stub.Close()
		s.stub = nil
	}
}

// Test_NativeWording_DriverRecoversOnItsOwn documents why this was never seen
// on a plain Redis. The driver recognises the reply, closes the pinned
// connection and retries on a fresh one, so the caller sees nothing at all.
func (s *FailoverSuite) Test_NativeWording_DriverRecoversOnItsOwn() {
	client := s.start(nativeReadOnly)

	connsBefore := s.stub.Conns()
	s.stub.Failover()

	s.NoError(client.Set(s.ctx, "k", "v", 0).Err(), "the driver retries this away")
	s.Greater(s.stub.Conns(), connsBefore, "the driver redialled by itself")
}

// Test_ProxyWording_SurfacesAndStaysPinned is the incident. The "ERR " prefix
// defeats the driver's prefix match, so the connection is never marked bad,
// the command is never retried, and the pool keeps handing out connections
// pinned to the demoted node for as long as the process lives.
func (s *FailoverSuite) Test_ProxyWording_SurfacesAndStaysPinned() {
	client := s.start(proxyReadOnly)

	connsBefore := s.stub.Conns()
	s.stub.Failover()

	for range 20 {
		err := client.Set(s.ctx, "k", "v", 0).Err()

		s.Require().Error(err)
		s.Require().Contains(err.Error(), "READONLY")
	}

	s.Equal(connsBefore, s.stub.Conns(), "the pool never redials on its own")
}

// Test_ProxyWording_IsClassifiedAsConnectionError is the fix. Recognising the
// reply is what gives any caller a reason to discard the client.
func (s *FailoverSuite) Test_ProxyWording_IsClassifiedAsConnectionError() {
	client := s.start(proxyReadOnly)

	s.stub.Failover()

	err := client.Set(s.ctx, "k", "v", 0).Err()

	s.Require().Error(err)
	s.True(rediscache.IsReadOnlyError(err))
	s.True(rediscache.IsConnectionError(err), "a failover must be treated as a connection fault")
}

// Test_ProxyWording_ResetRecovers closes the loop: discarding the client drops
// the pinned connections and the redial reaches the promoted primary.
func (s *FailoverSuite) Test_ProxyWording_ResetRecovers() {
	client := s.start(proxyReadOnly)

	connsBefore := s.stub.Conns()
	s.stub.Failover()

	err := client.Set(s.ctx, "k", "v", 0).Err()
	s.Require().Error(err)
	s.Require().True(rediscache.IsConnectionError(err))

	client.Reset()

	recovered := rediscache.Client()
	s.Require().NotNil(recovered)
	s.NotSame(client, recovered)

	s.NoError(recovered.Set(s.ctx, "k", "v", 0).Err(), "writes work again after the redial")
	s.Greater(s.stub.Conns(), connsBefore, "the pool dialled the promoted primary")
}

func TestFailover(t *testing.T) {
	suite.Run(t, &FailoverSuite{})
}
