package apikeyhandlers_test

import (
	"context"
	"fmt"
	"net/http"
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

type HandlerAPIKeyRemoveSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerAPIKeyRemoveSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerAPIKeyRemoveSuite) AfterTest(_, _ string) {
	admin.ResetMockLicense()
	s.conn.CloseTx()
}

func (s *HandlerAPIKeyRemoveSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	key := s.MockAPIKey(app, env, map[string]any{
		"UserID": types.ID(0),
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": usr.DefaultTeamID,
		"Value":  "SK_N32UH0PyJX7K5mMn9RcfpV7BnDK3R00tbuO4T22na2vvrBGv6cs9JlcM3mxfd9",
		"Scope":  apikey.SCOPE_TEAM,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/api-keys?keyId=%s", key.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
}

// Test_Team_DeveloperReadOnly verifies that a team member without write access
// can delete a read-only team key but not a full-access one.
func (s *HandlerAPIKeyRemoveSuite) Test_Team_DeveloperReadOnly() {
	owner := s.MockUser()
	developer := s.MockUser()

	s.Require().NoError(team.NewStore().AddMemberToTeam(context.Background(), &team.Member{
		TeamID: owner.DefaultTeamID,
		UserID: developer.ID,
		Role:   team.ROLE_DEVELOPER,
		Status: true,
	}))

	teamKey := func(readOnly bool) *factory.MockAPIKey {
		return s.MockAPIKey(nil, nil, map[string]any{
			"UserID":   types.ID(0),
			"AppID":    types.ID(0),
			"EnvID":    types.ID(0),
			"TeamID":   owner.DefaultTeamID,
			"Scope":    apikey.SCOPE_TEAM,
			"ReadOnly": readOnly,
		})
	}

	remove := func(key *factory.MockAPIKey) int {
		return shttptest.RequestWithHeaders(
			shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
			shttp.MethodDelete,
			fmt.Sprintf("/api-keys?keyId=%s", key.ID.String()),
			nil,
			map[string]string{
				"Authorization": usertest.Authorization(developer.ID),
			},
		).Code
	}

	s.Equal(http.StatusForbidden, remove(teamKey(false)))
	s.Equal(http.StatusOK, remove(teamKey(true)))
}

// Test_Audit_TeamKey verifies that removing a team key is audit-logged on the
// team with the key's details.
func (s *HandlerAPIKeyRemoveSuite) Test_Audit_TeamKey() {
	admin.SetMockLicense()

	usr := s.MockUser()
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID":   types.ID(0),
		"AppID":    types.ID(0),
		"EnvID":    types.ID(0),
		"TeamID":   usr.DefaultTeamID,
		"Name":     "CI",
		"Scope":    apikey.SCOPE_TEAM,
		"ReadOnly": false,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/api-keys?keyId=%s", key.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		TeamID: usr.DefaultTeamID,
	})

	s.Require().NoError(err)
	s.Require().Len(audits, 1)
	s.Equal("DELETE:APIKEY", audits[0].Action)
	s.Equal("CI", audits[0].Diff.Old.APIKeyName)
	s.Equal(apikey.SCOPE_TEAM, audits[0].Diff.Old.APIKeyScope)
	s.Require().NotNil(audits[0].Diff.Old.APIKeyReadOnly)
	s.False(*audits[0].Diff.Old.APIKeyReadOnly)
}

// Test_Audit_EnvKey verifies that removing an env key is logged on the team
// that owns the environment.
func (s *HandlerAPIKeyRemoveSuite) Test_Audit_EnvKey() {
	admin.SetMockLicense()

	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env, map[string]any{
		"Name":     "Monitor",
		"ReadOnly": true,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/api-keys?keyId=%s", key.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.Require().NoError(err)
	s.Require().Len(audits, 1)
	s.Equal("DELETE:APIKEY", audits[0].Action)
	s.Equal(appl.TeamID, audits[0].TeamID)
	s.Equal(appl.ID, audits[0].AppID)
	s.Equal("Monitor", audits[0].Diff.Old.APIKeyName)
	s.True(*audits[0].Diff.Old.APIKeyReadOnly)
}

// Test_Audit_PersonalKeySkipped verifies that personal keys, which belong to
// no team, are not logged since no audit view could show them.
func (s *HandlerAPIKeyRemoveSuite) Test_Audit_PersonalKeySkipped() {
	admin.SetMockLicense()

	usr := s.MockUser()
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr.ID,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
		"Scope":  apikey.SCOPE_USER,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/api-keys?keyId=%s", key.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	count := 0
	err := s.conn.QueryRow("SELECT COUNT(*) FROM audit_logs WHERE user_id = $1", usr.ID).Scan(&count)

	s.Require().NoError(err)
	s.Equal(0, count)
}

func (s *HandlerAPIKeyRemoveSuite) Test_Success_UserID() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	key := s.MockAPIKey(app, env, map[string]any{
		"UserID": usr.ID,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
		"Value":  "SK_N32UH0PyJX7K5mMn9RcfpV7BnDK3R00tbuO4T22na2vvrBGv6cs9JlcM3mxfd9",
		"Scope":  apikey.SCOPE_USER,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/api-keys?keyId=%s", key.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
}

func (s *HandlerAPIKeyRemoveSuite) Test_Forbidden_ScopeUser() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr2.ID,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
		"Value":  "SK_N32UH0PyJX7K5mMn9RcfpV7BnDK3R00tbuO4T22na2vvrBGv6cs9JlcM3mxfd9",
		"Scope":  apikey.SCOPE_USER,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/api-keys?keyId=%s", key.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr1.ID),
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

func (s *HandlerAPIKeyRemoveSuite) Test_Forbidden_ScopeTeam() {
	usr := s.MockUser()
	usr2 := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	key := s.MockAPIKey(app, env, map[string]any{
		"UserID": types.ID(0),
		"TeamID": usr2.DefaultTeamID,
		"EnvID":  types.ID(0),
		"AppID":  types.ID(0),
		"Value":  "SK_N32UH0PyJX7K5mMn9RcfpV7BnDK3R00tbuO4T22na2vvrBGv6cs9JlcM3mxfd9",
		"Scope":  apikey.SCOPE_TEAM,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/api-keys?keyId=%s", key.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

func (s *HandlerAPIKeyRemoveSuite) Test_Forbidden_ScopeEnv() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	usr2 := s.MockUser()
	app2 := s.MockApp(usr2)
	env2 := s.MockEnv(app2)

	key := s.MockAPIKey(app, env, map[string]any{
		"UserID": types.ID(0),
		"TeamID": types.ID(0),
		"EnvID":  env2.ID,
		"AppID":  types.ID(0),
		"Value":  "SK_N32UH0PyJX7K5mMn9RcfpV7BnDK3R00tbuO4T22na2vvrBGv6cs9JlcM3mxfd9",
		"Scope":  apikey.SCOPE_ENV,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/api-keys?keyId=%s", key.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

func (s *HandlerAPIKeyRemoveSuite) Test_BadRequest() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		"/api-keys",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

type removeKeyParams struct {
	// Remover is the user asking to delete the key.
	Remover *factory.MockUser

	// App is the app the key belongs to.
	App *factory.MockApp

	// Owner is the user a personal key belongs to. Nil for app-level keys.
	Owner *factory.MockUser
}

// removeKey creates a key of p.App and asks p.Remover to delete it.
func (s *HandlerAPIKeyRemoveSuite) removeKey(p removeKeyParams) shttptest.Response {
	overrides := map[string]any{
		"UserID": types.ID(0),
		"TeamID": types.ID(0),
		"EnvID":  types.ID(0),
		"Scope":  apikey.SCOPE_APP,
	}

	if p.Owner != nil {
		overrides["UserID"] = p.Owner.ID
		overrides["Scope"] = apikey.SCOPE_USER
	}

	key := s.MockAPIKey(p.App, nil, overrides)

	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/api-keys?keyId=%s", key.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(p.Remover.ID),
		},
	)
}

func (s *HandlerAPIKeyRemoveSuite) Test_Success_ScopeApp() {
	usr := s.MockUser()

	s.Equal(http.StatusOK, s.removeKey(removeKeyParams{Remover: usr, App: s.MockApp(usr)}).Code)
}

// Test_Forbidden_ScopeApp verifies that an app-level key of another team's app
// cannot be deleted.
func (s *HandlerAPIKeyRemoveSuite) Test_Forbidden_ScopeApp() {
	victimApp := s.MockApp(s.MockUser())

	s.Equal(http.StatusForbidden, s.removeKey(removeKeyParams{Remover: s.MockUser(), App: victimApp}).Code)
}

// Test_Forbidden_PersonalKeyOfTeammate verifies that a personal key which also
// names an app can only be deleted by its owner, not by other team members.
func (s *HandlerAPIKeyRemoveSuite) Test_Forbidden_PersonalKeyOfTeammate() {
	owner := s.MockUser()
	appl := s.MockApp(owner)
	teammate := s.MockUser()

	s.Require().NoError(team.NewStore().AddMemberToTeam(context.Background(), &team.Member{
		TeamID: appl.TeamID,
		UserID: teammate.ID,
		Role:   team.ROLE_OWNER,
		Status: true,
	}))

	s.Equal(http.StatusForbidden, s.removeKey(removeKeyParams{Remover: teammate, App: appl, Owner: owner}).Code)
}

// Test_Forbidden_TeamKeyDeveloper verifies that a team member without write
// access cannot delete the team's keys.
func (s *HandlerAPIKeyRemoveSuite) Test_Forbidden_TeamKeyDeveloper() {
	owner := s.MockUser()
	developer := s.MockUser()

	s.Require().NoError(team.NewStore().AddMemberToTeam(context.Background(), &team.Member{
		TeamID: owner.DefaultTeamID,
		UserID: developer.ID,
		Role:   team.ROLE_DEVELOPER,
		Status: true,
	}))

	key := s.MockAPIKey(s.MockApp(owner), nil, map[string]any{
		"UserID": types.ID(0),
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": owner.DefaultTeamID,
		"Scope":  apikey.SCOPE_TEAM,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/api-keys?keyId=%s", key.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(developer.ID),
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

func TestHandlerAPIKeyRemove(t *testing.T) {
	suite.Run(t, &HandlerAPIKeyRemoveSuite{})
}
