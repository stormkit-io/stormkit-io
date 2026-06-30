package snippetshandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/snippetshandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerSnippetsAnalyticsSuite struct {
	suite.Suite
	*factory.Factory

	conn             databasetest.TestDB
	mockCacheService *mocks.CacheInterface
}

func (s *HandlerSnippetsAnalyticsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockCacheService = &mocks.CacheInterface{}
	appcache.DefaultCacheService = s.mockCacheService
	admin.SetMockLicense()
}

func (s *HandlerSnippetsAnalyticsSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	appcache.DefaultCacheService = nil
	admin.ResetMockLicense()
}

func (s *HandlerSnippetsAnalyticsSuite) enable(usr, appID, envID string) shttptest.Response {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/snippets/analytics",
		map[string]any{"appId": appID, "envId": envID},
		map[string]string{"Authorization": usr},
	)
}

func (s *HandlerSnippetsAnalyticsSuite) disable(usr, appID, envID string) shttptest.Response {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/snippets/analytics?envId=%s&appId=%s", envID, appID),
		nil,
		map[string]string{"Authorization": usr},
	)
}

func (s *HandlerSnippetsAnalyticsSuite) Test_Enable_CreatesManagedSnippet() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	s.mockCacheService.On("Reset", env.ID).Return(nil)

	res := s.enable(usertest.Authorization(usr.ID), app.ID.String(), env.ID.String())
	s.Equal(http.StatusCreated, res.Code)

	snippets, err := buildconf.SnippetsStore().SnippetsByEnvID(context.Background(), buildconf.SnippetFilters{EnvID: env.ID})
	s.NoError(err)
	s.Require().Len(snippets, 1)
	s.Equal(snippetshandlers.AnalyticsSnippetTitle, snippets[0].Title)
	s.Equal("head", snippets[0].Location)
	s.True(snippets[0].Enabled)
	s.True(snippets[0].Interpolate)
	s.Contains(snippets[0].Content, "{{SK_REQUEST_ID}}")
	s.Contains(snippets[0].Content, "/_stormkit/analytics.js")
}

func (s *HandlerSnippetsAnalyticsSuite) Test_Enable_Idempotent() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	s.mockCacheService.On("Reset", env.ID).Return(nil)

	auth := usertest.Authorization(usr.ID)
	s.Equal(http.StatusCreated, s.enable(auth, app.ID.String(), env.ID.String()).Code)
	s.Equal(http.StatusOK, s.enable(auth, app.ID.String(), env.ID.String()).Code)

	snippets, err := buildconf.SnippetsStore().SnippetsByEnvID(context.Background(), buildconf.SnippetFilters{EnvID: env.ID})
	s.NoError(err)
	s.Len(snippets, 1)
}

func (s *HandlerSnippetsAnalyticsSuite) Test_Disable_RemovesSnippet() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	s.mockCacheService.On("Reset", env.ID).Return(nil)

	auth := usertest.Authorization(usr.ID)
	s.Equal(http.StatusCreated, s.enable(auth, app.ID.String(), env.ID.String()).Code)
	s.Equal(http.StatusOK, s.disable(auth, app.ID.String(), env.ID.String()).Code)

	snippets, err := buildconf.SnippetsStore().SnippetsByEnvID(context.Background(), buildconf.SnippetFilters{EnvID: env.ID})
	s.NoError(err)
	s.Len(snippets, 0)
}

func (s *HandlerSnippetsAnalyticsSuite) Test_Disable_WhenAbsentIsOK() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	res := s.disable(usertest.Authorization(usr.ID), app.ID.String(), env.ID.String())
	s.Equal(http.StatusOK, res.Code)
}

func (s *HandlerSnippetsAnalyticsSuite) Test_Enable_RejectsNonEnterprise() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	admin.ResetMockLicense()

	res := s.enable(usertest.Authorization(usr.ID), app.ID.String(), env.ID.String())
	s.Equal(http.StatusForbidden, res.Code)
}

func TestHandlerSnippetsAnalytics(t *testing.T) {
	suite.Run(t, &HandlerSnippetsAnalyticsSuite{})
}
