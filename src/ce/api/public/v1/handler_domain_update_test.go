package publicapiv1_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type HandlerUpdateDomainSuite struct {
	suite.Suite
	*factory.Factory

	conn      databasetest.TestDB
	mockCache *mocks.CacheInterface
}

func (s *HandlerUpdateDomainSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.mockCache = &mocks.CacheInterface{}
	appcache.DefaultCacheService = s.mockCache
	s.Factory = factory.New(s.conn)
}

func (s *HandlerUpdateDomainSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	appcache.DefaultCacheService = nil
}

func (s *HandlerUpdateDomainSuite) mockDomain(appID, envID types.ID) *buildconf.DomainModel {
	domain := &buildconf.DomainModel{
		AppID:      appID,
		EnvID:      envID,
		Name:       "example.org",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domain))

	return domain
}

func (s *HandlerUpdateDomainSuite) request(usrID types.ID, body map[string]any) int {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPut,
		"/v1/domains",
		body,
		map[string]string{
			"Authorization": usertest.Authorization(usrID),
		},
	).Code
}

func (s *HandlerUpdateDomainSuite) Test_ExcludeFromAnalytics() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	domain := s.mockDomain(app.ID, env.ID)

	s.False(domain.AnalyticsExcluded)
	s.mockCache.On("Reset", types.ID(0), "example.org").Return(nil)

	code := s.request(usr.ID, map[string]any{
		"appId":             app.ID.String(),
		"envId":             env.ID.String(),
		"domainId":          domain.ID.String(),
		"analyticsExcluded": true,
	})

	s.Equal(http.StatusOK, code)

	updated, err := buildconf.DomainStore().DomainByID(context.Background(), domain.ID)

	s.NoError(err)
	s.True(updated.AnalyticsExcluded)
	s.mockCache.AssertCalled(s.T(), "Reset", types.ID(0), "example.org")
}

func (s *HandlerUpdateDomainSuite) Test_IncludeInAnalytics() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	domain := s.mockDomain(app.ID, env.ID)

	domain.AnalyticsExcluded = true
	s.NoError(buildconf.DomainStore().UpdateDomainFlags(context.Background(), domain))

	s.mockCache.On("Reset", types.ID(0), "example.org").Return(nil)

	code := s.request(usr.ID, map[string]any{
		"appId":             app.ID.String(),
		"envId":             env.ID.String(),
		"domainId":          domain.ID.String(),
		"analyticsExcluded": false,
	})

	s.Equal(http.StatusOK, code)

	updated, err := buildconf.DomainStore().DomainByID(context.Background(), domain.ID)

	s.NoError(err)
	s.False(updated.AnalyticsExcluded)
}

// An omitted analyticsExcluded must leave the stored value alone, otherwise a
// partial update would silently re-enable tracking.
func (s *HandlerUpdateDomainSuite) Test_OmittedFieldKeepsValue() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	domain := s.mockDomain(app.ID, env.ID)

	domain.AnalyticsExcluded = true
	s.NoError(buildconf.DomainStore().UpdateDomainFlags(context.Background(), domain))

	s.mockCache.On("Reset", types.ID(0), "example.org").Return(nil)

	code := s.request(usr.ID, map[string]any{
		"appId":    app.ID.String(),
		"envId":    env.ID.String(),
		"domainId": domain.ID.String(),
	})

	s.Equal(http.StatusOK, code)

	updated, err := buildconf.DomainStore().DomainByID(context.Background(), domain.ID)

	s.NoError(err)
	s.True(updated.AnalyticsExcluded)

	// A no-op update must not purge the host config cache on every node.
	s.mockCache.AssertNotCalled(s.T(), "Reset", types.ID(0), "example.org")
}

// Re-sending the value the domain already holds is a no-op: no cache purge and
// no audit entry whose old and new values are identical.
func (s *HandlerUpdateDomainSuite) Test_UnchangedValueIsNoOp() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	domain := s.mockDomain(app.ID, env.ID)

	code := s.request(usr.ID, map[string]any{
		"appId":             app.ID.String(),
		"envId":             env.ID.String(),
		"domainId":          domain.ID.String(),
		"analyticsExcluded": false,
	})

	s.Equal(http.StatusOK, code)
	s.mockCache.AssertNotCalled(s.T(), "Reset", types.ID(0), "example.org")
}

func (s *HandlerUpdateDomainSuite) Test_MissingDomainID() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	code := s.request(usr.ID, map[string]any{
		"appId":             app.ID.String(),
		"envId":             env.ID.String(),
		"analyticsExcluded": true,
	})

	s.Equal(http.StatusBadRequest, code)
}

// A domain belonging to another app must not be reachable through this app's
// credentials.
func (s *HandlerUpdateDomainSuite) Test_DomainOfAnotherApp() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	otherApp := s.MockApp(usr)
	otherEnv := s.MockEnv(otherApp)
	domain := s.mockDomain(otherApp.ID, otherEnv.ID)

	code := s.request(usr.ID, map[string]any{
		"appId":             app.ID.String(),
		"envId":             env.ID.String(),
		"domainId":          domain.ID.String(),
		"analyticsExcluded": true,
	})

	s.Equal(http.StatusNotFound, code)

	updated, err := buildconf.DomainStore().DomainByID(context.Background(), domain.ID)

	s.NoError(err)
	s.False(updated.AnalyticsExcluded)
}

// Credentials scoped to one environment must not reach a domain belonging to a
// sibling environment, even though both environments share the same app.
func (s *HandlerUpdateDomainSuite) Test_DomainOfAnotherEnv() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	otherEnv := s.MockEnv(app, map[string]any{"Name": "staging", "Branch": "staging"})
	domain := s.mockDomain(app.ID, otherEnv.ID)

	code := s.request(usr.ID, map[string]any{
		"appId":             app.ID.String(),
		"envId":             env.ID.String(),
		"domainId":          domain.ID.String(),
		"analyticsExcluded": true,
	})

	s.Equal(http.StatusNotFound, code)

	updated, err := buildconf.DomainStore().DomainByID(context.Background(), domain.ID)

	s.NoError(err)
	s.False(updated.AnalyticsExcluded)
}

func TestHandlerUpdateDomain(t *testing.T) {
	suite.Run(t, &HandlerUpdateDomainSuite{})
}
