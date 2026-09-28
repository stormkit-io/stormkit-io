package apphandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerAppSettingsSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerAppSettingsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerAppSettingsSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetCache(context.Background())
}

func (s *HandlerAppSettingsSuite) settings(appl *factory.MockApp) map[string]any {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/app/%s/settings", appl.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(appl.UserID),
		},
	)

	s.Require().Equal(http.StatusOK, response.Code)

	return response.Map()
}

// Test_InboundWebhook_BitbucketDeployKey verifies that the webhook URL is shown
// when Bitbucket webhooks have to be registered by hand.
func (s *HandlerAppSettingsSuite) Test_InboundWebhook_BitbucketDeployKey() {
	appl := s.MockApp(nil, map[string]any{"Repo": "bitbucket/stormkit-test/test-repo"})
	s.MockEnv(appl)

	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{Bitbucket: admin.BitbucketConfig{DeployKey: "deploy-key"}}
	admin.SetConfig(&cnf)

	webhook, _ := s.settings(appl)["inboundWebhook"].(string)
	secret, found := strings.CutPrefix(webhook, cnf.ApiURL("/app/webhooks/bitbucket/"))

	s.True(found)
	s.NotEmpty(secret)
}

func (s *HandlerAppSettingsSuite) Test_InboundWebhook_HiddenWithoutDeployKey() {
	appl := s.MockApp(nil, map[string]any{"Repo": "bitbucket/stormkit-test/test-repo"})
	s.MockEnv(appl)

	s.NotContains(s.settings(appl), "inboundWebhook")
}

func (s *HandlerAppSettingsSuite) Test_Success() {
	appl := s.MockApp(nil)
	env := s.MockEnv(appl)
	dt := "4918AvvjzfkADxmczoedDAdvczvz"

	_, err := s.conn.Exec("UPDATE apps SET deploy_trigger = $1 WHERE app_id = 1", dt)
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/app/%s/settings", appl.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(appl.UserID),
		},
	)

	expected := fmt.Sprintf(`{"deployTrigger":"%s","runtime":"%s","envs":["%s"]}`, dt, config.DefaultNodeRuntime, env.Name)
	s.Equal(expected, response.String())
	s.Equal(http.StatusOK, response.Code)
}

func TestHandlerAppSettings(t *testing.T) {
	suite.Run(t, &HandlerAppSettingsSuite{})
}
