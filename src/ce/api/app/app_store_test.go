package app_test

import (
	"context"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type AppStoreSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *AppStoreSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *AppStoreSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *AppStoreSuite) Test_AppByEnvID() {
	usr := s.MockUser()
	apl := s.MockApp(usr)
	env := s.MockEnv(apl)

	myApp, err := app.NewStore().AppByEnvID(context.Background(), env.ID)
	s.NoError(err)
	s.NotEmpty(myApp)
	s.Equal(apl.DisplayName, myApp.DisplayName)
	s.Equal(apl.ID, myApp.ID)
}

func (s *AppStoreSuite) Test_AppByDisplayName() {
	usr := s.MockUser()
	apl := s.MockApp(usr)

	myApp, err := app.NewStore().AppByDisplayName(context.Background(), apl.DisplayName)
	s.NoError(err)
	s.NotEmpty(myApp)
	s.Equal(apl.DisplayName, myApp.DisplayName)
	s.Equal(apl.ID, myApp.ID)
}

func (s *AppStoreSuite) Test_AppByDomainName() {
	usr := s.MockUser()
	apl := s.MockApp(usr)
	env := s.MockEnv(apl)

	domain := &buildconf.DomainModel{
		AppID:      apl.ID,
		EnvID:      env.ID,
		Name:       "www.stormkit.io",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
		Token:      null.StringFrom("my-custom-token"),
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domain))

	myApp, err := app.NewStore().AppByDomainName(context.Background(), "www.stormkit.io")
	s.NoError(err)
	s.NotEmpty(myApp)
	s.Equal(apl.DisplayName, myApp.DisplayName)
	s.Equal(apl.ID, myApp.ID)
}

func (s *AppStoreSuite) Test_DeployCandidates() {
	appl := s.MockApp(nil, map[string]any{
		"AutoDeploy": null.NewString("", false),
	})

	s.MockEnv(appl, map[string]any{
		"Name":       "production",
		"AutoDeploy": false,
	})

	envDev := s.MockEnv(appl, map[string]any{
		"Name":       "development",
		"AutoDeploy": true,
		"MailerConf": &buildconf.MailerConf{
			Username: "test",
			Password: "test-pwd",
			Host:     "smtp.gmail.com",
			Port:     "587",
		},
	})

	candidates, err := app.NewStore().DeployCandidates(context.Background(), appl.Repo)
	s.NoError(err)
	s.Len(candidates, 1)
	s.Equal(envDev.Name, candidates[0].EnvName)
	s.NotNil(candidates[0].MailerConf)
	s.Equal("smtp://test:test-pwd@smtp.gmail.com:587", candidates[0].MailerConf.String())
}

func (s *AppStoreSuite) Test_Apps_FilterByRepo() {
	usr := s.MockUser()
	apl := s.MockApp(usr)
	s.MockApp(usr, map[string]any{"Repo": "github/other/unrelated"})

	apps, err := app.NewStore().Apps(context.Background(), app.AppsArgs{
		TeamID: usr.DefaultTeamID,
		Filter: apl.Repo,
	})
	s.NoError(err)
	s.Len(apps, 1)
	s.Equal(apl.ID, apps[0].ID)
}

func (s *AppStoreSuite) Test_Apps_FilterByDisplayName() {
	usr := s.MockUser()
	apl := s.MockApp(usr)
	s.MockApp(usr)

	apps, err := app.NewStore().Apps(context.Background(), app.AppsArgs{
		TeamID: usr.DefaultTeamID,
		Filter: apl.DisplayName,
	})
	s.NoError(err)
	s.Len(apps, 1)
	s.Equal(apl.ID, apps[0].ID)
}

func (s *AppStoreSuite) Test_Apps_ExactRepo() {
	usr := s.MockUser()
	apl := s.MockApp(usr, map[string]any{"Repo": "github/acme/exact-repo"})
	s.MockApp(usr, map[string]any{"Repo": "github/acme/exact-repo-other"})

	apps, err := app.NewStore().Apps(context.Background(), app.AppsArgs{
		TeamID: usr.DefaultTeamID,
		Repo:   "github/acme/exact-repo",
	})
	s.NoError(err)
	s.Len(apps, 1)
	s.Equal(apl.ID, apps[0].ID)
}

func (s *AppStoreSuite) Test_Apps_ExactRepo_CaseInsensitive() {
	usr := s.MockUser()
	apl := s.MockApp(usr, map[string]any{"Repo": "github/acme/My-App"})

	apps, err := app.NewStore().Apps(context.Background(), app.AppsArgs{
		TeamID: usr.DefaultTeamID,
		Repo:   "GITHUB/ACME/MY-APP",
	})
	s.NoError(err)
	s.Len(apps, 1)
	s.Equal(apl.ID, apps[0].ID)
}

func (s *AppStoreSuite) Test_Apps_ExactDisplayName() {
	usr := s.MockUser()
	apl := s.MockApp(usr)
	s.MockApp(usr) // different display name

	apps, err := app.NewStore().Apps(context.Background(), app.AppsArgs{
		TeamID:      usr.DefaultTeamID,
		DisplayName: apl.DisplayName,
	})
	s.NoError(err)
	s.Len(apps, 1)
	s.Equal(apl.ID, apps[0].ID)
}

func (s *AppStoreSuite) Test_Apps_ExactRepoAndDisplayName() {
	usr := s.MockUser()
	apl := s.MockApp(usr, map[string]any{"Repo": "github/acme/combo"})
	// Same repo, different display name:
	s.MockApp(usr, map[string]any{"Repo": "github/acme/combo"})

	apps, err := app.NewStore().Apps(context.Background(), app.AppsArgs{
		TeamID:      usr.DefaultTeamID,
		Repo:        "github/acme/combo",
		DisplayName: apl.DisplayName,
	})
	s.NoError(err)
	s.Len(apps, 1)
	s.Equal(apl.ID, apps[0].ID)
}

func TestAppStore(t *testing.T) {
	suite.Run(t, &AppStoreSuite{})
}
