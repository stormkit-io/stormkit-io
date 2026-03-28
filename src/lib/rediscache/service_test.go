package rediscache_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stretchr/testify/suite"
)

type ServiceSuite struct {
	suite.Suite
}

func (s *ServiceSuite) BeforeTest(_, _ string) {
	os.Setenv("STORMKIT_SERVICE_NAME", "test-service")
	rediscache.DefaultService = nil
	rediscache.ResetService()
}

func (s *ServiceSuite) AfterTest(_, _ string) {
	os.Unsetenv("STORMKIT_SERVICE_NAME")
}

func (s *ServiceSuite) Test_Service_Singleton() {
	service1 := rediscache.Service()
	service2 := rediscache.Service()

	s.Same(service1, service2, "Service should return the same instance")
	s.NotNil(service1)
}

func (s *ServiceSuite) Test_Service_Key() {
	service := rediscache.Service()
	key := service.Key("test-key")
	s.Contains(key, "service:test-key:test-service:")
}

func (s *ServiceSuite) Test_SubscribeAndBroadcast() {
	service := rediscache.Service()
	channel := "test-event-channel"
	called := ""

	err := service.SubscribeAsync(channel, func(ctx context.Context, s ...string) {
		called = s[0]
		ctx.Done()
	})

	s.NoError(err)

	s.Eventually(func() bool {
		s.NoError(service.Broadcast(channel, "with-payload"))
		return called == "with-payload"
	}, 5*time.Second, 500*time.Millisecond)
}

func (s *ServiceSuite) Test_DelAll() {
	service := rediscache.Service()
	services := []string{"service1", "service2"}

	for _, svc := range services {
		s.NoError(service.SetAll("test_key", "test_value", []string{svc}))
	}

	s.NoError(service.DelAll("test_key", services))

	for _, svc := range services {
		status, err := service.GetAll("test_key", []string{svc})
		s.NoError(err)
		s.Equal("", status[svc], "Expected key to be deleted for service: %s", svc)
	}
}

func (s *ServiceSuite) Test_Service_RegistrationHasTTL() {
	// Use short durations so this test doesn't take long.
	prevTTL, prevInterval := rediscache.ServiceRegistrationTTL, rediscache.HeartbeatInterval
	rediscache.ServiceRegistrationTTL = 2 * time.Second
	rediscache.HeartbeatInterval = 500 * time.Millisecond

	defer func() {
		rediscache.ServiceRegistrationTTL = prevTTL
		rediscache.HeartbeatInterval = prevInterval
	}()

	service := rediscache.Service()
	key := service.Key("sd")

	// Use PTTL for millisecond precision since the TTL may be sub-second.
	pttl, err := rediscache.Client().PTTL(context.Background(), key).Result()
	s.NoError(err)
	s.Greater(pttl.Milliseconds(), int64(0), "service discovery key should have a TTL > 0")
}

func (s *ServiceSuite) Test_Service_HeartbeatKeepsKeyAlive() {
	// Register with a short TTL; the heartbeat must refresh it before it expires.
	// Redis truncates durations below 1s, so use values >= 1s.
	prevTTL, prevInterval := rediscache.ServiceRegistrationTTL, rediscache.HeartbeatInterval
	rediscache.ServiceRegistrationTTL = 1500 * time.Millisecond
	rediscache.HeartbeatInterval = 300 * time.Millisecond

	defer func() {
		rediscache.ServiceRegistrationTTL = prevTTL
		rediscache.HeartbeatInterval = prevInterval
	}()

	service := rediscache.Service()
	key := service.Key("sd")

	// Wait long enough that the TTL would have expired without a heartbeat (2×TTL).
	// The heartbeat fires every 300ms and keeps extending the 1.5s TTL, so the
	// key should still be alive at the 3s mark.
	time.Sleep(3 * time.Second)

	exists, err := rediscache.Client().Exists(context.Background(), key).Result()
	s.NoError(err)
	s.Equal(int64(1), exists, "service discovery key should still exist after heartbeat refreshes")
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, &ServiceSuite{})
}
