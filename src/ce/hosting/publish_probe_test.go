package hosting

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type WarmupSuite struct {
	suite.Suite

	ctx context.Context
}

func (s *WarmupSuite) SetupTest() {
	s.ctx = context.Background()

	cnf := admin.MustConfig().Clone()
	cnf.DomainConfig = &admin.DomainConfig{Dev: "https://stormkit.dev"}
	admin.SetConfig(&cnf)
}

func (s *WarmupSuite) request(deadline time.Time) deploy.WarmupRequest {
	return deploy.WarmupRequest{
		WarmupID:     "warmup-1",
		AppID:        types.ID(1),
		EnvID:        types.ID(2),
		DeploymentID: types.ID(42),
		DisplayName:  "my-app",
		EnvName:      "production",
		Deadline:     deadline.Unix(),
	}
}

// Test_Probe_WaitsWhileUnavailable verifies the deployment is only reported
// ready once it stops answering "service unavailable" — which is what both of
// the pages served while a process boots return.
func (s *WarmupSuite) Test_Probe_WaitsWhileUnavailable() {
	statuses := []int{
		http.StatusServiceUnavailable,
		http.StatusServiceUnavailable,
		http.StatusOK,
	}

	attempts := 0

	restore := stubWarmupRequest(func(string) int {
		status := statuses[min(attempts, len(statuses)-1)]
		attempts++

		return status
	})

	defer restore()

	result := warmer{}.probe(s.ctx, s.request(time.Now().Add(30*time.Second)))

	s.Equal(rediscache.StatusOK, result.Status)
	s.Equal(http.StatusOK, result.StatusCode)
	s.Equal(3, attempts)
}

// Test_Probe_AnyNonServerErrorIsReady covers the deployments that have nothing
// to boot: a static-only one answers 200, an api-only one 404, one behind an
// auth wall 401, and one pinning a custom port gets Stormkit's own 400.
func (s *WarmupSuite) Test_Probe_AnyNonServerErrorIsReady() {
	for _, status := range []int{
		http.StatusOK,
		http.StatusMovedPermanently,
		http.StatusBadRequest,
		http.StatusUnauthorized,
		http.StatusNotFound,
	} {
		restore := stubWarmupRequest(func(string) int { return status })

		result := warmer{}.probe(s.ctx, s.request(time.Now().Add(5*time.Second)))

		restore()

		s.Equal(rediscache.StatusOK, result.Status, "status %d should count as answering", status)
		s.Equal(status, result.StatusCode)
	}
}

// Test_Probe_ServerErrorIsNotReady covers a server that fails to spawn: the
// edge turns that into a 500, which must not pass the gate.
func (s *WarmupSuite) Test_Probe_ServerErrorIsNotReady() {
	restore := stubWarmupRequest(func(string) int { return http.StatusInternalServerError })
	defer restore()

	result := warmer{}.probe(s.ctx, s.request(time.Now()))

	s.Equal(rediscache.StatusErr, result.Status)
	s.Equal(http.StatusInternalServerError, result.StatusCode)
}

// Test_Probe_EmptyResponseIsNotReady covers a request that produced nothing at
// all, which must not be read as an answer.
func (s *WarmupSuite) Test_Probe_EmptyResponseIsNotReady() {
	restore := stubWarmupRequest(func(string) int { return 0 })
	defer restore()

	result := warmer{}.probe(s.ctx, s.request(time.Now()))

	s.Equal(rediscache.StatusErr, result.Status)
}

func (s *WarmupSuite) Test_Probe_TimesOutWhileUnavailable() {
	restore := stubWarmupRequest(func(string) int { return http.StatusServiceUnavailable })
	defer restore()

	result := warmer{}.probe(s.ctx, s.request(time.Now()))

	s.Equal(rediscache.StatusErr, result.Status)
	s.Equal(http.StatusServiceUnavailable, result.StatusCode)
	s.Contains(result.Reason, "did not serve a request")
}

// Test_Probe_StopsWhenTheNodeShutsDown keeps a draining node from probing on
// for minutes after it has been asked to stop.
func (s *WarmupSuite) Test_Probe_StopsWhenTheNodeShutsDown() {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	attempts := 0

	restore := stubWarmupRequest(func(string) int {
		attempts++
		return http.StatusServiceUnavailable
	})

	defer restore()

	result := warmer{}.probe(ctx, s.request(time.Now().Add(time.Hour)))

	s.Equal(rediscache.StatusErr, result.Status)
	s.Equal(1, attempts)
}

// Test_Host_AddressesTheDeploymentItself verifies the probe asks for the
// deployment's own endpoint: it is the only hostname that resolves to a
// deployment which is not published yet.
func (s *WarmupSuite) Test_Host_AddressesTheDeploymentItself() {
	host := warmupHost(s.request(time.Now()))

	s.Equal("my-app--42.stormkit.dev", host)
}

// Test_Probe_SkipsWithoutAnEndpoint covers an instance with no preview domain:
// nothing addresses an unpublished deployment there, so the publish is let
// through rather than blocked on a warm-up that cannot happen.
func (s *WarmupSuite) Test_Probe_SkipsWithoutAnEndpoint() {
	cnf := admin.MustConfig().Clone()
	cnf.DomainConfig = nil
	admin.SetConfig(&cnf)

	attempts := 0

	restore := stubWarmupRequest(func(string) int {
		attempts++
		return http.StatusOK
	})

	defer restore()

	result := warmer{}.probe(s.ctx, s.request(time.Now().Add(time.Hour)))

	s.Equal(rediscache.StatusOK, result.Status)
	s.Contains(result.Reason, "no endpoint")
	s.Zero(attempts)
}

func (s *WarmupSuite) Test_HandlePublishWarmup_IgnoresMalformedPayload() {
	s.NotPanics(func() {
		HandlePublishWarmup(s.ctx)
		HandlePublishWarmup(s.ctx, "not json")
	})
}

func TestWarmupSuite(t *testing.T) {
	suite.Run(t, new(WarmupSuite))
}

// stubWarmupRequest replaces the request the probe makes, and neutralises the
// process cleanup that follows a failure. It returns a function that puts both
// back.
func stubWarmupRequest(fn func(string) int) func() {
	request, discard, resolve := warmupRequest, warmupDiscard, warmupResolve

	warmupRequest = fn
	warmupDiscard = func(*appconf.Config) {}
	warmupResolve = func(string, types.ID) (*appconf.Config, error) {
		return &appconf.Config{DeploymentID: types.ID(42)}, nil
	}

	return func() {
		warmupRequest = request
		warmupDiscard = discard
		warmupResolve = resolve
	}
}
