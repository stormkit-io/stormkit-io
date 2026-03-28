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
	rediscache.ServiceRegistrationTTL = 500 * time.Millisecond
	rediscache.HeartbeatInterval = 100 * time.Millisecond
	defer func() {
		rediscache.ServiceRegistrationTTL = 30 * time.Second
		rediscache.HeartbeatInterval = 10 * time.Second
	}()

	service := rediscache.Service()
	key := service.Key("sd")

	ttl, err := rediscache.Client().TTL(context.Background(), key).Result()
	s.NoError(err)
	s.Greater(ttl.Milliseconds(), int64(0), "service discovery key should have a TTL > 0")
}

func (s *ServiceSuite) Test_Service_HeartbeatKeepsKeyAlive() {
	// Register with a very short TTL; the heartbeat must refresh it before it expires.
	rediscache.ServiceRegistrationTTL = 300 * time.Millisecond
	rediscache.HeartbeatInterval = 100 * time.Millisecond
	defer func() {
		rediscache.ServiceRegistrationTTL = 30 * time.Second
		rediscache.HeartbeatInterval = 10 * time.Second
	}()

	service := rediscache.Service()
	key := service.Key("sd")

	// Wait long enough that the TTL would have expired without a heartbeat (3×TTL).
	time.Sleep(900 * time.Millisecond)

	exists, err := rediscache.Client().Exists(context.Background(), key).Result()
	s.NoError(err)
	s.Equal(int64(1), exists, "service discovery key should still exist after heartbeat refreshes")
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, &ServiceSuite{})
}
