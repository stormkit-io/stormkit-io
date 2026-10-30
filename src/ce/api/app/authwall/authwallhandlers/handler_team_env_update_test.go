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

// put sends the body to the update endpoint as usr, with the fixture's team.
func (s *HandlerTeamEnvUpdateSuite) put(usr *factory.MockUser, body map[string]any) shttptest.Response {
	body["teamId"] = s.tf.team.ID.String()

	return s.tf.request(usr, shttp.MethodPut, "/team/auth-wall/envs", body)
}

func (s *HandlerTeamEnvUpdateSuite) config(envID types.ID) *authwall.Config {
	cfg, err := authwall.Store().AuthWallConfig(context.Background(), envID)
	s.Require().NoError(err)

	return cfg
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

// Test_PublicPaths verifies that public paths are saved cleaned up, survive a
// change of mode, and are audited.
func (s *HandlerTeamEnvUpdateSuite) Test_PublicPaths() {
	response := s.put(s.tf.developer, map[string]any{
		"envId":       s.tf.env.ID.String(),
		"authWall":    authwall.StatusAll,
		"publicPaths": []string{" /api/webhooks/* ", "/robots.txt", "/robots.txt", ""},
	})

	s.Equal(http.StatusOK, response.Code)
	s.Equal([]string{"/api/webhooks/*", "/robots.txt"}, s.config(s.tf.env.ID).PublicPaths)

	entry := s.tf.lastAudit()
	s.Require().NotNil(entry)
	s.Empty(entry.Diff.Old.AuthWallPublicPaths)
	s.Equal([]string{"/api/webhooks/*", "/robots.txt"}, entry.Diff.New.AuthWallPublicPaths)

	s.Equal(http.StatusOK, s.update(s.tf.developer, s.tf.env.ID, authwall.StatusDev).Code)

	cfg := s.config(s.tf.env.ID)
	s.Equal(authwall.StatusDev, cfg.Status)
	s.Equal([]string{"/api/webhooks/*", "/robots.txt"}, cfg.PublicPaths)

	response = s.put(s.tf.developer, map[string]any{
		"envId":       s.tf.env.ID.String(),
		"publicPaths": []string{},
	})

	s.Equal(http.StatusOK, response.Code)

	cfg = s.config(s.tf.env.ID)
	s.Equal(authwall.StatusDev, cfg.Status)
	s.Empty(cfg.PublicPaths)
}

func (s *HandlerTeamEnvUpdateSuite) Test_BadRequest_PublicPaths() {
	for _, paths := range [][]string{
		{"api/webhooks"},
		{"/api/*/hooks"},
		{"/api*"},
		{"/*"},
		{"/api/../admin"},
		{"/search?q=1"},
	} {
		response := s.put(s.tf.owner, map[string]any{
			"envId":       s.tf.env.ID.String(),
			"publicPaths": paths,
		})

		s.Equal(http.StatusBadRequest, response.Code, paths)
	}

	s.Empty(s.config(s.tf.env.ID).PublicPaths)
}

// Test_BadRequest_NothingToUpdate verifies that a request without a mode does
// not turn the wall off.
func (s *HandlerTeamEnvUpdateSuite) Test_BadRequest_NothingToUpdate() {
	s.Equal(http.StatusOK, s.update(s.tf.owner, s.tf.env.ID, authwall.StatusAll).Code)
	s.Equal(http.StatusBadRequest, s.put(s.tf.owner, map[string]any{"envId": s.tf.env.ID.String()}).Code)
	s.Equal(authwall.StatusAll, s.status(s.tf.env.ID))
}

func TestHandlerTeamEnvUpdateSuite(t *testing.T) {
	suite.Run(t, &HandlerTeamEnvUpdateSuite{})
}
