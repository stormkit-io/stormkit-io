package authwallhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type HandlerTeamEnvUpdateSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	tf   *teamFixture
}

func (s *HandlerTeamEnvUpdateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.tf = newTeamFixture(&s.Suite, s.Factory)
	admin.SetMockLicense()
}

func (s *HandlerTeamEnvUpdateSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

// update sets the wall of the environment as usr.
func (s *HandlerTeamEnvUpdateSuite) update(usr *factory.MockUser, envID types.ID, status string) shttptest.Response {
	return s.tf.request(usr, shttp.MethodPut, "/team/auth-wall/envs", map[string]any{
		"teamId":   s.tf.team.ID.String(),
		"envId":    envID.String(),
		"authWall": status,
	})
}

func (s *HandlerTeamEnvUpdateSuite) status(envID types.ID) string {
	cfg, err := authwall.Store().AuthWallConfig(context.Background(), envID)
	s.Require().NoError(err)

	if cfg == nil {
		return ""
	}

	return cfg.Status
}

// Test_Success_Developer verifies that developers can switch the wall of the
// team's environments.
func (s *HandlerTeamEnvUpdateSuite) Test_Success_Developer() {
	response := s.update(s.tf.developer, s.tf.env.ID, authwall.StatusAll)

	s.Equal(http.StatusOK, response.Code)
	s.Equal(authwall.StatusAll, s.status(s.tf.env.ID))

	entry := s.tf.lastAudit()
	s.Require().NotNil(entry)
	s.Equal("UPDATE:AUTHWALL", entry.Action)
	s.Equal(s.tf.app.ID, entry.AppID)
	s.Equal(s.tf.env.ID, entry.EnvID)
	s.Equal("off", entry.Diff.Old.AuthWallStatus)
	s.Equal("all", entry.Diff.New.AuthWallStatus)

	s.Equal(http.StatusOK, s.update(s.tf.developer, s.tf.env.ID, authwall.StatusDisabled).Code)
	s.Equal("", s.status(s.tf.env.ID))
}

func (s *HandlerTeamEnvUpdateSuite) Test_BadRequest_Status() {
	s.Equal(http.StatusBadRequest, s.update(s.tf.owner, s.tf.env.ID, "everything").Code)
}

// Test_NotFound_OtherTeam verifies that a team cannot switch the wall of
// another team's environment.
func (s *HandlerTeamEnvUpdateSuite) Test_NotFound_OtherTeam() {
	other := newTeamFixture(&s.Suite, s.Factory)

	s.Equal(http.StatusNotFound, s.update(s.tf.owner, other.env.ID, authwall.StatusAll).Code)
	s.Equal("", s.status(other.env.ID))
}

func (s *HandlerTeamEnvUpdateSuite) Test_Forbidden_NotMember() {
	s.Equal(http.StatusForbidden, s.update(s.MockUser(), s.tf.env.ID, authwall.StatusAll).Code)
}

func TestHandlerTeamEnvUpdateSuite(t *testing.T) {
	suite.Run(t, &HandlerTeamEnvUpdateSuite{})
}
