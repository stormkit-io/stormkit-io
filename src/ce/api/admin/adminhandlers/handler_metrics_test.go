package adminhandlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/sysstats"
)

type HandlerMetricsSuite struct {
	suite.Suite
	*factory.Factory
	conn   databasetest.TestDB
	store  *sysstats.Store
	target string
	ctx    context.Context
}

func (s *HandlerMetricsSuite) BeforeTest(suiteName, _ string) {
	s.ctx = context.Background()
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.store = sysstats.NewStore(sysstats.NewStoreParams{})
	s.target = fmt.Sprintf("metrics-test-%d:9100", time.Now().UnixNano())

	// The instance config is cached in a process-global pointer, so without
	// this a config written by one test is still visible to the next.
	admin.ResetCache(s.ctx)
}

func (s *HandlerMetricsSuite) AfterTest(_, _ string) {
	_ = s.store.Drop(s.ctx, s.target)
	admin.ResetCache(s.ctx)
	s.conn.CloseTx()
}

func (s *HandlerMetricsSuite) request(method, target string, body any, admin bool) shttptest.Response {
	usr := s.MockUser(map[string]any{"IsAdmin": admin})

	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		method,
		target,
		body,
		map[string]string{"Authorization": usertest.Authorization(usr.ID)},
	)
}

func (s *HandlerMetricsSuite) configureTargets(targets ...string) {
	cfg := admin.MustConfig()
	cfg.MonitoringConfig = &admin.MonitoringConfig{Targets: targets}

	s.Require().NoError(admin.Store().UpsertConfig(s.ctx, cfg))
}

func (s *HandlerMetricsSuite) appendSample(cpu float64) {
	s.Require().NoError(s.store.Append(s.ctx, &sysstats.Sample{
		Target:     s.target,
		Timestamp:  time.Now().Unix(),
		Reachable:  true,
		CPUPercent: cpu,
		CPUValid:   true,
		Filesystems: []sysstats.Filesystem{
			{Mountpoint: "/", SizeBytes: 100, AvailBytes: 40},
		},
	}))
}

func (s *HandlerMetricsSuite) Test_Get_ReturnsMachinesAndDependencies() {
	s.configureTargets(s.target)
	s.appendSample(42)

	resp := s.request(shttp.MethodGet, "/admin/metrics", nil, true)
	s.Require().Equal(http.StatusOK, resp.Code)

	data := map[string]any{}
	s.Require().NoError(json.Unmarshal(resp.Byte(), &data))

	machines, ok := data["machines"].([]any)
	s.Require().True(ok)
	s.Require().NotEmpty(machines)

	machine := s.findMachine(machines, s.target)
	s.Require().NotNil(machine)
	s.Equal(true, machine["manual"])

	sample, ok := machine["sample"].(map[string]any)
	s.Require().True(ok)
	s.Equal(float64(42), sample["cpuPercent"])

	s.Contains(data, "dependencies")
	s.Contains(data, "pool")
	s.Equal(float64(24), data["retentionHours"])
}

// A machine that has not been scraped yet must still be listed, with a null
// sample rather than being hidden.
func (s *HandlerMetricsSuite) Test_Get_MachineWithNoSamplesYet() {
	s.configureTargets(s.target)

	resp := s.request(shttp.MethodGet, "/admin/metrics", nil, true)
	s.Require().Equal(http.StatusOK, resp.Code)

	data := map[string]any{}
	s.Require().NoError(json.Unmarshal(resp.Byte(), &data))

	machine := s.findMachine(data["machines"].([]any), s.target)
	s.Require().NotNil(machine)
	s.Nil(machine["sample"])
}

func (s *HandlerMetricsSuite) Test_Get_Unauthorized_NonAdmin() {
	resp := s.request(shttp.MethodGet, "/admin/metrics", nil, false)
	s.Equal(http.StatusUnauthorized, resp.Code)
}

func (s *HandlerMetricsSuite) Test_History() {
	s.appendSample(10)

	resp := s.request(shttp.MethodGet, "/admin/metrics/history?target="+s.target, nil, true)
	s.Require().Equal(http.StatusOK, resp.Code)

	data := map[string]any{}
	s.Require().NoError(json.Unmarshal(resp.Byte(), &data))

	samples, ok := data["samples"].([]any)
	s.Require().True(ok)
	s.Require().Len(samples, 1)
	s.Equal(s.target, data["target"])
}

func (s *HandlerMetricsSuite) Test_History_RequiresTarget() {
	resp := s.request(shttp.MethodGet, "/admin/metrics/history", nil, true)
	s.Equal(http.StatusBadRequest, resp.Code)
}

func (s *HandlerMetricsSuite) Test_History_RejectsInvalidSince() {
	resp := s.request(shttp.MethodGet, "/admin/metrics/history?target=x&since=yesterday", nil, true)
	s.Equal(http.StatusBadRequest, resp.Code)
}

func (s *HandlerMetricsSuite) Test_History_Unauthorized_NonAdmin() {
	resp := s.request(shttp.MethodGet, "/admin/metrics/history?target=x", nil, false)
	s.Equal(http.StatusUnauthorized, resp.Code)
}

func (s *HandlerMetricsSuite) Test_Targets_GetAndUpdate() {
	resp := s.request(shttp.MethodGet, "/admin/metrics/targets", nil, true)
	s.Require().Equal(http.StatusOK, resp.Code)

	data := map[string]any{}
	s.Require().NoError(json.Unmarshal(resp.Byte(), &data))
	s.Empty(data["targets"], "no manual targets on a fresh instance")

	resp = s.request(shttp.MethodPut, "/admin/metrics/targets", map[string]any{
		"targets": []string{" db-host:9100 ", "db-host:9100", ""},
	}, true)

	s.Require().Equal(http.StatusOK, resp.Code)
	s.Require().NoError(json.Unmarshal(resp.Byte(), &data))
	s.Equal([]any{"db-host:9100"}, data["targets"], "trimmed, de-duplicated, blanks dropped")
}

func (s *HandlerMetricsSuite) Test_Targets_RejectsUnsafeValues() {
	for _, target := range []string{"file:///etc/passwd", "gopher://evil", "has space", "http://"} {
		resp := s.request(shttp.MethodPut, "/admin/metrics/targets", map[string]any{
			"targets": []string{target},
		}, true)

		s.Equal(http.StatusBadRequest, resp.Code, "%q is fetched by the server and must be rejected", target)
	}
}

// Removing a target drops its history, so a machine no longer monitored stops
// showing up straight away.
func (s *HandlerMetricsSuite) Test_Targets_RemovalDropsHistory() {
	s.configureTargets(s.target)
	s.appendSample(10)

	resp := s.request(shttp.MethodPut, "/admin/metrics/targets", map[string]any{
		"targets": []string{},
	}, true)

	s.Require().Equal(http.StatusOK, resp.Code)

	latest, err := s.store.Latest(s.ctx, s.target)
	s.Require().NoError(err)
	s.Nil(latest)
}

func (s *HandlerMetricsSuite) Test_Targets_Unauthorized_NonAdmin() {
	s.Equal(http.StatusUnauthorized, s.request(shttp.MethodGet, "/admin/metrics/targets", nil, false).Code)
	s.Equal(http.StatusUnauthorized, s.request(shttp.MethodPut, "/admin/metrics/targets", map[string]any{}, false).Code)
}

func (s *HandlerMetricsSuite) findMachine(machines []any, host string) map[string]any {
	for _, item := range machines {
		machine, ok := item.(map[string]any)

		if ok && machine["host"] == host {
			return machine
		}
	}

	return nil
}

func TestHandlerMetricsSuite(t *testing.T) {
	suite.Run(t, new(HandlerMetricsSuite))
}
