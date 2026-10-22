package authwallhandlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stretchr/testify/suite"
)

type HandlerTeamLoginsDeleteSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	tf   *teamFixture
}

func (s *HandlerTeamLoginsDeleteSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.tf = newTeamFixture(&s.Suite, s.Factory)
	admin.SetMockLicense()
}

func (s *HandlerTeamLoginsDeleteSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

// Test_Success verifies that the team's logins are deleted, while another
// team's login passed along is left alone.
func (s *HandlerTeamLoginsDeleteSuite) Test_Success() {
	login := s.tf.login("client@example.org")
	other := newTeamFixture(&s.Suite, s.Factory)
	foreign := other.login("foreign@example.org")

	path := fmt.Sprintf("/team/auth-wall/logins?teamId=%s&id=%s,%s", s.tf.team.ID, login.LoginID, foreign.LoginID)
	response := s.tf.request(s.tf.owner, shttp.MethodDelete, path, nil)

	s.Equal(http.StatusOK, response.Code)
	s.False(s.tf.opens(s.tf.env.ID, "client@example.org", "123pass-word"))
	s.True(other.opens(other.env.ID, "foreign@example.org", "123pass-word"))

	entry := s.tf.lastAudit()
	s.Require().NotNil(entry)
	s.Equal("DELETE:AUTHWALL", entry.Action)
	s.Equal(login.LoginID.String(), entry.Diff.Old.AuthWallDeleteLoginIDs, "only the deleted login is audited")
}

func (s *HandlerTeamLoginsDeleteSuite) Test_Forbidden() {
	login := s.tf.login("client@example.org")
	path := fmt.Sprintf("/team/auth-wall/logins?teamId=%s&id=%s", s.tf.team.ID, login.LoginID)

	response := s.tf.request(s.tf.developer, shttp.MethodDelete, path, nil)

	s.Equal(http.StatusForbidden, response.Code)
	s.True(s.tf.opens(s.tf.env.ID, "client@example.org", "123pass-word"))
}

func TestHandlerTeamLoginsDeleteSuite(t *testing.T) {
	suite.Run(t, &HandlerTeamLoginsDeleteSuite{})
}
