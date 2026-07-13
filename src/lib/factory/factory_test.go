package factory_test

import (
	"context"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type FactorySuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *FactorySuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *FactorySuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *FactorySuite) TestMockUserDefault() {
	a := assert.New(s.T())

	s.MockUser()

	usr, err := user.NewStore().UserByID(1)
	a.NoError(err)
	a.Equal(usr.ID, types.ID(1))
	a.Equal(usr.FirstName.ValueOrZero(), "David")
}

func (s *FactorySuite) TestMockUserWithExtend() {
	a := assert.New(s.T())

	s.MockUser(map[string]any{
		"FirstName": null.NewString("Leonardo", true),
	})

	usr, err := user.NewStore().UserByID(1)
	a.NoError(err)
	a.Equal(usr.ID, types.ID(1))
	a.Equal(usr.FirstName.ValueOrZero(), "Leonardo")
}

func (s *FactorySuite) TestMockApp() {
	a := assert.New(s.T())

	s.MockApp(nil, map[string]any{
		"Repo": "bitbucket/my/test/repo",
	})

	appl, err := app.NewStore().AppByID(context.Background(), 1)
	a.NoError(err)
	a.Equal(types.ID(1), appl.ID)
	a.Equal("bitbucket/my/test/repo", appl.Repo)
}

func (s *FactorySuite) TestMockAppWithCustomUser() {
	a := assert.New(s.T())

	s.MockApp(
		s.MockUser(map[string]any{
			"FirstName": null.NewString("Foo", true),
			"LastName":  null.NewString("Bar", true),
		}),
	)

	appl, err := app.NewStore().AppByID(context.Background(), 1)
	a.NoError(err)
	a.Equal(appl.ID, types.ID(1))
	a.Equal(appl.Repo, "github/svedova/react-minimal")

	usr, err := user.NewStore().UserByID(1)
	a.NoError(err)
	a.Equal(types.ID(1), usr.ID)
	a.Equal("Foo", usr.FirstName.ValueOrZero())
	a.Equal("Bar", usr.LastName.ValueOrZero())
}

func (s *FactorySuite) TestMockEnv() {
	a := assert.New(s.T())

	s.MockEnv(nil, map[string]any{
		"Branch": "release",
	})

	env, err := buildconf.NewStore().EnvironmentByID(context.Background(), 1)
	a.NoError(err)
	a.Equal(types.ID(1), env.ID)
	a.Equal("release", env.Branch)
	a.Equal("production", env.Name)
}

func (s *FactorySuite) Test_GettingAppAndEnv() {
	app := s.MockApp(nil)
	env := s.MockEnv(app)

	s.Equal(app.ID, s.GetApp().ID)
	s.Equal(env.ID, s.GetEnv().ID)
	s.Equal(env.AppID, app.ID)
}

func (s *FactorySuite) Test_MockDeployment() {
	a := assert.New(s.T())

	s.MockDeployment(nil)

	depl, err := deploy.NewStore().DeploymentByID(context.Background(), 1)

	a.NoError(err)
	a.NotNil(depl)
	a.Equal(types.ID(1), depl.ID)
	a.Equal(types.ID(1), depl.AppID)
}

func TestFactory(t *testing.T) {
	suite.Run(t, &FactorySuite{})
}
