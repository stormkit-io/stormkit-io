package authwallhandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerTeamLoginUpdateSuite struct {
	suite.Suite
	*factory.Factory
	conn  databasetest.TestDB
	tf    *teamFixture
	login *authwall.AuthWall
}

func (s *HandlerTeamLoginUpdateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.tf = newTeamFixture(&s.Suite, s.Factory)
	s.login = s.tf.login("client@example.org")
	admin.SetMockLicense()
}

func (s *HandlerTeamLoginUpdateSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

// update patches the suite's login as usr.
func (s *HandlerTeamLoginUpdateSuite) update(usr *factory.MockUser, body map[string]any) shttptest.Response {
	body["teamId"] = s.tf.team.ID.String()

	if _, ok := body["id"]; !ok {
		body["id"] = s.login.LoginID.String()
	}

	return s.tf.request(usr, shttp.MethodPatch, "/team/auth-wall/logins", body)
}

func (s *HandlerTeamLoginUpdateSuite) Test_Success_Apps() {
	otherEnv := s.MockEnv(s.tf.otherApp)
	response := s.update(s.tf.owner, map[string]any{"appIds": []string{s.tf.otherApp.ID.String()}})

	s.Equal(http.StatusOK, response.Code)
	s.False(s.tf.opens(s.tf.env.ID, "client@example.org", "123pass-word"))
	s.True(s.tf.opens(otherEnv.ID, "client@example.org", "123pass-word"))

	entry := s.tf.lastAudit()
	s.Require().NotNil(entry)
	s.Equal("UPDATE:AUTHWALL_LOGIN", entry.Action)
	s.Equal(s.tf.app.ID.String(), entry.Diff.Old.AuthWallLoginApps)
	s.Equal(s.tf.otherApp.ID.String(), entry.Diff.New.AuthWallLoginApps)
}

// Test_Success_AllApps verifies that a login switched to all apps opens every
// app of the team, and switched back opens only the apps listed.
func (s *HandlerTeamLoginUpdateSuite) Test_Success_AllApps() {
	otherEnv := s.MockEnv(s.tf.otherApp)

	s.Equal(http.StatusOK, s.update(s.tf.owner, map[string]any{"allApps": true}).Code)
	s.True(s.tf.opens(otherEnv.ID, "client@example.org", "123pass-word"))

	s.Equal(http.StatusOK, s.update(s.tf.owner, map[string]any{"allApps": false, "appIds": []string{s.tf.app.ID.String()}}).Code)
	s.False(s.tf.opens(otherEnv.ID, "client@example.org", "123pass-word"))
	s.True(s.tf.opens(s.tf.env.ID, "client@example.org", "123pass-word"))
}

// Test_Success_NarrowAllApps verifies that sending apps to a login that opens
// all apps limits it to those apps.
func (s *HandlerTeamLoginUpdateSuite) Test_Success_NarrowAllApps() {
	otherEnv := s.MockEnv(s.tf.otherApp)

	s.Equal(http.StatusOK, s.update(s.tf.owner, map[string]any{"allApps": true}).Code)
	s.Equal(http.StatusOK, s.update(s.tf.owner, map[string]any{"appIds": []string{s.tf.otherApp.ID.String()}}).Code)

	s.False(s.tf.opens(s.tf.env.ID, "client@example.org", "123pass-word"))
	s.True(s.tf.opens(otherEnv.ID, "client@example.org", "123pass-word"))
}

// Test_Success_DeletedApp verifies that a login that opened a since-deleted
// app can still be changed.
func (s *HandlerTeamLoginUpdateSuite) Test_Success_DeletedApp() {
	s.Equal(http.StatusOK, s.update(s.tf.owner, map[string]any{"appIds": []string{s.tf.app.ID.String(), s.tf.otherApp.ID.String()}}).Code)

	_, err := s.conn.Exec("UPDATE apps SET deleted_at = NOW() WHERE app_id = $1", s.tf.otherApp.ID)
	s.Require().NoError(err)

	s.Equal(http.StatusOK, s.update(s.tf.owner, map[string]any{"allApps": false}).Code)
}

// Test_Success_PasswordAndMetadata verifies that the password and metadata
// change while the app access is kept.
func (s *HandlerTeamLoginUpdateSuite) Test_Success_PasswordAndMetadata() {
	response := s.update(s.tf.owner, map[string]any{"password": "new-password", "metadata": "vip"})

	s.Equal(http.StatusOK, response.Code)
	s.False(s.tf.opens(s.tf.env.ID, "client@example.org", "123pass-word"))
	s.True(s.tf.opens(s.tf.env.ID, "client@example.org", "new-password"))

	login, err := authwall.Store().TeamLogin(s.T().Context(), s.tf.team.ID, s.login.LoginID)
	s.Require().NoError(err)
	s.Equal("vip", login.Metadata)

	entry := s.tf.lastAudit()
	s.Require().NotNil(entry)
	s.True(*entry.Diff.New.AuthWallPasswordChanged)
	s.Equal("vip", entry.Diff.New.AuthWallLoginMetadata)
}

func (s *HandlerTeamLoginUpdateSuite) Test_BadRequest() {
	for _, body := range []map[string]any{
		{"allApps": false, "appIds": []string{}},
		{"appIds": []string{s.MockApp(s.MockUser()).ID.String()}},
		{"password": "short"},
		{"metadata": "line\nbreak"},
	} {
		s.Equal(http.StatusBadRequest, s.update(s.tf.owner, body).Code, body)
	}

	s.True(s.tf.opens(s.tf.env.ID, "client@example.org", "123pass-word"))
}

// Test_NotFound_OtherTeam verifies that a team cannot change another team's
// login.
func (s *HandlerTeamLoginUpdateSuite) Test_NotFound_OtherTeam() {
	other := newTeamFixture(&s.Suite, s.Factory)
	foreign := other.login("foreign@example.org")

	response := s.update(s.tf.owner, map[string]any{"id": foreign.LoginID.String(), "password": "new-password"})

	s.Equal(http.StatusNotFound, response.Code)
	s.True(other.opens(other.env.ID, "foreign@example.org", "123pass-word"))
}

func (s *HandlerTeamLoginUpdateSuite) Test_Forbidden() {
	response := s.update(s.tf.developer, map[string]any{"allApps": true})
	s.Equal(http.StatusForbidden, response.Code)
}

func TestHandlerTeamLoginUpdateSuite(t *testing.T) {
	suite.Run(t, &HandlerTeamLoginUpdateSuite{})
}
