package publicapiv1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type HandlerAnalyticsGetSuite struct {
	suite.Suite
	*factory.Factory

	conn       databasetest.TestDB
	selfHosted bool
	user       *factory.MockUser
	env        *factory.MockEnv
	www        types.ID
	blog       types.ID
	others     types.ID
}

func (s *HandlerAnalyticsGetSuite) SetupSuite() {
	s.selfHosted = config.IsSelfHosted()
	admin.SetMockLicense()
}

// TearDownSuite also restores the edition, which SetMockLicense switches to
// self-hosted, so later suites in the package see the state they expect.
func (s *HandlerAnalyticsGetSuite) TearDownSuite() {
	admin.ResetMockLicense()
	config.SetIsSelfHosted(s.selfHosted)
}

func (s *HandlerAnalyticsGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	s.user = s.MockUser()
	app := s.MockApp(s.user)
	s.env = s.MockEnv(app)
	otherEnv := s.MockEnv(app, map[string]any{"Name": "staging"})

	s.www = s.domain(app.ID, s.env.ID, "www.example.org")
	s.blog = s.domain(app.ID, s.env.ID, "blog.example.org")
	s.others = s.domain(app.ID, otherEnv.ID, "other.example.org")

	today := time.Now().UTC().Format(time.DateOnly)

	for id, views := range map[types.ID]int{s.www: 3, s.blog: 4, s.others: 100} {
		_, err := s.conn.Exec(`WITH path AS (INSERT INTO analytics_paths_agg
			(aggregate_date, domain_id, request_path, request_path_hash, unique_visitors, page_views, server_views, client_views)
			VALUES ($1::date, $2, '/', decode(md5('/'), 'hex'), 1, $3, $3, 0) RETURNING *)
			INSERT INTO analytics_site_agg SELECT domain_id, aggregate_date, page_views, server_views, client_views, engaged_seconds, engaged_views FROM path`, today, id, views)

		s.Require().NoError(err)
	}
}

func (s *HandlerAnalyticsGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAnalyticsGetSuite) domain(appID, envID types.ID, name string) types.ID {
	d := &buildconf.DomainModel{
		AppID:      appID,
		EnvID:      envID,
		Name:       name,
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
		Token:      null.StringFrom(name),
	}

	s.Require().NoError(buildconf.DomainStore().Insert(context.Background(), d))

	return d.ID
}

func (s *HandlerAnalyticsGetSuite) request(query string) shttptest.Response {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/analytics?envId=%s%s", s.env.ID.String(), query),
		nil,
		map[string]string{"Authorization": usertest.Authorization(s.user.ID)},
	)
}

func (s *HandlerAnalyticsGetSuite) pageViews(raw string) float64 {
	var body struct {
		Totals map[string]float64 `json:"totals"`
	}

	s.Require().NoError(json.Unmarshal([]byte(raw), &body))

	return body.Totals["pageViews"]
}

// Test_Success_SumsEnvironmentDomains verifies that a report covers every
// domain of the key's environment and none of another environment's.
func (s *HandlerAnalyticsGetSuite) Test_Success_SumsEnvironmentDomains() {
	res := s.request("&metric=visitors&span=1d")
	body := res.String()

	s.Equal(http.StatusOK, res.Code)
	s.Equal(float64(7), s.pageViews(body))
	s.Contains(body, `"series":[`)
}

func (s *HandlerAnalyticsGetSuite) Test_Success_DomainFilter() {
	res := s.request(fmt.Sprintf("&metric=paths&domainId=%s", s.blog.String()))

	s.Equal(http.StatusOK, res.Code)
	s.Equal(float64(4), s.pageViews(res.String()))
}

// Test_Success_ManyDomains verifies that every domain of the environment is
// found, beyond the size of one domain query page.
func (s *HandlerAnalyticsGetSuite) Test_Success_ManyDomains() {
	var last types.ID

	for i := 0; i < 260; i++ {
		last = s.domain(s.env.AppID, s.env.ID, fmt.Sprintf("d%d.example.org", i))
	}

	_, err := s.conn.Exec(`WITH path AS (INSERT INTO analytics_paths_agg
		(aggregate_date, domain_id, request_path, request_path_hash, unique_visitors, page_views, server_views, client_views)
		VALUES ($1::date, $2, '/', decode(md5('/'), 'hex'), 1, 5, 5, 0) RETURNING *)
		INSERT INTO analytics_site_agg SELECT domain_id, aggregate_date, page_views, server_views, client_views, engaged_seconds, engaged_views FROM path`, time.Now().UTC().Format(time.DateOnly), last)

	s.Require().NoError(err)

	all := s.request("&metric=paths")
	s.Equal(http.StatusOK, all.Code)
	s.Equal(float64(12), s.pageViews(all.String()))

	one := s.request(fmt.Sprintf("&metric=paths&domainId=%s", last.String()))
	s.Equal(http.StatusOK, one.Code)
	s.Equal(float64(5), s.pageViews(one.String()))
}

func (s *HandlerAnalyticsGetSuite) Test_NotFound_DomainOfAnotherEnv() {
	res := s.request(fmt.Sprintf("&domainId=%s", s.others.String()))

	s.Equal(http.StatusNotFound, res.Code)
}

func (s *HandlerAnalyticsGetSuite) Test_BadRequest() {
	res := s.request("&metric=bogus&limit=0")
	body := res.String()

	s.Equal(http.StatusBadRequest, res.Code)
	s.Contains(body, "metric must be one of")
	s.Contains(body, "limit must be between 1 and 50")

	res = s.request("&domainId=abc")

	s.Equal(http.StatusBadRequest, res.Code)
	s.Contains(res.String(), "domainId must be a numeric domain id")
}

func (s *HandlerAnalyticsGetSuite) Test_EnterpriseOnly() {
	admin.ResetMockLicense()
	defer admin.SetMockLicense()

	res := s.request("")

	s.Equal(http.StatusPaymentRequired, res.Code)
	s.Contains(res.String(), "enterprise-only")
}

func TestHandlerAnalyticsGetSuite(t *testing.T) {
	suite.Run(t, new(HandlerAnalyticsGetSuite))
}
