package publicapiv1_test

import (
	"context"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

// withMockLicense enables an enterprise license for one test and returns a
// function that restores the previous license and edition.
func withMockLicense() func() {
	selfHosted := config.IsSelfHosted()
	admin.SetMockLicense()

	return func() {
		admin.ResetMockLicense()
		config.SetIsSelfHosted(selfHosted)
	}
}

// mockAnalytics gives the environment a domain with five page views on /blog/a
// today.
func (s *HandlerMCPSuite) mockAnalytics(env *factory.MockEnv) types.ID {
	d := &buildconf.DomainModel{
		AppID:      env.AppID,
		EnvID:      env.ID,
		Name:       "analytics.example.org",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
		Token:      null.StringFrom("analytics-token"),
	}

	s.Require().NoError(buildconf.DomainStore().Insert(context.Background(), d))

	_, err := s.conn.Exec(`WITH path AS (INSERT INTO analytics_paths_agg
		(aggregate_date, domain_id, request_path, request_path_hash, unique_visitors, page_views, server_views, client_views)
		VALUES ($1::date, $2, '/blog/a', decode(md5('/blog/a'), 'hex'), 2, 5, 4, 1) RETURNING *)
		INSERT INTO analytics_site_agg SELECT domain_id, aggregate_date, page_views, server_views, client_views, engaged_seconds, engaged_views FROM path`,
		time.Now().UTC().Format(time.DateOnly), d.ID)

	s.Require().NoError(err)

	return d.ID
}

func (s *HandlerMCPSuite) Test_GetAnalytics_Success() {
	defer withMockLicense()()

	usr := s.MockUser()
	mockEnv := s.MockEnv(s.MockApp(usr))
	key := s.userKey(usr)

	s.mockAnalytics(mockEnv)

	resp := s.post(key.Value, mcpToolCall(1, "get_analytics", map[string]any{
		"envId":  mockEnv.ID.String(),
		"metric": "paths",
		"span":   "7d",
		"limit":  5,
	}))

	data := s.toolContent(s.rpcOK(resp))
	rows := data["rows"].([]any)

	s.Require().Len(rows, 1)
	s.Equal("/blog/a", rows[0].(map[string]any)["path"])
	s.Equal(float64(5), data["totals"].(map[string]any)["pageViews"])
	s.NotEmpty(data["notes"])
}

// Test_GetAnalytics_NumericDomainID verifies that a domain id sent as a JSON
// number filters the report instead of being dropped.
func (s *HandlerMCPSuite) Test_GetAnalytics_NumericDomainID() {
	defer withMockLicense()()

	usr := s.MockUser()
	mockEnv := s.MockEnv(s.MockApp(usr))
	key := s.userKey(usr)

	s.mockAnalytics(mockEnv)

	resp := s.post(key.Value, mcpToolCall(1, "get_analytics", map[string]any{
		"envId":    mockEnv.ID.String(),
		"domainId": 999999,
	}))

	env := s.rpcOK(resp)
	s.True(env["result"].(map[string]any)["isError"].(bool), "an unknown numeric domain is reported, not ignored")
}

func (s *HandlerMCPSuite) Test_GetAnalytics_InvalidArguments() {
	defer withMockLicense()()

	usr := s.MockUser()
	mockEnv := s.MockEnv(s.MockApp(usr))
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "get_analytics", map[string]any{
		"envId":  mockEnv.ID.String(),
		"metric": "bogus",
	}))

	env := s.rpcOK(resp)
	s.True(env["result"].(map[string]any)["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_GetAnalytics_Forbidden_NotMember() {
	defer withMockLicense()()

	mockEnv := s.MockEnv(s.MockApp(s.MockUser()))
	key := s.userKey(s.MockUser())

	s.mockAnalytics(mockEnv)

	resp := s.post(key.Value, mcpToolCall(1, "get_analytics", map[string]any{
		"envId": mockEnv.ID.String(),
	}))

	env := s.rpcOK(resp)
	s.True(env["result"].(map[string]any)["isError"].(bool))
}
