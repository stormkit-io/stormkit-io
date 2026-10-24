package publicapiv1_test

import (
	"context"
	"net/http"
	"strings"
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

// readOnlyTools are the tools a read-only API key can see and call.
var readOnlyTools = []string{
	"get_deployment",
	"list_deployments",
	"get_runtime_logs",
	"get_access_logs",
	"get_analytics",
	"list_apps",
	"list_environments",
	"list_domains",
	"list_triggers",
	"get_trigger_logs",
	"list_teams",
	"get_mailer_config",
	"list_emails",
	"get_auth_config",
}

type HandlerMCPReadOnlySuite struct {
	suite.Suite
	*factory.Factory

	conn         databasetest.TestDB
	mockDeployer *mocks.Deployer
}

func (s *HandlerMCPReadOnlySuite) BeforeTest(suiteName, _ string) {
	config.SetIsStormkitCloud(false)

	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockDeployer = &mocks.Deployer{}
	s.mockDeployer.On("Deploy", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deployservice.MockDeployer = s.mockDeployer
}

func (s *HandlerMCPReadOnlySuite) AfterTest(_, _ string) {
	deployservice.MockDeployer = nil
	buildconf.SendMailFunc = buildconf.SendMailWithDeadline
	s.conn.CloseTx()
}

// userKey creates a full-access SCOPE_USER API key owned by usr.
func (s *HandlerMCPReadOnlySuite) userKey(usr *factory.MockUser) *factory.MockAPIKey {
	return s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr.ID,
		"Scope":  apikey.SCOPE_USER,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
	})
}

// readOnlyKey creates a read-only SCOPE_USER API key owned by usr.
func (s *HandlerMCPReadOnlySuite) readOnlyKey(usr *factory.MockUser) *factory.MockAPIKey {
	return s.MockAPIKey(nil, nil, map[string]any{
		"UserID":   usr.ID,
		"Scope":    apikey.SCOPE_USER,
		"AppID":    types.ID(0),
		"EnvID":    types.ID(0),
		"TeamID":   types.ID(0),
		"ReadOnly": true,
	})
}

// post sends a POST to /v1/mcp with the given body and Authorization header.
func (s *HandlerMCPReadOnlySuite) post(keyValue string, body any) map[string]any {
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

// tools lists the tools visible to the key, keyed by name.
func (s *HandlerMCPReadOnlySuite) tools(keyValue string) map[string]map[string]any {
	result := s.post(keyValue, mcpBody(1, "tools/list", map[string]any{}))
	tools := map[string]map[string]any{}

	for _, t := range result["tools"].([]any) {
		tool := t.(map[string]any)
		tools[tool["name"].(string)] = tool
	}

	return tools
}

func (s *HandlerMCPReadOnlySuite) Test_ToolsList_ReadOnlyKeySeesReadOnlyTools() {
	key := s.readOnlyKey(s.MockUser())
	tools := s.tools(key.Value)

	names := []string{}

	for name := range tools {
		names = append(names, name)
	}

	s.ElementsMatch(readOnlyTools, names)
}

// Test_ToolsList_Annotations verifies that every tool carries annotations and
// that exactly the read-only tools are marked as such.
func (s *HandlerMCPReadOnlySuite) Test_ToolsList_Annotations() {
	key := s.userKey(s.MockUser())
	tools := s.tools(key.Value)

	readOnly := []string{}

	for name, tool := range tools {
		annotations, ok := tool["annotations"].(map[string]any)
		s.Require().True(ok, "%s has no annotations", name)

		if annotations["readOnlyHint"] == true {
			readOnly = append(readOnly, name)
			s.Equal(false, annotations["destructiveHint"], name)
		}
	}

	s.ElementsMatch(readOnlyTools, readOnly)

	for _, name := range []string{"delete_deployment", "delete_domain", "delete_trigger", "stop_deployment", "publish_deployment"} {
		s.Equal(true, tools[name]["annotations"].(map[string]any)["destructiveHint"], name)
	}

	for _, name := range []string{"invoke_trigger", "send_test_email"} {
		s.Equal(true, tools[name]["annotations"].(map[string]any)["openWorldHint"], name)
	}

	s.Equal(false, tools["deploy"]["annotations"].(map[string]any)["destructiveHint"])
}

func (s *HandlerMCPReadOnlySuite) Test_ToolCall_ReadOnlyKeyCanRead() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	key := s.readOnlyKey(usr)

	result := s.post(key.Value, mcpToolCall(1, "list_apps", map[string]any{
		"teamId": appl.TeamID.String(),
	}))

	s.Nil(result["isError"], result)
}

func (s *HandlerMCPReadOnlySuite) Test_ToolCall_ReadOnlyKeyCannotWrite() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.readOnlyKey(usr)

	result := s.post(key.Value, mcpToolCall(1, "deploy", map[string]any{"envId": env.ID.String()}))

	s.Equal(true, result["isError"])

	text := result["content"].([]any)[0].(map[string]any)["text"].(string)
	s.True(strings.Contains(text, "read_only_api_key"), text)
	s.mockDeployer.AssertNotCalled(s.T(), "Deploy", mock.Anything, mock.Anything, mock.Anything)
}

// Test_ToolCall_ReadOnlyKeyUnknownTool verifies that an unknown tool gets the
// same JSON-RPC error whatever the key type.
func (s *HandlerMCPReadOnlySuite) Test_ToolCall_ReadOnlyKeyUnknownTool() {
	key := s.readOnlyKey(s.MockUser())

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/mcp",
		mcpToolCall(1, "no_such_tool", map[string]any{}),
		map[string]string{"Authorization": key.Value},
	)

	rpcErr, ok := resp.Map()["error"].(map[string]any)
	s.Require().True(ok)
	s.Equal(float64(-32601), rpcErr["code"])
}

// Test_ListDomains_ReadOnlyKeyHidesPrivateKey verifies that a custom
// certificate's private key is not returned to a read-only key.
func (s *HandlerMCPReadOnlySuite) Test_ListDomains_ReadOnlyKeyHidesPrivateKey() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	domain := &buildconf.DomainModel{
		AppID:      appl.ID,
		EnvID:      env.ID,
		Name:       "www.stormkit.io",
		Verified:   true,
		CustomCert: &buildconf.CustomCert{Value: "cert-value", Key: "private-key"},
	}

	s.Require().NoError(buildconf.DomainStore().Insert(context.Background(), domain))
	s.Require().NoError(buildconf.DomainStore().UpdateDomainCert(context.Background(), domain))

	key := s.readOnlyKey(usr)

	result := s.post(key.Value, mcpToolCall(1, "list_domains", map[string]any{"envId": env.ID.String()}))
	text := result["content"].([]any)[0].(map[string]any)["text"].(string)

	s.Nil(result["isError"], text)
	s.Contains(text, domain.Name)
	s.Contains(text, "cert-value")
	s.NotContains(text, "private-key")
}

func TestHandlerMCPReadOnlySuite(t *testing.T) {
	suite.Run(t, &HandlerMCPReadOnlySuite{})
}
