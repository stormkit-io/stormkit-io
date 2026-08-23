package publicapiv1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/smtp"
	"strings"
	"testing"
	"time"

	null "gopkg.in/guregu/null.v3"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
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

// mcpBody builds a JSON-RPC 2.0 request body.
func mcpBody(id any, method string, params any) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  method,
		"params":  params,
	}
}

// mcpToolCall builds a tools/call JSON-RPC body.
func mcpToolCall(id any, tool string, arguments map[string]any) map[string]any {
	return mcpBody(id, "tools/call", map[string]any{
		"name":      tool,
		"arguments": arguments,
	})
}

type HandlerMCPSuite struct {
	suite.Suite
	*factory.Factory

	conn         databasetest.TestDB
	mockDeployer *mocks.Deployer
}

func (s *HandlerMCPSuite) BeforeTest(suiteName, _ string) {
	// Reset edition to development — earlier tests in the same package (e.g.
	// TestServices/Test_Services_StormkitCloud) leak the cloud edition through
	// the shared package-level `edition` variable.
	config.SetIsStormkitCloud(false)

	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockDeployer = &mocks.Deployer{}
	s.mockDeployer.On("Deploy", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deployservice.MockDeployer = s.mockDeployer
}

func (s *HandlerMCPSuite) AfterTest(_, _ string) {
	deployservice.MockDeployer = nil
	buildconf.SendMailFunc = buildconf.SendMailWithDeadline
	s.conn.CloseTx()
}

func (s *HandlerMCPSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

// userKey creates a SCOPE_USER API key owned by usr.
func (s *HandlerMCPSuite) userKey(usr *factory.MockUser) *factory.MockAPIKey {
	return s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr.ID,
		"Scope":  apikey.SCOPE_USER,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
	})
}

// post sends a POST to /v1/mcp with the given body and Authorization header.
func (s *HandlerMCPSuite) post(keyValue string, body any) shttptest.Response {
	return shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/mcp",
		body,
		map[string]string{"Authorization": keyValue},
	)
}

// rpcOK asserts the response is HTTP 200 and returns the decoded JSON-RPC result.
func (s *HandlerMCPSuite) rpcOK(resp shttptest.Response) map[string]any {
	var envelope map[string]any
	s.Equal(http.StatusOK, resp.Code)
	s.NoError(json.Unmarshal([]byte(resp.String()), &envelope))
	s.Equal("2.0", envelope["jsonrpc"])
	s.Nil(envelope["error"], "expected no JSON-RPC error, got: %v", envelope["error"])
	return envelope
}

// rpcError asserts the response is HTTP 200 (JSON-RPC transport) with an error
// field and returns the error object.
func (s *HandlerMCPSuite) rpcError(resp shttptest.Response) map[string]any {
	var envelope map[string]any
	s.Equal(http.StatusOK, resp.Code)
	s.NoError(json.Unmarshal([]byte(resp.String()), &envelope))
	s.NotNil(envelope["error"], "expected a JSON-RPC error")
	return envelope["error"].(map[string]any)
}

// toolContent extracts the first "content" text from a tools/call result.
func (s *HandlerMCPSuite) toolContent(envelope map[string]any) map[string]any {
	result := envelope["result"].(map[string]any)
	content := result["content"].([]any)
	s.NotEmpty(content)
	text := content[0].(map[string]any)["text"].(string)
	var data map[string]any
	s.NoError(json.Unmarshal([]byte(text), &data))
	return data
}

func (s *HandlerMCPSuite) Test_Forbidden_NoKey() {
	resp := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/mcp",
		mcpBody(1, "tools/list", map[string]any{}),
		map[string]string{},
	)
	s.Equal(http.StatusForbidden, resp.Code)
}

func (s *HandlerMCPSuite) Test_Forbidden_LowScopeKey() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)

	// SCOPE_ENV key is below the SCOPE_USER minimum required for /v1/mcp.
	envKey := s.MockAPIKey(appl, env)

	resp := s.post(envKey.Value, mcpBody(1, "tools/list", map[string]any{}))
	s.Equal(http.StatusForbidden, resp.Code)
}

func (s *HandlerMCPSuite) Test_ParseError_InvalidJSON() {
	usr := s.MockUser()
	key := s.userKey(usr)

	// shttptest always JSON-encodes the body, so we bypass it here to send
	// genuinely invalid JSON and trigger the -32700 parse error path.
	r := httptest.NewRequest(shttp.MethodPost, "/v1/mcp", strings.NewReader("{bad json"))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", key.Value)
	w := httptest.NewRecorder()
	s.handler().ServeHTTP(w, r)

	var envelope map[string]any
	s.Equal(http.StatusOK, w.Code)
	s.NoError(json.Unmarshal(w.Body.Bytes(), &envelope))
	s.NotNil(envelope["error"])

	errObj := envelope["error"].(map[string]any)
	s.EqualValues(-32700, errObj["code"])
}

// Test_ParseError_TypeMismatch verifies that a well-formed JSON body whose field
// types don't match the expected schema returns -32600 (invalid request), not
// -32700 (parse error).
func (s *HandlerMCPSuite) Test_ParseError_TypeMismatch() {
	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  []string{"not", "a", "string"},
	})

	errObj := s.rpcError(resp)
	s.EqualValues(-32600, errObj["code"])
}

func (s *HandlerMCPSuite) Test_InvalidRequest_WrongVersion() {
	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, map[string]any{
		"jsonrpc": "1.0",
		"id":      1,
		"method":  "tools/list",
		"params":  map[string]any{},
	})

	errObj := s.rpcError(resp)
	s.EqualValues(-32600, errObj["code"])
}

func (s *HandlerMCPSuite) Test_MethodNotFound() {
	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpBody(1, "unknown/method", map[string]any{}))
	errObj := s.rpcError(resp)
	s.EqualValues(-32601, errObj["code"])
}

func (s *HandlerMCPSuite) Test_Initialize() {
	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpBody(1, "initialize", map[string]any{}))
	env := s.rpcOK(resp)

	result := env["result"].(map[string]any)
	s.Equal("2025-11-25", result["protocolVersion"])

	serverInfo := result["serverInfo"].(map[string]any)
	s.Equal("stormkit", serverInfo["name"])
}

// Test_NotificationsInitialized verifies that notifications/initialized returns
// HTTP 202 with no body, per the JSON-RPC 2.0 rule that servers MUST NOT reply
// to notifications.
func (s *HandlerMCPSuite) Test_NotificationsInitialized() {
	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpBody(nil, "notifications/initialized", map[string]any{}))
	s.Equal(http.StatusAccepted, resp.Code)
	s.Empty(resp.String())
}

// Test_SSEStream verifies that GET /v1/mcp returns a text/event-stream response,
// as required by the MCP Streamable HTTP transport (2025-11-25).
func (s *HandlerMCPSuite) Test_SSEStream() {
	usr := s.MockUser()
	key := s.userKey(usr)

	// Use a short timeout so the handler exits after auth succeeds.
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	r := httptest.NewRequest(http.MethodGet, "/v1/mcp", nil)
	r = r.WithContext(ctx)
	r.Header.Set("Authorization", key.Value)

	w := httptest.NewRecorder()
	s.handler().ServeHTTP(w, r)

	s.Equal(http.StatusOK, w.Code)
	s.Equal("text/event-stream", w.Header().Get("Content-Type"))
}

// Test_SSEStream_Forbidden verifies that the SSE endpoint requires auth.
func (s *HandlerMCPSuite) Test_SSEStream_Forbidden() {
	r := httptest.NewRequest(http.MethodGet, "/v1/mcp", nil)
	w := httptest.NewRecorder()
	s.handler().ServeHTTP(w, r)
	s.Equal(http.StatusForbidden, w.Code)
}

func (s *HandlerMCPSuite) Test_ToolsList_ReturnsExpectedTools() {
	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpBody(1, "tools/list", map[string]any{}))
	env := s.rpcOK(resp)

	result := env["result"].(map[string]any)
	tools := result["tools"].([]any)

	names := make([]string, 0, len(tools))

	for _, t := range tools {
		names = append(names, t.(map[string]any)["name"].(string))
	}

	s.ElementsMatch([]string{
		"create_app",
		"deploy",
		"get_deployment",
		"get_runtime_logs",
		"get_access_logs",
		"publish_deployment",
		"delete_deployment",
		"restart_deployment",
		"prioritize_deployment",
		"list_deployments",
		"stop_deployment",
		"list_apps",
		"list_environments",
		"create_environment",
		"update_environment",
		"list_domains",
		"create_domain",
		"delete_domain",
		"list_triggers",
		"create_trigger",
		"update_trigger",
		"delete_trigger",
		"invoke_trigger",
		"get_trigger_logs",
		"list_teams",
		"create_team",
		"get_mailer_config",
		"configure_mailer",
		"list_emails",
		"send_test_email",
		"get_auth_config",
		"configure_auth_provider",
		"configure_auth",
		"enable_database_integration",
		"configure_database_integration",
	}, names)
}

func (s *HandlerMCPSuite) Test_ToolsList_HidesDatabaseToolsOnCloud() {
	config.SetIsStormkitCloud(true)
	defer config.SetIsStormkitCloud(false)

	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpBody(1, "tools/list", map[string]any{}))
	env := s.rpcOK(resp)

	result := env["result"].(map[string]any)
	tools := result["tools"].([]any)

	names := make([]string, 0, len(tools))

	for _, t := range tools {
		names = append(names, t.(map[string]any)["name"].(string))
	}

	s.NotContains(names, "enable_database_integration")
	s.NotContains(names, "configure_database_integration")
	// The auth-config tools are gated the same way (self-hosted only).
	s.NotContains(names, "get_auth_config")
	s.NotContains(names, "configure_auth")
	s.NotContains(names, "configure_auth_provider")
	// The mailer tools are not gated -- mailer configuration ships in every
	// edition, same as the /v1/mail endpoint.
	s.Contains(names, "get_mailer_config")
	s.Contains(names, "configure_mailer")
	s.Contains(names, "list_emails")
	s.Contains(names, "send_test_email")
}

func (s *HandlerMCPSuite) Test_EnableDatabaseIntegration_RejectedOnCloud() {
	config.SetIsStormkitCloud(true)
	defer config.SetIsStormkitCloud(false)

	usr := s.MockUser()
	key := s.userKey(usr)
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)

	resp := s.post(key.Value, mcpToolCall(1, "enable_database_integration", map[string]any{
		"envId": fmt.Sprint(env.ID),
	}))

	errObj := s.rpcError(resp)
	s.EqualValues(-32601, errObj["code"])
}

func (s *HandlerMCPSuite) Test_EnableDatabaseIntegration_MissingEnvID() {
	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "enable_database_integration", map[string]any{}))
	envelope := s.rpcOK(resp)

	result := envelope["result"].(map[string]any)
	s.True(result["isError"].(bool), "expected an isError tool response")
}

func (s *HandlerMCPSuite) Test_ConfigureDatabaseIntegration_PartialUpdate() {
	usr := s.MockUser()
	key := s.userKey(usr)
	appl := s.MockApp(usr)
	env := s.MockEnv(appl, map[string]any{
		"SchemaConf": &buildconf.SchemaConf{
			SchemaName:        buildconf.SchemaName(appl.ID, 0),
			MigrationsEnabled: true,
			MigrationsFolder:  "db/migrations",
			InjectEnvVars:     true,
		},
	})

	// Send only injectEnvVars: false — the other two fields must be preserved.
	resp := s.post(key.Value, mcpToolCall(1, "configure_database_integration", map[string]any{
		"envId":         fmt.Sprint(env.ID),
		"injectEnvVars": false,
	}))

	s.rpcOK(resp)

	stored, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)
	s.NoError(err)
	s.NotNil(stored.SchemaConf)
	s.False(stored.SchemaConf.InjectEnvVars, "injectEnvVars should have flipped to false")
	s.True(stored.SchemaConf.MigrationsEnabled, "migrationsEnabled must be preserved when omitted")
	s.Equal("db/migrations", stored.SchemaConf.MigrationsFolder, "migrationsFolder must be preserved when omitted")
}

func (s *HandlerMCPSuite) Test_UnknownTool() {
	usr := s.MockUser()
	key := s.userKey(usr)
	resp := s.post(key.Value, mcpToolCall(1, "no_such_tool", map[string]any{}))
	errObj := s.rpcError(resp)
	s.EqualValues(-32601, errObj["code"])
}

func (s *HandlerMCPSuite) Test_ListApps_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "list_apps", map[string]any{
		"teamId": appl.TeamID.String(),
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	apps, ok := data["apps"].([]any)
	s.True(ok)
	s.NotEmpty(apps)
}

func (s *HandlerMCPSuite) Test_ListApps_Forbidden_NotMember() {
	usr1 := s.MockUser()
	appl := s.MockApp(usr1)

	usr2 := s.MockUser()
	key := s.userKey(usr2)

	resp := s.post(key.Value, mcpToolCall(1, "list_apps", map[string]any{
		"teamId": appl.TeamID.String(),
	}))

	env := s.rpcOK(resp) // transport is still 200
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_ListEnvironments_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "list_environments", map[string]any{
		"appId": appl.ID.String(),
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	envs, ok := data["environments"].([]any)
	s.True(ok)
	s.NotEmpty(envs)
}

func (s *HandlerMCPSuite) Test_ListEnvironments_Forbidden_NotMember() {
	usr1 := s.MockUser()
	appl := s.MockApp(usr1)

	usr2 := s.MockUser()
	key := s.userKey(usr2)

	resp := s.post(key.Value, mcpToolCall(1, "list_environments", map[string]any{
		"appId": appl.ID.String(),
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_ListEnvironments_MasksEnvVarValues() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	key := s.userKey(usr)

	// Seed an environment with a user secret and an SK_ framework variable.
	createResp := s.post(key.Value, mcpToolCall(1, "create_environment", map[string]any{
		"appId":  appl.ID.String(),
		"name":   "staging",
		"branch": "main",
		"envVars": map[string]any{
			"SECRET_TOKEN": "super-secret-value",
			"SK_SECRET":    "also-secret",
		},
	}))
	s.rpcOK(createResp)

	resp := s.post(key.Value, mcpToolCall(2, "list_environments", map[string]any{
		"appId": appl.ID.String(),
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)
	envs := data["environments"].([]any)
	s.Require().NotEmpty(envs)

	// The raw value must not appear anywhere in the serialized tool output.
	raw, _ := json.Marshal(data)
	s.NotContains(string(raw), "super-secret-value")

	var found bool

	for _, e := range envs {
		build, _ := e.(map[string]any)["build"].(map[string]any)

		if build == nil {
			continue
		}

		vars, _ := build["vars"].(map[string]any)

		if v, ok := vars["SECRET_TOKEN"]; ok {
			found = true
			// Every value masked (keys kept), including SK_-prefixed names.
			s.Equal("", v, "env var value must be masked in MCP output")
			s.Equal("", vars["SK_SECRET"], "SK_-prefixed var value must be masked")
		}
	}

	s.True(found, "expected the seeded SECRET_TOKEN key to be present (masked)")
}

func (s *HandlerMCPSuite) Test_ListEnvironments_MissingAppId() {
	usr := s.MockUser()
	key := s.userKey(usr)
	resp := s.post(key.Value, mcpToolCall(1, "list_environments", map[string]any{}))
	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_ListDomains_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "list_domains", map[string]any{
		"envId": mockEnv.ID.String(),
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	_, ok := data["domains"]
	s.True(ok)
}

func (s *HandlerMCPSuite) Test_ListDomains_Forbidden_NotMember() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	appl := s.MockApp(usr1)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr2)

	resp := s.post(key.Value, mcpToolCall(1, "list_domains", map[string]any{
		"envId": mockEnv.ID.String(),
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_GetDeployment_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	depl := s.MockDeployment(mockEnv)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "get_deployment", map[string]any{
		"envId":        mockEnv.ID.String(),
		"deploymentId": depl.ID.String(),
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	got, ok := data["deployment"].(map[string]any)
	s.True(ok)
	s.Equal(depl.ID.String(), got["id"])
}

func (s *HandlerMCPSuite) Test_GetDeployment_MissingDeploymentId() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "get_deployment", map[string]any{
		"envId": mockEnv.ID.String(),
		// deploymentId omitted intentionally
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_GetDeployment_Forbidden_NotMember() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	appl := s.MockApp(usr1)
	mockEnv := s.MockEnv(appl)
	depl := s.MockDeployment(mockEnv)
	key := s.userKey(usr2)

	resp := s.post(key.Value, mcpToolCall(1, "get_deployment", map[string]any{
		"envId":        mockEnv.ID.String(),
		"deploymentId": depl.ID.String(),
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_PublishDeployment_MissingDeploymentId() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "publish_deployment", map[string]any{
		"envId": mockEnv.ID.String(),
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_CreateEnvironment_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "create_environment", map[string]any{
		"appId":  appl.ID.String(),
		"name":   "staging",
		"branch": "main",
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	// handlerEnvAdd returns {"envId": "<id>"}
	_, ok := data["envId"]
	s.True(ok, "expected 'envId' key in response, got: %v", data)
}

func (s *HandlerMCPSuite) Test_CreateEnvironment_Forbidden_NotMember() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	appl := s.MockApp(usr1)
	key := s.userKey(usr2)

	resp := s.post(key.Value, mcpToolCall(1, "create_environment", map[string]any{
		"appId":  appl.ID.String(),
		"name":   "staging",
		"branch": "main",
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_UpdateEnvironment_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)
	newBranch := "develop"

	resp := s.post(key.Value, mcpToolCall(1, "update_environment", map[string]any{
		"envId":  mockEnv.ID.String(),
		"branch": newBranch,
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	// handlerEnvUpdate returns {"ok": true}
	ok, _ := data["ok"].(bool)
	s.True(ok)
}

// The markdown flag is what turns content negotiation on for a deployment, so
// an agent has to be able to set it the same way it sets any other build
// setting.
func (s *HandlerMCPSuite) Test_UpdateEnvironment_SetsMarkdown() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "update_environment", map[string]any{
		"envId":    mockEnv.ID.String(),
		"markdown": true,
	}))

	s.rpcOK(resp)

	updated, err := buildconf.NewStore().EnvironmentByID(context.Background(), mockEnv.ID)

	s.Require().NoError(err)
	s.True(updated.Data.Markdown.ValueOrZero())
}

func (s *HandlerMCPSuite) Test_UpdateEnvironment_Forbidden_NotMember() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	appl := s.MockApp(usr1)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr2)

	resp := s.post(key.Value, mcpToolCall(1, "update_environment", map[string]any{
		"envId":  mockEnv.ID.String(),
		"branch": "develop",
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_Deploy_MissingEnvId() {
	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "deploy", map[string]any{
		"branch": "main",
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_Deploy_Forbidden_NotMember() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	appl := s.MockApp(usr1)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr2)

	resp := s.post(key.Value, mcpToolCall(1, "deploy", map[string]any{
		"envId": fmt.Sprintf("%d", mockEnv.ID),
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_Deploy_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "deploy", map[string]any{
		"envId": mockEnv.ID.String(),
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	_, ok := data["id"]
	s.True(ok, "expected deployment 'id' in response, got: %v", data)
	s.mockDeployer.AssertCalled(s.T(), "Deploy", mock.Anything, mock.Anything, mock.Anything)
}

func (s *HandlerMCPSuite) Test_Deploy_WithBranchOverride() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "deploy", map[string]any{
		"envId":  mockEnv.ID.String(),
		"branch": "feature/my-branch",
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	s.Equal("feature/my-branch", data["branch"])
}

func (s *HandlerMCPSuite) Test_Deploy_MissingEnvId_ShowsErrorText() {
	usr := s.MockUser()
	key := s.userKey(usr)
	resp := s.post(key.Value, mcpToolCall(1, "deploy", map[string]any{}))
	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
	content := result["content"].([]any)[0].(map[string]any)
	s.Contains(content["text"].(string), "envId")
}

func (s *HandlerMCPSuite) Test_GetDeployment_WithLogs() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	depl := s.MockDeployment(mockEnv)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "get_deployment", map[string]any{
		"envId":        mockEnv.ID.String(),
		"deploymentId": depl.ID.String(),
		"logs":         true,
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	got, ok := data["deployment"].(map[string]any)
	s.True(ok)
	s.Equal(depl.ID.String(), got["id"])
}

func (s *HandlerMCPSuite) Test_PublishDeployment_Forbidden_NotMember() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	appl := s.MockApp(usr1)
	mockEnv := s.MockEnv(appl)
	depl := s.MockDeployment(mockEnv)
	key := s.userKey(usr2)

	resp := s.post(key.Value, mcpToolCall(1, "publish_deployment", map[string]any{
		"envId":        mockEnv.ID.String(),
		"deploymentId": depl.ID.String(),
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_PublishDeployment_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	depl := s.MockDeployment(mockEnv, map[string]any{"ExitCode": null.IntFrom(0)})
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "publish_deployment", map[string]any{
		"envId":        mockEnv.ID.String(),
		"deploymentId": depl.ID.String(),
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	s.Equal(true, data["ok"])
}

func (s *HandlerMCPSuite) Test_PublishDeployment_RunningDeployment() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	depl := s.MockDeployment(mockEnv)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "publish_deployment", map[string]any{
		"envId":        mockEnv.ID.String(),
		"deploymentId": depl.ID.String(),
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_CreateEnvironment_MissingAppId() {
	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "create_environment", map[string]any{
		"name":   "staging",
		"branch": "main",
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
	content := result["content"].([]any)[0].(map[string]any)
	s.Contains(content["text"].(string), "appId")
}

func (s *HandlerMCPSuite) Test_CreateEnvironment_WithOptionalFields() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "create_environment", map[string]any{
		"appId":      appl.ID.String(),
		"name":       "preview",
		"branch":     "develop",
		"buildCmd":   "npm run build",
		"distFolder": "dist",
		"autoDeploy": true,
		"envVars":    map[string]any{"NODE_ENV": "preview"},
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	_, ok := data["envId"]
	s.True(ok, "expected 'envId' in response, got: %v", data)
}

func (s *HandlerMCPSuite) Test_UpdateEnvironment_MissingEnvId() {
	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "update_environment", map[string]any{
		"branch": "main",
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
	content := result["content"].([]any)[0].(map[string]any)
	s.Contains(content["text"].(string), "envId")
}

func (s *HandlerMCPSuite) Test_UpdateEnvironment_WithEnvVars() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "update_environment", map[string]any{
		"envId":   mockEnv.ID.String(),
		"envVars": map[string]any{"API_KEY": "secret"},
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	ok, _ := data["ok"].(bool)
	s.True(ok)
}

// Test_UpdateEnvironment_EnvVars_Merge verifies that envVars merges into the
// existing set. Values cannot be read back, so a caller cannot reconstruct the
// keys a wholesale replace would drop.
func (s *HandlerMCPSuite) Test_UpdateEnvironment_EnvVars_Merge() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "update_environment", map[string]any{
		"envId":   mockEnv.ID.String(),
		"envVars": map[string]any{"API_KEY": "secret"},
	}))

	data := s.toolContent(s.rpcOK(resp))
	s.Equal([]any{"API_KEY", "NODE_ENV"}, data["envVars"])

	updated, err := buildconf.NewStore().EnvironmentByID(context.Background(), mockEnv.ID)
	s.Require().NoError(err)
	s.Equal(map[string]string{"NODE_ENV": "production", "API_KEY": "secret"}, updated.Data.Vars)
}

func (s *HandlerMCPSuite) Test_UpdateEnvironment_EnvVars_Overwrite() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "update_environment", map[string]any{
		"envId":   mockEnv.ID.String(),
		"envVars": map[string]any{"NODE_ENV": "staging"},
	}))

	data := s.toolContent(s.rpcOK(resp))
	s.Equal([]any{"NODE_ENV"}, data["envVars"])

	updated, err := buildconf.NewStore().EnvironmentByID(context.Background(), mockEnv.ID)
	s.Require().NoError(err)
	s.Equal(map[string]string{"NODE_ENV": "staging"}, updated.Data.Vars)
}

// Test_UpdateEnvironment_EnvVars_EmptyValueUnsets verifies that an empty value
// removes the key, the only way to drop a variable through the tool.
func (s *HandlerMCPSuite) Test_UpdateEnvironment_EnvVars_EmptyValueUnsets() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "update_environment", map[string]any{
		"envId":   mockEnv.ID.String(),
		"envVars": map[string]any{"API_KEY": "secret", "NODE_ENV": ""},
	}))

	data := s.toolContent(s.rpcOK(resp))
	s.Equal([]any{"API_KEY"}, data["envVars"])

	updated, err := buildconf.NewStore().EnvironmentByID(context.Background(), mockEnv.ID)
	s.Require().NoError(err)
	s.Equal(map[string]string{"API_KEY": "secret"}, updated.Data.Vars)
}

func (s *HandlerMCPSuite) Test_UpdateEnvironment_EnvVars_UnsetLastKey() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "update_environment", map[string]any{
		"envId":   mockEnv.ID.String(),
		"envVars": map[string]any{"NODE_ENV": ""},
	}))

	data := s.toolContent(s.rpcOK(resp))
	s.Empty(data["envVars"])

	updated, err := buildconf.NewStore().EnvironmentByID(context.Background(), mockEnv.ID)
	s.Require().NoError(err)
	s.Empty(updated.Data.Vars)
}

func (s *HandlerMCPSuite) Test_UpdateEnvironment_WithoutEnvVars_LeavesVarsUntouched() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "update_environment", map[string]any{
		"envId":  mockEnv.ID.String(),
		"branch": "develop",
	}))

	s.rpcOK(resp)

	updated, err := buildconf.NewStore().EnvironmentByID(context.Background(), mockEnv.ID)
	s.Require().NoError(err)
	s.Equal(map[string]string{"NODE_ENV": "production"}, updated.Data.Vars)
}

// Test_UpdateEnvironment_PartialUpdate_BooleanFieldsNotZeroed verifies that updating
// a single non-boolean field (e.g. branch) does not zero out pre-existing boolean
// fields (e.g. AutoDeploy) that were not included in the update payload.
func (s *HandlerMCPSuite) Test_UpdateEnvironment_PartialUpdate_BooleanFieldsNotZeroed() {
	usr := s.MockUser()
	key := s.userKey(usr)
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl, map[string]any{
		"AutoDeploy":  true,
		"AutoPublish": false,
	})

	resp := s.post(key.Value, mcpToolCall(1, "update_environment", map[string]any{
		"envId":  mockEnv.ID.String(),
		"branch": "develop",
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	ok, _ := data["ok"].(bool)
	s.True(ok)

	updated, err := buildconf.NewStore().EnvironmentByID(context.Background(), mockEnv.ID)
	s.Require().NoError(err)
	s.True(updated.AutoDeploy, "AutoDeploy must not be zeroed by a partial update")
	s.False(updated.AutoPublish, "AutoPublish must not be zeroed by a partial update")
}

func (s *HandlerMCPSuite) Test_ListDomains_MissingEnvId() {
	usr := s.MockUser()
	key := s.userKey(usr)
	resp := s.post(key.Value, mcpToolCall(1, "list_domains", map[string]any{}))
	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
	content := result["content"].([]any)[0].(map[string]any)
	s.Contains(content["text"].(string), "envId")
}

func (s *HandlerMCPSuite) Test_CreateDomain_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "create_domain", map[string]any{
		"envId":  mockEnv.ID.String(),
		"domain": "app.example.com",
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	// HandlerDomainAdd returns {"domainId": "<id>", "token": "<token>"}
	_, ok := data["domainId"]
	s.True(ok, "expected 'domainId' in response, got: %v", data)
	s.NotEmpty(data["token"])

	domains, err := buildconf.DomainStore().Domains(context.Background(), buildconf.DomainFilters{
		EnvID: mockEnv.ID,
	})
	s.Require().NoError(err)
	s.Len(domains, 1)
	s.Equal("app.example.com", domains[0].Name)
}

func (s *HandlerMCPSuite) Test_CreateDomain_InvalidDomain() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "create_domain", map[string]any{
		"envId":  mockEnv.ID.String(),
		"domain": "not a domain",
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_CreateDomain_Forbidden_NotMember() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	appl := s.MockApp(usr1)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr2)

	resp := s.post(key.Value, mcpToolCall(1, "create_domain", map[string]any{
		"envId":  mockEnv.ID.String(),
		"domain": "app.example.com",
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_DeleteDomain_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	// Create a domain first via the tool, then delete it.
	createResp := s.post(key.Value, mcpToolCall(1, "create_domain", map[string]any{
		"envId":  mockEnv.ID.String(),
		"domain": "app.example.com",
	}))
	domainID := s.toolContent(s.rpcOK(createResp))["domainId"].(string)

	resp := s.post(key.Value, mcpToolCall(2, "delete_domain", map[string]any{
		"envId":    mockEnv.ID.String(),
		"domainId": domainID,
	}))

	s.rpcOK(resp)

	domains, err := buildconf.DomainStore().Domains(context.Background(), buildconf.DomainFilters{
		EnvID: mockEnv.ID,
	})
	s.Require().NoError(err)
	s.Empty(domains)
}

func (s *HandlerMCPSuite) Test_DeleteDomain_MissingDomainId() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "delete_domain", map[string]any{
		"envId": mockEnv.ID.String(),
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
	content := result["content"].([]any)[0].(map[string]any)
	s.Contains(content["text"].(string), "domainId")
}

func (s *HandlerMCPSuite) Test_DeleteDomain_Forbidden_NotMember() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	appl := s.MockApp(usr1)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr2)

	resp := s.post(key.Value, mcpToolCall(1, "delete_domain", map[string]any{
		"envId":    mockEnv.ID.String(),
		"domainId": "123",
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_RestartDeployment_MissingEnvId() {
	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "restart_deployment", map[string]any{
		"deploymentId": "123",
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_RestartDeployment_Forbidden_NotMember() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	appl := s.MockApp(usr1)
	mockEnv := s.MockEnv(appl)
	depl := s.MockDeployment(mockEnv, map[string]any{"ExitCode": null.IntFrom(1)})
	key := s.userKey(usr2)

	resp := s.post(key.Value, mcpToolCall(1, "restart_deployment", map[string]any{
		"envId":        mockEnv.ID.String(),
		"deploymentId": depl.ID.String(),
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_RestartDeployment_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	depl := s.MockDeployment(mockEnv, map[string]any{"ExitCode": null.IntFrom(1)})
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "restart_deployment", map[string]any{
		"envId":        mockEnv.ID.String(),
		"deploymentId": depl.ID.String(),
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	s.Equal(true, data["ok"])
}

func (s *HandlerMCPSuite) Test_CreateTeam_Success() {
	admin.SetMockLicense()
	defer func() {
		admin.ResetMockLicense()
		config.SetIsSelfHosted(false)
	}()

	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "create_team", map[string]any{
		"name": "My Awesome Team",
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)
	team := data["team"].(map[string]any)

	s.Equal("My Awesome Team", team["name"])
	s.Equal("my-awesome-team", team["slug"])
	s.Equal("owner", team["currentUserRole"])
}

func (s *HandlerMCPSuite) Test_CreateTeam_MissingName() {
	admin.SetMockLicense()
	defer func() {
		admin.ResetMockLicense()
		config.SetIsSelfHosted(false)
	}()

	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "create_team", map[string]any{}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_CreateTeam_Forbidden_NotEnterprise() {
	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "create_team", map[string]any{
		"name": "My Awesome Team",
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_ConfigureAuth_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "configure_auth", map[string]any{
		"envId":              mockEnv.ID.String(),
		"status":             true,
		"successUrl":         "/auth/success",
		"tokenTtl":           120,
		"oauthServerEnabled": true,
		"oauthResourcePath":  "mcp",
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	s.Equal(true, data["status"])
	s.Equal("/auth/success", data["successUrl"])
	s.Equal("/mcp", data["oauthResourcePath"])
	s.Equal(true, data["oauthServerEnabled"])

	stored, err := buildconf.NewStore().EnvironmentByID(context.Background(), mockEnv.ID)
	s.Require().NoError(err)
	s.Require().NotNil(stored.AuthConf)
	s.True(stored.AuthConf.Status)
	s.Len(stored.AuthConf.Secret, 128)
}

// Test_ConfigureAuth_IsPatch verifies that omitted fields keep their stored value.
func (s *HandlerMCPSuite) Test_ConfigureAuth_IsPatch() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	s.NoError(buildconf.NewStore().SaveAuthConf(context.Background(), mockEnv.ID, &buildconf.SKAuthConf{
		Secret:     "seed-secret",
		Status:     true,
		SuccessURL: "/dashboard",
		TTL:        60,
	}))

	resp := s.post(key.Value, mcpToolCall(1, "configure_auth", map[string]any{
		"envId":              mockEnv.ID.String(),
		"oauthServerEnabled": true,
	}))

	s.rpcOK(resp)

	stored, err := buildconf.NewStore().EnvironmentByID(context.Background(), mockEnv.ID)
	s.Require().NoError(err)
	s.True(stored.AuthConf.Status, "status must survive a patch that omits it")
	s.Equal("/dashboard", stored.AuthConf.SuccessURL, "successUrl must survive a patch that omits it")
	s.Equal(60, stored.AuthConf.TTL)
	s.Equal("seed-secret", stored.AuthConf.Secret)
	s.Require().NotNil(stored.AuthConf.OAuthServer)
	s.True(stored.AuthConf.OAuthServer.Enabled)
}

func (s *HandlerMCPSuite) Test_GetAuthConfig_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	s.NoError(buildconf.NewStore().SaveAuthConf(context.Background(), mockEnv.ID, &buildconf.SKAuthConf{
		Secret:     "seed-secret",
		Status:     true,
		SuccessURL: "/welcome",
		TTL:        30,
	}))

	resp := s.post(key.Value, mcpToolCall(1, "get_auth_config", map[string]any{
		"envId": mockEnv.ID.String(),
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	s.Equal(true, data["status"])
	s.Equal("/welcome", data["successUrl"])
	// The signing secret is never exposed.
	raw, _ := json.Marshal(data)
	s.NotContains(string(raw), "seed-secret")
}

func (s *HandlerMCPSuite) Test_ConfigureAuth_MissingEnvId() {
	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "configure_auth", map[string]any{
		"status": true,
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
	content := result["content"].([]any)[0].(map[string]any)
	s.Contains(content["text"].(string), "envId")
}

func (s *HandlerMCPSuite) Test_ConfigureAuth_Forbidden_NotMember() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	appl := s.MockApp(usr1)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr2)

	resp := s.post(key.Value, mcpToolCall(1, "configure_auth", map[string]any{
		"envId":  mockEnv.ID.String(),
		"status": true,
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_AuthTools_RejectedOnCloud() {
	config.SetIsStormkitCloud(true)
	defer config.SetIsStormkitCloud(false)

	usr := s.MockUser()
	key := s.userKey(usr)
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)

	resp := s.post(key.Value, mcpToolCall(1, "configure_auth", map[string]any{
		"envId":  mockEnv.ID.String(),
		"status": true,
	}))

	errObj := s.rpcError(resp)
	s.EqualValues(-32601, errObj["code"])
}

func TestHandlerMCP(t *testing.T) {
	suite.Run(t, &HandlerMCPSuite{})
}

// ---------------------------------------------------------------------------
// Mailer tools
// ---------------------------------------------------------------------------

func (s *HandlerMCPSuite) Test_ConfigureMailer_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "configure_mailer", map[string]any{
		"envId":    mockEnv.ID.String(),
		"smtpHost": "smtp.gmail.com",
		"smtpPort": "587",
		"username": "noreply@acme.com",
		"password": "super-secret",
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)
	conf := data["config"].(map[string]any)

	s.Equal("smtp.gmail.com", conf["host"])
	s.Equal("587", conf["port"])
	s.Equal("noreply@acme.com", conf["username"])
	s.Equal(buildconf.PasswordPlaceholder, conf["password"],
		"the password must never be echoed back into the agent transcript")

	stored, err := buildconf.NewStore().EnvironmentByID(context.Background(), mockEnv.ID)
	s.Require().NoError(err)
	s.Require().NotNil(stored.MailerConf)
	s.Equal("super-secret", stored.MailerConf.Password)
	s.Equal("smtp.gmail.com", stored.MailerConf.Host)
}

// Test_ConfigureMailer_IsPatch verifies that omitted fields — the password in
// particular — keep their stored value. The port is patched rather than the
// host because repointing the host deliberately drops the stored credential;
// see Test_ConfigureMailer_HostChangeRequiresPassword.
func (s *HandlerMCPSuite) Test_ConfigureMailer_IsPatch() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl, map[string]any{
		"MailerConf": &buildconf.MailerConf{
			Host:     "smtp.gmail.com",
			Port:     "587",
			Username: "noreply@acme.com",
			Password: "original-pwd",
		},
	})
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "configure_mailer", map[string]any{
		"envId":    mockEnv.ID.String(),
		"smtpPort": "2525",
	}))

	s.rpcOK(resp)

	stored, err := buildconf.NewStore().EnvironmentByID(context.Background(), mockEnv.ID)
	s.Require().NoError(err)
	s.Equal("original-pwd", stored.MailerConf.Password, "password must survive a patch that omits it")
	s.Equal("noreply@acme.com", stored.MailerConf.Username)
	s.Equal("smtp.gmail.com", stored.MailerConf.Host)
	s.Equal("2525", stored.MailerConf.Port)
}

// Test_ConfigureMailer_HostChangeRequiresPassword pins the credential-rebinding
// guard on the MCP surface: an agent holding an environment key can never
// repoint the relay while keeping a password it is not allowed to read, so the
// tool must report the error rather than silently leaving the config untouched.
func (s *HandlerMCPSuite) Test_ConfigureMailer_HostChangeRequiresPassword() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl, map[string]any{
		"MailerConf": &buildconf.MailerConf{
			Host:     "smtp.gmail.com",
			Port:     "587",
			Username: "noreply@acme.com",
			Password: "original-pwd",
		},
	})
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "configure_mailer", map[string]any{
		"envId":    mockEnv.ID.String(),
		"smtpHost": "smtp.attacker.tld",
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)

	s.True(result["isError"].(bool), "the agent must see the rejection")

	errs := s.toolContent(env)["errors"].(map[string]any)
	s.Equal("Password is a required field.", errs["password"])

	stored, err := buildconf.NewStore().EnvironmentByID(context.Background(), mockEnv.ID)
	s.Require().NoError(err)
	s.Equal("smtp.gmail.com", stored.MailerConf.Host, "the relay must not move")
	s.Equal("original-pwd", stored.MailerConf.Password)
}

func (s *HandlerMCPSuite) Test_ConfigureMailer_Invalid() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "configure_mailer", map[string]any{
		"envId":    mockEnv.ID.String(),
		"username": "noreply@acme.com",
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_GetMailerConfig_MasksPassword() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl, map[string]any{
		"MailerConf": &buildconf.MailerConf{
			Host:     "smtp.gmail.com",
			Port:     "587",
			Username: "noreply@acme.com",
			Password: "super-secret",
		},
	})
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "get_mailer_config", map[string]any{
		"envId": mockEnv.ID.String(),
	}))

	env := s.rpcOK(resp)
	conf := s.toolContent(env)["config"].(map[string]any)

	s.Equal("smtp.gmail.com", conf["host"])
	s.Equal("noreply@acme.com", conf["username"])
	s.Equal(buildconf.PasswordPlaceholder, conf["password"],
		"the stored SMTP password must never reach the agent")
}

func (s *HandlerMCPSuite) Test_GetMailerConfig_Forbidden() {
	usr := s.MockUser()
	other := s.MockUser()
	appl := s.MockApp(other)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "get_mailer_config", map[string]any{
		"envId": mockEnv.ID.String(),
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_SendTestEmail_NoMailerConfigured() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "send_test_email", map[string]any{
		"envId":   mockEnv.ID.String(),
		"to":      "someone@acme.com",
		"subject": "Test",
		"body":    "Hello",
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)

	s.True(result["isError"].(bool), "sending without a mailer must report an error, not a silent success")
	s.Contains(result["content"].([]any)[0].(map[string]any)["text"], "configure_mailer")
}

func (s *HandlerMCPSuite) Test_SendTestEmail_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl, map[string]any{
		"MailerConf": &buildconf.MailerConf{
			Host:     "smtp.gmail.com",
			Port:     "587",
			Username: "noreply@acme.com",
			Password: "super-secret",
		},
	})
	key := s.userKey(usr)

	sent := struct {
		addr string
		from string
		to   []string
		msg  []byte
	}{}

	buildconf.SendMailFunc = func(addr string, a smtp.Auth, from string, to []string, msg []byte) error {
		sent.addr, sent.from, sent.to, sent.msg = addr, from, to, msg
		return nil
	}

	resp := s.post(key.Value, mcpToolCall(1, "send_test_email", map[string]any{
		"envId":   mockEnv.ID.String(),
		"to":      "someone@acme.com",
		"from":    "noreply@acme.com",
		"subject": "Test",
		"body":    "Hello",
	}))

	env := s.rpcOK(resp)
	_, isError := env["result"].(map[string]any)["isError"]
	s.False(isError)

	s.Equal("smtp.gmail.com:587", sent.addr)
	s.Equal("noreply@acme.com", sent.from)
	s.Equal([]string{"someone@acme.com"}, sent.to)
	s.Contains(string(sent.msg), "Hello")
}

// ---------------------------------------------------------------------------
// Auth provider tool
// ---------------------------------------------------------------------------

func (s *HandlerMCPSuite) Test_ConfigureAuthProvider_MagicLink() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl, map[string]any{
		"SchemaConf": &buildconf.SchemaConf{
			Host:              s.conn.Cfg.Host,
			Port:              s.conn.Cfg.Port,
			DBName:            s.conn.Cfg.DBName,
			SchemaName:        s.conn.Cfg.Schema,
			AppUserName:       s.conn.Cfg.User,
			AppPassword:       s.conn.Cfg.Password,
			MigrationPassword: s.conn.Cfg.Password,
			MigrationUserName: s.conn.Cfg.User,
			MigrationsEnabled: true,
		},
	})
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "configure_auth_provider", map[string]any{
		"envId":        mockEnv.ID.String(),
		"providerName": skauth.ProviderMagicLink,
		"fromAddress":  "Acme <noreply@acme.com>",
		"status":       true,
	}))

	env := s.rpcOK(resp)
	_, isError := env["result"].(map[string]any)["isError"]
	s.False(isError)

	provider, err := skauth.NewStore().Provider(context.Background(), mockEnv.ID, skauth.ProviderMagicLink)
	s.Require().NoError(err)
	s.Require().NotNil(provider)
	s.True(provider.Status)
	s.Equal("Acme <noreply@acme.com>", provider.Data.FromAddress)
}

func (s *HandlerMCPSuite) Test_ConfigureAuthProvider_RequiresSchema() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "configure_auth_provider", map[string]any{
		"envId":        mockEnv.ID.String(),
		"providerName": skauth.ProviderMagicLink,
		"fromAddress":  "noreply@acme.com",
		"status":       true,
	}))

	env := s.rpcOK(resp)
	result := env["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_ConfigureAuthProvider_RejectedOnCloud() {
	config.SetIsStormkitCloud(true)
	defer config.SetIsStormkitCloud(false)

	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "configure_auth_provider", map[string]any{
		"envId":        mockEnv.ID.String(),
		"providerName": skauth.ProviderMagicLink,
	}))

	errObj := s.rpcError(resp)
	s.EqualValues(-32601, errObj["code"])
}

// Test_ListEmails_OmitsBody keeps live magic-link tokens out of an agent's
// transcript: the mailer log stores those emails verbatim.
func (s *HandlerMCPSuite) Test_ListEmails_OmitsBody() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	err := buildconf.MailerStore().InsertEmail(context.Background(), buildconf.Email{
		EnvID:   mockEnv.ID,
		To:      "someone@acme.com",
		From:    "noreply@acme.com",
		Subject: "Your magic link",
		Body:    `<a href="https://app.example.com/_stormkit/auth/magic?token=live-token">link</a>`,
	})

	s.Require().NoError(err)

	resp := s.post(key.Value, mcpToolCall(1, "list_emails", map[string]any{
		"envId": mockEnv.ID.String(),
	}))

	env := s.rpcOK(resp)
	emails := s.toolContent(env)["emails"].([]any)

	s.Require().Len(emails, 1)

	email := emails[0].(map[string]any)
	s.Equal("Your magic link", email["subject"])
	s.Equal("s***@acme.com", email["to"], "the recipient's local part must not reach an agent")
	s.NotContains(email, "body")
}

// Test_ConfigureAuthProvider_AcceptsHyphenatedName pins the wire string an
// agent actually sends. The canonical id is "magiclink"; the tool schema and
// the docs previously advertised "magic-link", which fell through to the OAuth
// branch and failed with "Client ID is required".
func (s *HandlerMCPSuite) Test_ConfigureAuthProvider_AcceptsHyphenatedName() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl, map[string]any{
		"SchemaConf": &buildconf.SchemaConf{
			Host:              s.conn.Cfg.Host,
			Port:              s.conn.Cfg.Port,
			DBName:            s.conn.Cfg.DBName,
			SchemaName:        s.conn.Cfg.Schema,
			AppUserName:       s.conn.Cfg.User,
			AppPassword:       s.conn.Cfg.Password,
			MigrationPassword: s.conn.Cfg.Password,
			MigrationUserName: s.conn.Cfg.User,
			MigrationsEnabled: true,
		},
	})
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "configure_auth_provider", map[string]any{
		"envId":        mockEnv.ID.String(),
		"providerName": "magic-link",
		"fromAddress":  "noreply@acme.com",
		"status":       true,
	}))

	env := s.rpcOK(resp)
	_, isError := env["result"].(map[string]any)["isError"]
	s.False(isError, "the hyphenated spelling must resolve to the magiclink provider")

	provider, err := skauth.NewStore().Provider(context.Background(), mockEnv.ID, skauth.ProviderMagicLink)
	s.Require().NoError(err)
	s.Require().NotNil(provider)
	s.Equal("noreply@acme.com", provider.Data.FromAddress)
}

// Test_ConfigureAuthProvider_KeepsStatusWhenOmitted covers the patch semantics
// the tool advertises: rotating a credential must not disable a live provider.
func (s *HandlerMCPSuite) Test_ConfigureAuthProvider_KeepsStatusWhenOmitted() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl, map[string]any{
		"SchemaConf": &buildconf.SchemaConf{
			Host:              s.conn.Cfg.Host,
			Port:              s.conn.Cfg.Port,
			DBName:            s.conn.Cfg.DBName,
			SchemaName:        s.conn.Cfg.Schema,
			AppUserName:       s.conn.Cfg.User,
			AppPassword:       s.conn.Cfg.Password,
			MigrationPassword: s.conn.Cfg.Password,
			MigrationUserName: s.conn.Cfg.User,
			MigrationsEnabled: true,
		},
	})
	key := s.userKey(usr)

	enable := s.post(key.Value, mcpToolCall(1, "configure_auth_provider", map[string]any{
		"envId":        mockEnv.ID.String(),
		"providerName": skauth.ProviderGoogle,
		"clientId":     "client-id",
		"clientSecret": "client-secret",
		"status":       true,
	}))

	s.rpcOK(enable)

	// Rotate the client id without mentioning status.
	rotate := s.post(key.Value, mcpToolCall(2, "configure_auth_provider", map[string]any{
		"envId":        mockEnv.ID.String(),
		"providerName": skauth.ProviderGoogle,
		"clientId":     "rotated-id",
	}))

	s.rpcOK(rotate)

	provider, err := skauth.NewStore().Provider(context.Background(), mockEnv.ID, skauth.ProviderGoogle)
	s.Require().NoError(err)
	s.Require().NotNil(provider)
	s.True(provider.Status, "omitting status must not disable a live provider")
	s.Equal("rotated-id", provider.Data.ClientID)
	s.Equal("client-secret", provider.Data.ClientSecret, "omitted secret must be retained")
}

// Test_ConfigureAuthProvider_KeepsFromAddressWhenOmitted covers the same patch
// semantics for the from address: nothing on the MCP surface can read it back,
// so requiring it on every write would make a live magiclink provider
// impossible to disable.
func (s *HandlerMCPSuite) Test_ConfigureAuthProvider_KeepsFromAddressWhenOmitted() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl, map[string]any{
		"SchemaConf": &buildconf.SchemaConf{
			Host:              s.conn.Cfg.Host,
			Port:              s.conn.Cfg.Port,
			DBName:            s.conn.Cfg.DBName,
			SchemaName:        s.conn.Cfg.Schema,
			AppUserName:       s.conn.Cfg.User,
			AppPassword:       s.conn.Cfg.Password,
			MigrationPassword: s.conn.Cfg.Password,
			MigrationUserName: s.conn.Cfg.User,
			MigrationsEnabled: true,
		},
	})
	key := s.userKey(usr)

	enable := s.post(key.Value, mcpToolCall(1, "configure_auth_provider", map[string]any{
		"envId":        mockEnv.ID.String(),
		"providerName": skauth.ProviderMagicLink,
		"fromAddress":  "Acme <noreply@acme.com>",
		"status":       true,
	}))

	s.rpcOK(enable)

	disable := s.post(key.Value, mcpToolCall(2, "configure_auth_provider", map[string]any{
		"envId":        mockEnv.ID.String(),
		"providerName": skauth.ProviderMagicLink,
		"status":       false,
	}))

	env := s.rpcOK(disable)
	_, isError := env["result"].(map[string]any)["isError"]
	s.False(isError, "a live magiclink provider must be disableable without resending the address")

	provider, err := skauth.NewStore().Provider(context.Background(), mockEnv.ID, skauth.ProviderMagicLink)
	s.Require().NoError(err)
	s.Require().NotNil(provider)
	s.False(provider.Status)
	s.Equal("Acme <noreply@acme.com>", provider.Data.FromAddress, "omitted from address must be retained")
}

// Test_ConfigureAuthProvider_CoercesQuotedStatus pins the fail-open direction
// of a mistyped argument: a quoted boolean must still disable the provider,
// because an ignored status silently keeps a live sign-in method enabled.
func (s *HandlerMCPSuite) Test_ConfigureAuthProvider_CoercesQuotedStatus() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl, map[string]any{
		"SchemaConf": &buildconf.SchemaConf{
			Host:              s.conn.Cfg.Host,
			Port:              s.conn.Cfg.Port,
			DBName:            s.conn.Cfg.DBName,
			SchemaName:        s.conn.Cfg.Schema,
			AppUserName:       s.conn.Cfg.User,
			AppPassword:       s.conn.Cfg.Password,
			MigrationPassword: s.conn.Cfg.Password,
			MigrationUserName: s.conn.Cfg.User,
			MigrationsEnabled: true,
		},
	})
	key := s.userKey(usr)

	enable := s.post(key.Value, mcpToolCall(1, "configure_auth_provider", map[string]any{
		"envId":        mockEnv.ID.String(),
		"providerName": skauth.ProviderEmail,
		"status":       true,
	}))

	s.rpcOK(enable)

	disable := s.post(key.Value, mcpToolCall(2, "configure_auth_provider", map[string]any{
		"envId":        mockEnv.ID.String(),
		"providerName": skauth.ProviderEmail,
		"status":       "false",
	}))

	s.rpcOK(disable)

	provider, err := skauth.NewStore().Provider(context.Background(), mockEnv.ID, skauth.ProviderEmail)
	s.Require().NoError(err)
	s.Require().NotNil(provider)
	s.False(provider.Status, "a quoted boolean must not be dropped into keep-existing")
}

// Test_ConfigureMailer_CoercesNumericPort covers the same coercion for a
// patch-style string field: a bare JSON number must not be dropped, or the tool
// answers 200 with the port it failed to change.
func (s *HandlerMCPSuite) Test_ConfigureMailer_CoercesNumericPort() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl, map[string]any{
		"MailerConf": &buildconf.MailerConf{
			Host:     "smtp.gmail.com",
			Port:     "587",
			Username: "noreply@acme.com",
			Password: "original-pwd",
		},
	})
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "configure_mailer", map[string]any{
		"envId":    mockEnv.ID.String(),
		"smtpPort": 2525,
	}))

	s.rpcOK(resp)

	stored, err := buildconf.NewStore().EnvironmentByID(context.Background(), mockEnv.ID)
	s.Require().NoError(err)
	s.Equal("2525", stored.MailerConf.Port, "a numeric port must not be silently ignored")
}
