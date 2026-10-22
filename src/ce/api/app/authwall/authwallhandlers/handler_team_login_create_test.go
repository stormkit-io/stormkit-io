package authwallhandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerTeamLoginCreateSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	tf   *teamFixture
}

func (s *HandlerTeamLoginCreateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.tf = newTeamFixture(&s.Suite, s.Factory)
	admin.SetMockLicense()
}

func (s *HandlerTeamLoginCreateSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

// create posts a login to the team as usr, filling in the team and
// credentials.
func (s *HandlerTeamLoginCreateSuite) create(usr *factory.MockUser, body map[string]any) shttptest.Response {
	body["teamId"] = s.tf.team.ID.String()

	if _, ok := body["email"]; !ok {
		body["email"] = "client@example.org"
	}

	body["password"] = "client-password"

	return s.tf.request(usr, shttp.MethodPost, "/team/auth-wall/logins", body)
}

func (s *HandlerTeamLoginCreateSuite) Test_Success() {
	response := s.create(s.tf.owner, map[string]any{
		"metadata": "acme",
		"appIds":   []string{s.tf.app.ID.String(), s.tf.app.ID.String()},
	})

	s.Equal(http.StatusCreated, response.Code)

	login := response.Map()["login"].(map[string]any)
	s.Equal("client@example.org", login["email"])
	s.Equal("acme", login["metadata"])
	s.Equal([]any{s.tf.app.ID.String()}, login["appIds"])

	s.True(s.tf.opens(s.tf.env.ID, "client@example.org", "client-password"))
	s.False(s.tf.opens(s.MockEnv(s.tf.otherApp).ID, "client@example.org", "client-password"))

	entry := s.tf.lastAudit()
	s.Require().NotNil(entry)
	s.Equal("CREATE:AUTHWALL", entry.Action)
	s.Equal(s.tf.app.ID.String(), entry.Diff.New.AuthWallLoginApps)
	s.Equal("client@example.org", entry.Diff.New.AuthWallCreateLoginEmail)
}

func (s *HandlerTeamLoginCreateSuite) Test_Success_AllApps() {
	response := s.create(s.tf.owner, map[string]any{"allApps": true})

	s.Equal(http.StatusCreated, response.Code)
	s.True(s.tf.opens(s.MockEnv(s.tf.otherApp).ID, "client@example.org", "client-password"))
	s.Equal("all", s.tf.lastAudit().Diff.New.AuthWallLoginApps)
}

// Test_BadRequest_Apps verifies that a login must open at least one app, and
// only apps of its team.
func (s *HandlerTeamLoginCreateSuite) Test_BadRequest_Apps() {
	foreignApp := s.MockApp(s.MockUser())

	for _, appIDs := range [][]string{{}, {foreignApp.ID.String()}, {s.tf.app.ID.String(), foreignApp.ID.String()}} {
		response := s.create(s.tf.owner, map[string]any{"appIds": appIDs})
		s.Equal(http.StatusBadRequest, response.Code)
	}
}

func (s *HandlerTeamLoginCreateSuite) Test_BadRequest_Email() {
	response := s.create(s.tf.owner, map[string]any{"email": "not-an-email", "allApps": true})
	s.Equal(http.StatusBadRequest, response.Code)
}

func (s *HandlerTeamLoginCreateSuite) Test_Conflict() {
	s.tf.login("CLIENT@example.org")

	response := s.create(s.tf.owner, map[string]any{"allApps": true})
	s.Equal(http.StatusConflict, response.Code)
}

func (s *HandlerTeamLoginCreateSuite) Test_Forbidden() {
	for _, usr := range []*factory.MockUser{s.tf.developer, s.MockUser()} {
		response := s.create(usr, map[string]any{"allApps": true})
		s.Equal(http.StatusForbidden, response.Code)
	}
}

func TestHandlerTeamLoginCreateSuite(t *testing.T) {
	suite.Run(t, &HandlerTeamLoginCreateSuite{})
}
