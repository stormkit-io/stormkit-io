package buildconfhandlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/buildconfhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerEnvGetSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerEnvGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerEnvGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerEnvGetSuite) Test_Success() {
	env := s.MockEnv(nil, map[string]any{
		"AutoDeploy":  false,
		"AutoPublish": true,
	})

	appl := env.GetApp()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(buildconfhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/app/%d/envs/%s", appl.ID, env.Name),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(1),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	exp := `{
		"config":{
		   "name":"production",
		   "env":"production",
		   "branch":"main",
		   "build":{
			  "distFolder":"build",
			  "buildCmd":"npm run build",
			  "previewLinks": null,
			  "vars":{
				 "NODE_ENV":"production"
			  }
		   },
		   "autoPublish":true,
		   "id":"1",
		   "appId":"1",
		   "autoDeploy": false,
		   "autoDeployBranches": null,
		   "autoDeployCommits": null,
		   "authConf": null,
		   "domain":{
			  "verified": false
		   }
		}
	 }`

	s.JSONEq(exp, response.String())
}

func TestHandlerEnvGet(t *testing.T) {
	suite.Run(t, &HandlerEnvGetSuite{})
}
