package publicapiv1_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// userOnlyTools need a user-level key.
var userOnlyTools = []string{"list_teams", "create_team"}

// teamTools need at least a team key.
var teamTools = []string{"list_apps", "create_app"}

// appTools need at least an app key.
var appTools = []string{"list_environments", "create_environment"}

type HandlerMCPKeyScopeSuite struct {
	suite.Suite
	*factory.Factory

	conn         databasetest.TestDB
	mockDeployer *mocks.Deployer
}

func (s *HandlerMCPKeyScopeSuite) BeforeTest(suiteName, _ string) {
	config.SetIsStormkitCloud(false)

	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockDeployer = &mocks.Deployer{}
	s.mockDeployer.On("Deploy", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deployservice.MockDeployer = s.mockDeployer
}

func (s *HandlerMCPKeyScopeSuite) AfterTest(_, _ string) {
	deployservice.MockDeployer = nil
	buildconf.SendMailFunc = buildconf.SendMailWithDeadline
	s.conn.CloseTx()
}

// key creates an API key bound to the given IDs.
func (s *HandlerMCPKeyScopeSuite) key(scope string, ids map[string]types.ID) *factory.MockAPIKey {
	fields := map[string]any{
		"Scope":  scope,
		"UserID": types.ID(0),
		"TeamID": types.ID(0),
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
	}

	for k, v := range ids {
		fields[k] = v
	}

	return s.MockAPIKey(nil, nil, fields)
}

// post sends a POST to /v1/mcp and returns the JSON-RPC result.
func (s *HandlerMCPKeyScopeSuite) post(keyValue string, body any) map[string]any {
	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/mcp",
		body,
		map[string]string{"Authorization": keyValue},
	)

	s.Require().Equal(http.StatusOK, resp.Code)

	envelope := resp.Map()
	s.Require().Nil(envelope["error"])

	return envelope["result"].(map[string]any)
}

// toolNames lists the names of the tools visible to the key.
func (s *HandlerMCPKeyScopeSuite) toolNames(keyValue string) []string {
	result := s.post(keyValue, mcpBody(1, "tools/list", map[string]any{}))
	names := []string{}

	for _, t := range result["tools"].([]any) {
		names = append(names, t.(map[string]any)["name"].(string))
	}

	return names
}

// without returns names minus the excluded ones.
func (s *HandlerMCPKeyScopeSuite) without(names []string, excluded ...[]string) []string {
	skip := map[string]bool{}

	for _, list := range excluded {
		for _, name := range list {
			skip[name] = true
		}
	}

	kept := []string{}

	for _, name := range names {
		if !skip[name] {
			kept = append(kept, name)
		}
	}

	return kept
}

// call invokes a tool and returns its result and text content.
func (s *HandlerMCPKeyScopeSuite) call(keyValue, tool string, args map[string]any) (map[string]any, string) {
	result := s.post(keyValue, mcpToolCall(1, tool, args))
	text := result["content"].([]any)[0].(map[string]any)["text"].(string)

	return result, text
}

func (s *HandlerMCPKeyScopeSuite) Test_ToolsList_TeamKey() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	all := s.toolNames(s.key(apikey.SCOPE_USER, map[string]types.ID{"UserID": usr.ID}).Value)
	teamKey := s.key(apikey.SCOPE_TEAM, map[string]types.ID{"TeamID": appl.TeamID})

	s.ElementsMatch(s.without(all, userOnlyTools), s.toolNames(teamKey.Value))
}

func (s *HandlerMCPKeyScopeSuite) Test_ToolsList_AppKey() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	all := s.toolNames(s.key(apikey.SCOPE_USER, map[string]types.ID{"UserID": usr.ID}).Value)
	appKey := s.key(apikey.SCOPE_APP, map[string]types.ID{"AppID": appl.ID})

	s.ElementsMatch(s.without(all, userOnlyTools, teamTools), s.toolNames(appKey.Value))
}

func (s *HandlerMCPKeyScopeSuite) Test_ToolsList_EnvKey() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	all := s.toolNames(s.key(apikey.SCOPE_USER, map[string]types.ID{"UserID": usr.ID}).Value)
	envKey := s.MockAPIKey(appl, env)

	s.ElementsMatch(s.without(all, userOnlyTools, teamTools, appTools), s.toolNames(envKey.Value))
}

// Test_ToolsList_ReadOnlyTeamKey verifies that the read-only and scope filters
// both apply.
func (s *HandlerMCPKeyScopeSuite) Test_ToolsList_ReadOnlyTeamKey() {
	appl := s.MockApp(s.MockUser())
	key := s.MockAPIKey(nil, nil, map[string]any{
		"Scope":    apikey.SCOPE_TEAM,
		"TeamID":   appl.TeamID,
		"AppID":    types.ID(0),
		"EnvID":    types.ID(0),
		"ReadOnly": true,
	})

	s.ElementsMatch(s.without(readOnlyTools, userOnlyTools), s.toolNames(key.Value))
}

// Test_ToolCall_TeamKeyCannotListTeams verifies that a hidden tool called
// anyway fails as a tool error that names the key's scope.
func (s *HandlerMCPKeyScopeSuite) Test_ToolCall_TeamKeyCannotListTeams() {
	appl := s.MockApp(s.MockUser())
	teamKey := s.key(apikey.SCOPE_TEAM, map[string]types.ID{"TeamID": appl.TeamID})

	result, text := s.call(teamKey.Value, "list_teams", map[string]any{})

	s.Equal(true, result["isError"])
	s.Contains(text, "api_key_scope")
	s.Contains(text, "single team")
}

func (s *HandlerMCPKeyScopeSuite) Test_ToolCall_TeamKeyListsOwnAppsOnly() {
	appl := s.MockApp(s.MockUser())
	other := s.MockApp(s.MockUser())
	teamKey := s.key(apikey.SCOPE_TEAM, map[string]types.ID{"TeamID": appl.TeamID})

	result, text := s.call(teamKey.Value, "list_apps", map[string]any{"teamId": appl.TeamID.String()})
	s.Nil(result["isError"], text)
	s.Contains(text, appl.DisplayName)

	result, _ = s.call(teamKey.Value, "list_apps", map[string]any{"teamId": other.TeamID.String()})
	s.Equal(true, result["isError"])
}

// Test_ToolCall_TeamKeyDeploysWithinTeam verifies that a team key can deploy
// its own team's environments and nothing outside the team.
func (s *HandlerMCPKeyScopeSuite) Test_ToolCall_TeamKeyDeploysWithinTeam() {
	appl := s.MockApp(s.MockUser())
	env := s.MockEnv(appl)
	otherApp := s.MockApp(s.MockUser())
	otherEnv := s.MockEnv(otherApp)
	teamKey := s.key(apikey.SCOPE_TEAM, map[string]types.ID{"TeamID": appl.TeamID})

	result, _ := s.call(teamKey.Value, "deploy", map[string]any{"envId": otherEnv.ID.String()})
	s.Equal(true, result["isError"])
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy", mock.Anything, mock.Anything, mock.Anything)

	result, text := s.call(teamKey.Value, "deploy", map[string]any{"envId": env.ID.String()})
	s.Nil(result["isError"], text)
	s.mockDeployer.AssertNumberOfCalls(s.T(), "Deploy", 1)
}

func (s *HandlerMCPKeyScopeSuite) Test_ToolCall_AppKeyListsOwnEnvironmentsOnly() {
	appl := s.MockApp(s.MockUser())
	other := s.MockApp(s.MockUser())
	appKey := s.key(apikey.SCOPE_APP, map[string]types.ID{"AppID": appl.ID})

	result, text := s.call(appKey.Value, "list_environments", map[string]any{"appId": appl.ID.String()})
	s.Nil(result["isError"], text)

	result, _ = s.call(appKey.Value, "list_environments", map[string]any{"appId": other.ID.String()})
	s.Equal(true, result["isError"])
}

// Test_ToolCall_EnvKeyStaysInEnv verifies that an environment key cannot read
// a sibling environment of the same app.
func (s *HandlerMCPKeyScopeSuite) Test_ToolCall_EnvKeyStaysInEnv() {
	appl := s.MockApp(s.MockUser())
	env := s.MockEnv(appl)
	sibling := s.MockEnv(appl, map[string]any{"Name": "staging"})
	envKey := s.MockAPIKey(appl, env)

	result, text := s.call(envKey.Value, "list_deployments", map[string]any{"envId": env.ID.String()})
	s.Nil(result["isError"], text)

	result, _ = s.call(envKey.Value, "list_deployments", map[string]any{"envId": sibling.ID.String()})
	s.Equal(true, result["isError"])
}

// Test_ToolCall_EnvKeyDefaultsEnvID verifies that an environment key can call
// a tool without passing its own envId.
func (s *HandlerMCPKeyScopeSuite) Test_ToolCall_EnvKeyDefaultsEnvID() {
	appl := s.MockApp(s.MockUser())
	env := s.MockEnv(appl)
	envKey := s.MockAPIKey(appl, env)

	result, text := s.call(envKey.Value, "deploy", map[string]any{})
	s.Nil(result["isError"], text)
	s.mockDeployer.AssertNumberOfCalls(s.T(), "Deploy", 1)
}

// Test_ToolCall_ScopedKeysDefaultIDs verifies that team and app keys can list
// their own apps and environments without passing the ID.
func (s *HandlerMCPKeyScopeSuite) Test_ToolCall_ScopedKeysDefaultIDs() {
	appl := s.MockApp(s.MockUser())
	teamKey := s.key(apikey.SCOPE_TEAM, map[string]types.ID{"TeamID": appl.TeamID})
	appKey := s.key(apikey.SCOPE_APP, map[string]types.ID{"AppID": appl.ID})

	result, text := s.call(teamKey.Value, "list_apps", map[string]any{})
	s.Nil(result["isError"], text)
	s.Contains(text, appl.DisplayName)

	result, text = s.call(appKey.Value, "list_environments", map[string]any{})
	s.Nil(result["isError"], text)
}

// Test_ToolsList_ScopedKeyOptionalID verifies that the ID a scoped key fills in
// itself is not listed as required.
func (s *HandlerMCPKeyScopeSuite) Test_ToolsList_ScopedKeyOptionalID() {
	appl := s.MockApp(s.MockUser())
	env := s.MockEnv(appl)
	envKey := s.MockAPIKey(appl, env)
	result := s.post(envKey.Value, mcpBody(1, "tools/list", map[string]any{}))

	for _, t := range result["tools"].([]any) {
		tool := t.(map[string]any)

		if tool["name"] != "get_deployment" {
			continue
		}

		schema := tool["inputSchema"].(map[string]any)
		s.ElementsMatch([]any{"deploymentId"}, schema["required"])
		s.Contains(schema["properties"].(map[string]any)["envId"].(map[string]any)["description"], "Defaults to the one this API key is bound to.")

		return
	}

	s.Fail("get_deployment not listed")
}

// Test_ToolCall_ForeignEnvLooksMissing verifies that an environment the key
// cannot reach is answered exactly like one that does not exist.
func (s *HandlerMCPKeyScopeSuite) Test_ToolCall_ForeignEnvLooksMissing() {
	appl := s.MockApp(s.MockUser())
	otherEnv := s.MockEnv(s.MockApp(s.MockUser()))
	teamKey := s.key(apikey.SCOPE_TEAM, map[string]types.ID{"TeamID": appl.TeamID})

	_, foreign := s.call(teamKey.Value, "list_deployments", map[string]any{"envId": otherEnv.ID.String()})
	_, missing := s.call(teamKey.Value, "list_deployments", map[string]any{"envId": "99999999"})

	s.Equal(missing, foreign)
	s.Contains(foreign, "does not exist")
}

func TestHandlerMCPKeyScopeSuite(t *testing.T) {
	suite.Run(t, &HandlerMCPKeyScopeSuite{})
}
