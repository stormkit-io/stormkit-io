package apphandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/mocks"
)

type AppUpdateSuite struct {
	suite.Suite
	*factory.Factory

	conn             databasetest.TestDB
	mockCacheService *mocks.CacheInterface
}

func (s *AppUpdateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockCacheService = &mocks.CacheInterface{}
	appcache.DefaultCacheService = s.mockCacheService
	admin.SetMockLicense()
}

func (s *AppUpdateSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	appcache.DefaultCacheService = nil
	admin.ResetMockLicense()
}

func (s *AppUpdateSuite) Test_SuccessGithub() {
	// Set self-hosted to avoid license checks
	config.SetIsSelfHosted(true)

	usr := s.GetUser()
	mockApp := s.MockApp(usr)

	s.mockCacheService.On("Reset",
		types.ID(0),
		fmt.Sprintf("^%s(?:--\\d+)?", mockApp.DisplayName),
		"^stormkit-io(?:--\\d+)?").
		Return(nil).
		Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/app",
		map[string]any{
			"appId":       mockApp.ID.String(),
			"repo":        "github/stormkit-io/test-repo-update",
			"displayName": "stormkit-io",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	a := assert.New(s.T())
	a.Equal(http.StatusOK, response.Code)

	appl, err := app.NewStore().AppByID(context.Background(), mockApp.ID)
	a.NoError(err)
	a.Equal("github/stormkit-io/test-repo-update", appl.Repo)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		AppID: mockApp.ID,
	})

	s.NoError(err)
	s.Len(audits, 1)
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "UPDATE:APP",
		AppID:       mockApp.ID,
		TeamID:      mockApp.TeamID,
		UserID:      usr.ID,
		UserDisplay: usr.Display(),
		Diff: &audit.Diff{
			Old: audit.DiffFields{
				AppName: mockApp.DisplayName,
				AppRepo: mockApp.Repo,
			},
			New: audit.DiffFields{
				AppName: "stormkit-io",
				AppRepo: "github/stormkit-io/test-repo-update",
			},
		},
	}, audits[0])
}

func (s *AppUpdateSuite) Test_GitlabSuccess() {
	usr := s.GetUser()
	mockApp := s.MockApp(usr)

	s.mockCacheService.On("Reset",
		types.ID(0),
		fmt.Sprintf("^%s(?:--\\d+)?", mockApp.DisplayName),
		"^stormkit-io(?:--\\d+)?").
		Return(nil).
		Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/app",
		map[string]interface{}{
			"appId":       mockApp.ID.String(),
			"repo":        "gitlab/stormkit-io/test-repo-update",
			"displayName": "stormkit-io",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	appl, err := app.NewStore().AppByID(context.Background(), mockApp.ID)
	s.NoError(err)
	s.Equal("gitlab/stormkit-io/test-repo-update", appl.Repo)
}

func (s *AppUpdateSuite) Test_DuplicateDisplayName() {
	usr := s.GetUser()
	app1 := s.MockApp(usr, map[string]interface{}{"DisplayName": "my-app"})
	app2 := s.MockApp(usr)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/app",
		map[string]interface{}{
			"appId":       app2.ID.String(),
			"repo":        "github/stormkit-io/test-repo-update",
			"displayName": app1.DisplayName,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := fmt.Sprintf(`{"errors":{"displayName":"%s"},"ok":false}`, app.ErrDuplicateDisplayName.Error())
	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal(expected, response.String())
}

func (s *AppUpdateSuite) Test_SwitchinToBareApp() {
	usr := s.MockUser()
	app := s.MockApp(usr)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/app",
		map[string]any{
			"appId": app.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{
		"error": "It's not possible to convert an existing app to a bare app. Please create a new app instead." 
	}`

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *AppUpdateSuite) Test_InvalidDisplayName() {
	usr := s.GetUser()
	appl := s.MockApp(usr)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/app",
		map[string]interface{}{
			"appId":       appl.ID.String(),
			"repo":        "github/stormkit-io/test-repo-update",
			"displayName": "invalid display-name",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"errors": { "displayName": "%s" },
		"ok": false
	}`, app.ErrInvalidDisplayName.Error())

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *AppUpdateSuite) Test_InvalidRepoProvider() {
	usr := s.GetUser()
	appl := s.MockApp(usr)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/app",
		map[string]any{
			"appId":       appl.ID.String(),
			"repo":        "unknown.org/stormkit-io/app-www.git",
			"displayName": "my-display",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := fmt.Sprintf(`{"errors":{"repo":"%s"},"ok":false}`, app.ErrRepoInvalidProvider.Error())
	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal(expected, response.String())
}

func TestHandlerAppUpdate(t *testing.T) {
	suite.Run(t, &AppUpdateSuite{})
}
