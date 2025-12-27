package apphandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type HandleAppInsertSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandleAppInsertSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandleAppInsertSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandleAppInsertSuite) checkAudits(usr *factory.MockUser, appl *app.App) {
	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		TeamID: appl.TeamID,
	})

	s.NoError(err)
	s.Len(audits, 1)
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "CREATE:APP",
		TeamID:      appl.TeamID,
		UserID:      usr.ID,
		UserDisplay: usr.Display(),
		Diff: &audit.Diff{
			New: audit.DiffFields{
				AppName: appl.DisplayName,
				AppRepo: appl.Repo,
			},
		},
	}, audits[0])
}

func (s *HandleAppInsertSuite) Test_GithubRepo() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app",
		map[string]string{
			"repo":     "stormkit-test-acc/test-repo",
			"provider": "github",
			"teamId":   usr.DefaultTeamID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	body := response.String()

	s.Contains(body, `"repo":"github/stormkit-test-acc/test-repo"`)

	apps, err := app.NewStore().Apps(context.Background(), app.AppsArgs{
		TeamID: usr.DefaultTeamID,
		From:   0,
		Limit:  10,
	})

	s.NoError(err)
	s.Len(apps, 1)

	env, err := buildconf.NewStore().EnvironmentByID(context.Background(), apps[0].ID)
	s.NoError(err)
	s.True(env.AutoDeploy)

	s.checkAudits(usr, apps[0])
}

func (s *HandleAppInsertSuite) Test_BitbucketRepo() {
	utils.SetKeySize(2056)
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app",
		map[string]string{
			"repo":     "stormkit-test/test-repo",
			"provider": "bitbucket",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	body := response.String()
	s.Contains(body, `"repo":"bitbucket/stormkit-test/test-repo"`)

	apps, err := app.NewStore().Apps(context.Background(), app.AppsArgs{
		TeamID: usr.DefaultTeamID,
		From:   0,
		Limit:  10,
	})

	s.NoError(err)
	s.Len(apps, 1)

	env, err := buildconf.NewStore().EnvironmentByID(context.Background(), apps[0].ID)
	s.NoError(err)
	s.True(env.AutoDeploy)

	s.checkAudits(usr, apps[0])
}

// A Bare App is a an app that is created without a repo.
func (s *HandleAppInsertSuite) Test_BareApp() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app",
		map[string]string{
			"teamId": usr.DefaultTeamID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.Contains(response.String(), `"repo":""`)

	apps, err := app.NewStore().Apps(context.Background(), app.AppsArgs{
		TeamID: usr.DefaultTeamID,
		From:   0,
		Limit:  10,
	})

	s.NoError(err)
	s.Len(apps, 1)
	s.Equal("", apps[0].Repo)
	s.Equal(usr.DefaultTeamID, apps[0].TeamID)
}

func (s *HandleAppInsertSuite) Test_InvalidRepoProvider() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app",
		map[string]string{
			"repo":     "stormkit-test/test-repo",
			"provider": "invalid-provider",
			"teamId":   usr.DefaultTeamID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{"errors":{"provider":"The provider can only be github, gitlab or bitbucket."}}`
	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal(expected, response.String())

	apps, err := app.NewStore().Apps(context.Background(), app.AppsArgs{
		TeamID: usr.DefaultTeamID,
		From:   0,
		Limit:  10,
	})

	s.NoError(err)
	s.Len(apps, 0)
}

func TestHandleAppInsert(t *testing.T) {
	suite.Run(t, &HandleAppInsertSuite{})
}
