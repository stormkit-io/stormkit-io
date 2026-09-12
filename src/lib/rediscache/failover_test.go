package rediscache_test

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
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
	stub         *stubRedis
	prevAddr     string
	prevCooldown time.Duration
	ctx          context.Context
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
	s.prevCooldown = rediscache.AutoResetCooldown

	// Each test drives its own switchover, so they must not throttle each
	// other. Test_ProxyWording_DiscardsAtMostOncePerWindow sets its own.
	rediscache.AutoResetCooldown = 0
}

func (s *FailoverSuite) TearDownTest() {
	rediscache.AutoResetCooldown = s.prevCooldown

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

// Test_DriverAloneStaysPinnedOnProxyWording is the incident, isolated to the
// driver with none of our handling attached.
//
// The "ERR " prefix defeats the driver's prefix match, so the connection is
// never marked bad, the command is never retried, and the pool keeps handing
// out connections pinned to the demoted node for as long as the process lives.
// The same test against the native wording recovers by itself, which is why
// this was never seen on a plain Redis.
func (s *FailoverSuite) Test_DriverAloneStaysPinnedOnProxyWording() {
	for _, tc := range []struct {
		name          string
		wording       string
		wantRecovered bool
	}{
		{"proxy wording", proxyReadOnly, false},
		{"native wording", nativeReadOnly, true},
	} {
		s.Run(tc.name, func() {
			stub, err := newStubRedis(tc.wording)
			s.Require().NoError(err)

			defer stub.Close()

			bare := redis.NewClient(&redis.Options{Addr: stub.Addr()})
			defer bare.Close()

			s.Require().NoError(bare.Set(s.ctx, "k", "v", 0).Err())

			connsBefore := stub.Conns()
			stub.Failover()

			var lastErr error

			for range 20 {
				lastErr = bare.Set(s.ctx, "k", "v", 0).Err()
			}

			if tc.wantRecovered {
				s.NoError(lastErr, "the driver recognises this wording and redials")
				s.Greater(stub.Conns(), connsBefore)

				return
			}

			s.Require().Error(lastErr)
			s.Contains(lastErr.Error(), "READONLY")
			s.Equal(connsBefore, stub.Conns(), "the pool never redials on its own")
		})
	}
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

// Test_ProxyWording_RecoversWithoutCallerHelp is the behaviour the edge
// depends on. Certificates, one-time codes, token storage and the analytics
// queue all write from paths that know nothing about failovers, so recovery
// cannot require each caller to check for it.
func (s *FailoverSuite) Test_ProxyWording_RecoversWithoutCallerHelp() {
	client := s.start(proxyReadOnly)

	connsBefore := s.stub.Conns()
	s.stub.Failover()

	s.Require().Error(client.Set(s.ctx, "k", "v", 0).Err())

	// The discard is detached, since it closes the pool from inside the
	// command path that is still using it.
	s.Eventually(func() bool {
		return rediscache.Client() != client
	}, 2*time.Second, 10*time.Millisecond, "the client should be discarded without any caller asking")

	recovered := rediscache.Client()
	s.Require().NotNil(recovered)
	s.NoError(recovered.Set(s.ctx, "k", "v", 0).Err())
	s.Greater(s.stub.Conns(), connsBefore)
}

// Test_ProxyWording_RecoversFromPipeline covers the analytics queue, which
// writes through a pipeline. A pipeline reports failures per command rather
// than for the batch, so the refusal is only visible on the commands.
func (s *FailoverSuite) Test_ProxyWording_RecoversFromPipeline() {
	client := s.start(proxyReadOnly)

	s.stub.Failover()

	pipe := client.Pipeline()
	pipe.LPush(s.ctx, "queue", "record")
	_, err := pipe.Exec(s.ctx)

	s.Require().Error(err)

	s.Eventually(func() bool {
		return rediscache.Client() != client
	}, 2*time.Second, 10*time.Millisecond, "a refused pipeline should discard the client too")
}

// Test_NativeWording_DoesNotDiscard guards against over-reacting. The driver
// already recycles the connection for this wording, so discarding the whole
// client on top of that would turn a handled event into a visible one.
func (s *FailoverSuite) Test_NativeWording_DoesNotDiscard() {
	client := s.start(nativeReadOnly)

	s.stub.Failover()

	s.Require().NoError(client.Set(s.ctx, "k", "v", 0).Err())

	s.Never(func() bool {
		return rediscache.Client() != client
	}, 500*time.Millisecond, 50*time.Millisecond, "the driver handled it, nothing should be discarded")
}

// Test_ProxyWording_DiscardsAtMostOncePerWindow is the throttle. A switchover
// refuses every write for as long as it lasts, so without a window each
// refusal would tear down the pool and build another, hundreds of times a
// second, aborting in-flight reads each time.
func (s *FailoverSuite) Test_ProxyWording_DiscardsAtMostOncePerWindow() {
	rediscache.AutoResetCooldown = time.Minute

	client := s.start(proxyReadOnly)

	// Still in progress, so even a fresh connection is refused.
	s.stub.HoldReadOnly()

	// The switchover keeps refusing, so every one of these would discard the
	// client if nothing bounded it.
	for range 50 {
		s.Require().Error(client.Set(s.ctx, "k", "v", 0).Err())
	}

	s.Eventually(func() bool {
		return rediscache.Client() != client
	}, 2*time.Second, 10*time.Millisecond)

	replacement := rediscache.Client()

	for range 50 {
		s.Require().Error(replacement.Set(s.ctx, "k", "v", 0).Err())
	}

	s.Never(func() bool {
		return rediscache.Client() != replacement
	}, 500*time.Millisecond, 50*time.Millisecond, "a second discard inside the window would thrash the pool")
}

func TestFailover(t *testing.T) {
	suite.Run(t, &FailoverSuite{})
}
