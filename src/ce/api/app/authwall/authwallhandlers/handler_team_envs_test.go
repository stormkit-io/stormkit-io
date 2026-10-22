package authwallhandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type HandlerTeamEnvsSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	tf   *teamFixture
}

func (s *HandlerTeamEnvsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.tf = newTeamFixture(&s.Suite, s.Factory)
	admin.SetMockLicense()
}

func (s *HandlerTeamEnvsSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

// Test_Success verifies that every member sees the team's apps with the mode
// of each environment's wall, and nothing of other teams.
func (s *HandlerTeamEnvsSuite) Test_Success() {
	s.Require().NoError(authwall.Store().SetAuthWallConfig(context.Background(), s.tf.env.ID, &authwall.Config{Status: authwall.StatusDev}))
	newTeamFixture(&s.Suite, s.Factory)

	response := s.tf.request(s.tf.developer, shttp.MethodGet, fmt.Sprintf("/team/auth-wall/envs?teamId=%s", s.tf.team.ID), nil)

	s.Equal(http.StatusOK, response.Code)

	apps := response.Map()["apps"].([]any)
	s.Require().Len(apps, 1, "an app without environments is not listed")

	app := apps[0].(map[string]any)
	s.Equal(s.tf.app.ID.String(), app["id"])
	s.Equal([]any{
		map[string]any{"id": s.tf.env.ID.String(), "name": s.tf.env.Name, "authWall": "dev"},
	}, app["environments"])
}

func (s *HandlerTeamEnvsSuite) Test_Forbidden_NotMember() {
	response := s.tf.request(s.MockUser(), shttp.MethodGet, fmt.Sprintf("/team/auth-wall/envs?teamId=%s", s.tf.team.ID), nil)
	s.Equal(http.StatusForbidden, response.Code)
}

func TestHandlerTeamEnvsSuite(t *testing.T) {
	suite.Run(t, &HandlerTeamEnvsSuite{})
}
