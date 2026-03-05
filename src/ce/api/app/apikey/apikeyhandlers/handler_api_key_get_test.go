package apikeyhandlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey/apikeyhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type HandlerAPIKeyGetSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerAPIKeyGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerAPIKeyGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAPIKeyGetSuite) Test_Success_ScopeEnv() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	key := s.MockAPIKey(app, env, map[string]any{
		"UserID": types.ID(0),
		"Value":  "SK_N32UH0PyJX7K5mMn9RcfpV7BnDK3R00tbuO4T22na2vvrBGv6cs9JlcM3mxfd9",
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/api-keys?envId=%s", key.EnvID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{
		"keys": [{
			"appId":"1",
			"envId":"1",
			"teamId": "",
			"id":"1",
			"scope":"env",
			"name": "Default"
		}]}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerAPIKeyGetSuite) Test_Success_ScopeTeam() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	s.MockAPIKey(app, env, map[string]any{
		"UserID": types.ID(0),
		"EnvID":  types.ID(0),
		"AppID":  types.ID(0),
		"TeamID": usr.DefaultTeamID,
		"Value":  "SK_N32UH0PyJX7K5mMn9RcfpV7BnDK3R00tbuO4T22na2vvrBGv6cs9JlcM3mxfd9",
		"Scope":  apikey.SCOPE_TEAM,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apikeyhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/api-keys?teamId=%d", usr.DefaultTeamID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"keys": [{
			"id":"1",
			"envId":"",
			"appId":"",
			"teamId":"%d",
			"scope":"team",
			"name": "Default"
		}]}`,
		usr.DefaultTeamID,
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerAPIKeyGet(t *testing.T) {
	suite.Run(t, &HandlerAPIKeyGetSuite{})
}
