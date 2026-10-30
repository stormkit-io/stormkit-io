package authwallhandlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall/authwallhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type HandlerTeamLoginsSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	tf   *teamFixture
}

func (s *HandlerTeamLoginsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.tf = newTeamFixture(&s.Suite, s.Factory)
	admin.SetMockLicense()
}

func (s *HandlerTeamLoginsSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerTeamLoginsSuite) path() string {
	return fmt.Sprintf("/team/auth-wall/logins?teamId=%s", s.tf.team.ID.String())
}

func (s *HandlerTeamLoginsSuite) Test_Success() {
	bob := s.tf.login("bob@example.org", s.tf.app.ID, s.tf.otherApp.ID)
	alice := s.tf.login("alice@example.org")

	response := s.tf.request(s.tf.owner, shttp.MethodGet, s.path(), nil)

	s.Equal(http.StatusOK, response.Code)

	body := response.Map()
	logins := body["logins"].([]any)

	s.Equal(false, body["hasNextPage"])
	s.Require().Len(logins, 2)

	first := logins[0].(map[string]any)
	second := logins[1].(map[string]any)

	s.Equal(alice.LoginID.String(), first["id"])
	s.Equal("alice@example.org", first["email"])
	s.Equal(false, first["allApps"])
	s.Equal([]any{s.tf.app.ID.String()}, first["appIds"])
	s.Equal(bob.LoginID.String(), second["id"])
	s.Equal([]any{s.tf.app.ID.String(), s.tf.otherApp.ID.String()}, second["appIds"])
	s.NotContains(second, "password")
	s.Equal(false, body["magicLinkAvailable"])
}

// Test_MagicLinkAvailable verifies that the list tells whether sign-in links
// can be sent.
func (s *HandlerTeamLoginsSuite) Test_MagicLinkAvailable() {
	defer authwallhandlers.StubMagicLinkEmail(true, func(authwall.SendMagicLinkParams) {})()

	response := s.tf.request(s.tf.owner, shttp.MethodGet, s.path(), nil)

	s.Equal(http.StatusOK, response.Code)
	s.Equal(true, response.Map()["magicLinkAvailable"])
}

// Test_Success_DeletedApp verifies that a deleted app is no longer listed
// among the apps a login opens.
func (s *HandlerTeamLoginsSuite) Test_Success_DeletedApp() {
	s.tf.login("bob@example.org", s.tf.app.ID, s.tf.otherApp.ID)

	_, err := s.conn.Exec("UPDATE apps SET deleted_at = NOW() WHERE app_id = $1", s.tf.otherApp.ID)
	s.Require().NoError(err)

	response := s.tf.request(s.tf.owner, shttp.MethodGet, s.path(), nil)
	login := response.Map()["logins"].([]any)[0].(map[string]any)

	s.Equal([]any{s.tf.app.ID.String()}, login["appIds"])
}

// Test_Forbidden verifies that only owners and admins of the team see its
// logins.
func (s *HandlerTeamLoginsSuite) Test_Forbidden() {
	for _, usr := range []*factory.MockUser{s.tf.developer, s.MockUser()} {
		response := s.tf.request(usr, shttp.MethodGet, s.path(), nil)
		s.Equal(http.StatusForbidden, response.Code)
	}
}

func TestHandlerTeamLoginsSuite(t *testing.T) {
	suite.Run(t, &HandlerTeamLoginsSuite{})
}
