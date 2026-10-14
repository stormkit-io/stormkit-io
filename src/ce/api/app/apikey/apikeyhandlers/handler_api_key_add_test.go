package apikeyhandlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey/apikeyhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type HandlerAPIKeySetSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerAPIKeySetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerAPIKeySetSuite) AfterTest(_, _ string) {
	admin.ResetMockLicense()
	s.conn.CloseTx()
}

func (s *HandlerAPIKeySetSuite) Test_Success_EnvID() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/api-keys",
		map[string]string{
			"appId": "10501", // We want to test that appId is set to environment's appId when envId is provided
			"envId": env.ID.String(),
			"name":  "Default",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	data := apikey.Token{}

	s.NoError(json.Unmarshal([]byte(response.String()), &data))
	s.Equal(http.StatusCreated, response.Code)

	s.Equal(appl.ID, data.AppID)
	s.Equal(env.ID, data.EnvID)
	s.Equal(apikey.SCOPE_ENV, data.Scope)
	s.Equal("Default", data.Name)
	s.True(strings.HasPrefix(data.Value, "SK_"))
}

func (s *HandlerAPIKeySetSuite) Test_Success_ReadOnly() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/api-keys",
		map[string]any{
			"envId":    env.ID.String(),
			"name":     "Monitor",
			"readOnly": true,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	data := apikey.Token{}

	s.NoError(json.Unmarshal([]byte(response.String()), &data))
	s.Equal(http.StatusCreated, response.Code)
	s.True(data.ReadOnly)

	stored, err := apikey.NewStore().APIKey(context.Background(), data.Value)

	s.Require().NoError(err)
	s.Require().NotNil(stored)
	s.True(stored.ReadOnly)
}

// Test_Audit_EnvKey verifies that creating a key is audit-logged on the team
// that owns the environment, with its access level.
func (s *HandlerAPIKeySetSuite) Test_Audit_EnvKey() {
	admin.SetMockLicense()

	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/api-keys",
		map[string]any{
			"envId":    env.ID.String(),
			"name":     "Monitor",
			"readOnly": true,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusCreated, response.Code)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		TeamID: appl.TeamID,
	})

	s.Require().NoError(err)
	s.Require().Len(audits, 1)
	s.Equal("CREATE:APIKEY", audits[0].Action)
	s.Equal(usr.ID, audits[0].UserID)
	s.Equal(appl.ID, audits[0].AppID)
	s.Equal(env.ID, audits[0].EnvID)
	s.Equal("Monitor", audits[0].Diff.New.APIKeyName)
	s.Equal(apikey.SCOPE_ENV, audits[0].Diff.New.APIKeyScope)
	s.Require().NotNil(audits[0].Diff.New.APIKeyReadOnly)
	s.True(*audits[0].Diff.New.APIKeyReadOnly)
}

// Test_Audit_NotEnterprise verifies that nothing is logged without an
// enterprise license.
func (s *HandlerAPIKeySetSuite) Test_Audit_NotEnterprise() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/api-keys",
		map[string]string{
			"teamId": usr.DefaultTeamID.String(),
			"name":   "Default",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusCreated, response.Code)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		TeamID: usr.DefaultTeamID,
	})

	s.Require().NoError(err)
	s.Empty(audits)
}

func (s *HandlerAPIKeySetSuite) Test_Success_UserID() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/api-keys",
		map[string]string{
			"userId": usr.ID.String(),
			"name":   "MCP",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	data := apikey.Token{}

	s.NoError(json.Unmarshal([]byte(response.String()), &data))
	s.Equal(http.StatusCreated, response.Code)

	s.Equal(types.ID(0), data.AppID)
	s.Equal(types.ID(0), data.EnvID)
	s.Equal(usr.ID, data.UserID)
	s.Equal(apikey.SCOPE_USER, data.Scope)
	s.Equal("MCP", data.Name)
	s.True(strings.HasPrefix(data.Value, "SK_"))
}

func (s *HandlerAPIKeySetSuite) Test_Success_TeamID() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/api-keys",
		map[string]string{
			"teamId": usr.DefaultTeamID.String(),
			"name":   "Default",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusCreated, response.Code)
}

// Test_Team_DeveloperForbidden verifies that a team member without write
// access cannot create a team key, which would bypass role checks.
func (s *HandlerAPIKeySetSuite) Test_Team_DeveloperForbidden() {
	owner := s.MockUser()
	developer := s.MockUser()

	s.Require().NoError(team.NewStore().AddMemberToTeam(context.Background(), &team.Member{
		TeamID: owner.DefaultTeamID,
		UserID: developer.ID,
		Role:   team.ROLE_DEVELOPER,
		Status: true,
	}))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/api-keys",
		map[string]string{
			"teamId": owner.DefaultTeamID.String(),
			"name":   "Default",
		},
		map[string]string{
			"Authorization": usertest.Authorization(developer.ID),
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_Team_DeveloperReadOnly verifies that a team member without write access
// can still create a read-only team key.
func (s *HandlerAPIKeySetSuite) Test_Team_DeveloperReadOnly() {
	owner := s.MockUser()
	developer := s.MockUser()

	s.Require().NoError(team.NewStore().AddMemberToTeam(context.Background(), &team.Member{
		TeamID: owner.DefaultTeamID,
		UserID: developer.ID,
		Role:   team.ROLE_DEVELOPER,
		Status: true,
	}))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/api-keys",
		map[string]any{
			"teamId":   owner.DefaultTeamID.String(),
			"name":     "Dashboard",
			"readOnly": true,
		},
		map[string]string{
			"Authorization": usertest.Authorization(developer.ID),
		},
	)

	data := apikey.Token{}

	s.NoError(json.Unmarshal([]byte(response.String()), &data))
	s.Equal(http.StatusCreated, response.Code)
	s.Equal(owner.DefaultTeamID, data.TeamID)
	s.True(data.ReadOnly)
}

func (s *HandlerAPIKeySetSuite) Test_InvalidScope() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/api-keys",
		map[string]string{
			"appId": appl.ID.String(),
			"envId": env.ID.String(),
			"scope": "invalid_scope",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	data := map[string]string{}

	s.NoError(json.Unmarshal([]byte(response.String()), &data))
	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal(data["error"], "Invalid scope. Allowes scopes are: team, env, user, app, admin")
}

func (s *HandlerAPIKeySetSuite) Test_InvalidName() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/api-keys",
		map[string]string{
			"appId": appl.ID.String(),
			"envId": env.ID.String(),
			"scope": "env",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	data := map[string]string{}

	s.NoError(json.Unmarshal([]byte(response.String()), &data))
	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal(data["error"], "Key name is a required field.")
}

func (s *HandlerAPIKeySetSuite) Test_Invalid_EnvID() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/api-keys",
		map[string]string{
			"appId": "10501", // We want to test that appId is set to environment's appId when envId is provided
			"envId": "40101", // Non existing envID
			"name":  "Default",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	data := map[string]string{}

	s.NoError(json.Unmarshal([]byte(response.String()), &data))
	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal(data["error"], "Environment not found.")
}

func (s *HandlerAPIKeySetSuite) Test_Env_NoAccess() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	appl := s.MockApp(usr1)
	env := s.MockEnv(appl)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/api-keys",
		map[string]string{
			"envId": env.ID.String(),
			"name":  "Default",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr2.ID),
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

func (s *HandlerAPIKeySetSuite) Test_Invalid_AppID() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/api-keys",
		map[string]string{
			"appId": "10501", // Not found
			"name":  "Default",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	data := map[string]string{}

	s.NoError(json.Unmarshal([]byte(response.String()), &data))
	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal(data["error"], "Application not found.")
}

func (s *HandlerAPIKeySetSuite) Test_App_NoAccess() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	appl := s.MockApp(usr1)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/api-keys",
		map[string]string{
			"appId": appl.ID.String(),
			"name":  "Default",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr2.ID),
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

func (s *HandlerAPIKeySetSuite) Test_Team_NoAccess() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/api-keys",
		map[string]string{
			"teamId": usr1.DefaultTeamID.String(),
			"name":   "Default",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr2.ID),
		},
	)

	data := map[string]string{}

	s.NoError(json.Unmarshal([]byte(response.String()), &data))
	s.Equal(http.StatusBadRequest, response.Code)
	s.Equal(data["error"], "Team not found.")
}

func (s *HandlerAPIKeySetSuite) Test_UserID_NoAccess() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/api-keys",
		map[string]string{
			"userId": usr1.ID.String(),
			"name":   "Default",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr2.ID),
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

func TestHandlerAPIKeySet(t *testing.T) {
	suite.Run(t, &HandlerAPIKeySetSuite{})
}
