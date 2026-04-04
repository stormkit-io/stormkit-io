package publicapiv1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
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
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockDeployer = &mocks.Deployer{}
	s.mockDeployer.On("Deploy", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deployservice.MockDeployer = s.mockDeployer
}

func (s *HandlerMCPSuite) AfterTest(_, _ string) {
	deployservice.MockDeployer = nil
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
		"deploy",
		"get_deployment",
		"publish_deployment",
		"list_apps",
		"list_environments",
		"create_environment",
		"update_environment",
		"list_domains",
	}, names)
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
	depl := s.MockDeployment(mockEnv)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "publish_deployment", map[string]any{
		"envId":        mockEnv.ID.String(),
		"deploymentId": depl.ID.String(),
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	s.Equal(true, data["ok"])
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

func TestHandlerMCP(t *testing.T) {
	suite.Run(t, &HandlerMCPSuite{})
}
