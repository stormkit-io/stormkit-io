package publicapiv1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerEnvPullSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerEnvPullSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerEnvPullSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerEnvPullSuite) TestSuccess() {
	vars := map[string]string{
		"NODE_ENV": "production",
		"TEST_1":   "VALUE_1",
		"TEST_2":   "VALUE_1=VALUE_1",
	}

	env := s.MockEnv(nil, map[string]any{
		"Data": &buildconf.BuildConf{
			Vars: vars,
		},
	})

	key := s.MockAPIKey(nil, env)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/env/pull",
		nil,
		map[string]string{
			"Authorization": key.Value,
		},
	)

	jsonVal, err := json.Marshal(vars)
	s.NoError(err)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(response.String(), string(jsonVal))
}

// Revealing the values through an API key is audited too, attributed to the
// key's token name rather than a user.
func (s *HandlerEnvPullSuite) TestAPIKeyRevealIsAudited() {
	admin.SetMockLicense()
	defer admin.ResetMockLicense()

	env := s.MockEnv(nil, map[string]any{
		"Data": &buildconf.BuildConf{Vars: map[string]string{"SECRET": "shh"}},
	})

	key := s.MockAPIKey(nil, env)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		"/v1/env/pull",
		nil,
		map[string]string{"Authorization": key.Value},
	)

	s.Equal(http.StatusOK, response.Code)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})
	s.NoError(err)
	s.Len(audits, 1)
	s.Equal("REVEAL:ENV", audits[0].Action)
	s.NotEmpty(audits[0].TokenName)
}

// A dashboard (JWT) caller who is a team admin/owner can reveal the values,
// and the reveal is audited.
func (s *HandlerEnvPullSuite) TestJWTOwnerReveals() {
	admin.SetMockLicense()
	defer admin.ResetMockLicense()

	vars := map[string]string{"SECRET": "shh"}

	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{
		"Data": &buildconf.BuildConf{Vars: vars},
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/env/pull?envId=%s", env.ID.String()),
		nil,
		map[string]string{"Authorization": usertest.Authorization(usr.ID)},
	)

	jsonVal, _ := json.Marshal(vars)
	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(string(jsonVal), response.String())

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})
	s.NoError(err)
	s.Len(audits, 1)
	s.Equal("REVEAL:ENV", audits[0].Action)
}

// A developer (JWT) who is a member of the environment's team can reveal the
// values too — reveal requires team membership (enforced by the scope
// middleware), not an admin/owner role, so it stays consistent with API keys
// and lets developers edit env vars (which requires revealing first).
func (s *HandlerEnvPullSuite) TestJWTDeveloperReveals() {
	vars := map[string]string{"SECRET": "shh"}

	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{
		"Data": &buildconf.BuildConf{Vars: vars},
	})

	dev := s.MockUser()
	s.NoError(team.NewStore().AddMemberToTeam(context.Background(), &team.Member{
		TeamID: usr.DefaultTeamID,
		UserID: dev.ID,
		Role:   team.ROLE_DEVELOPER,
		Status: true,
	}))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/env/pull?envId=%s", env.ID.String()),
		nil,
		map[string]string{"Authorization": usertest.Authorization(dev.ID)},
	)

	jsonVal, _ := json.Marshal(vars)
	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(string(jsonVal), response.String())
}

func TestHandlerEnvPull(t *testing.T) {
	suite.Run(t, &HandlerEnvPullSuite{})
}
