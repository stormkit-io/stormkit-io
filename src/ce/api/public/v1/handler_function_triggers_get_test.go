package publicapiv1_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type HandlerTriggerFunctionGetSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerTriggerFunctionGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	utils.NewUnix = factory.MockNewUnix
	admin.SetMockLicense()
}

func (s *HandlerTriggerFunctionGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	utils.NewUnix = factory.OriginalNewUnix
	admin.ResetMockLicense()
}

func (s *HandlerTriggerFunctionGetSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	s.MockTriggerFunction(env, map[string]any{
		"Options": functiontrigger.Options{
			Method:  "POST",
			Payload: []byte(`{ "hello": "world" }`),
		},
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/triggers?appId=%d&envId=%d", app.ID, env.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{
		"triggers": [{
			"id": "1",
			"envId": "1",
			"cron": "*/1 * * * *",
			"description": "",
			"documentation": "",
			"nextRunAt": 1712418330,
			"options": {
				"method": "POST",
				"payload": "{ \"hello\": \"world\" }",
				"headers": null,
				"url": ""
			},
			"status": true
		}]
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerTriggerFunctionGetSuite) Test_MasksHeaders() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	s.MockTriggerFunction(env, map[string]any{
		"Options": functiontrigger.Options{
			Method:  "GET",
			URL:     "https://example.org",
			Headers: shttp.Headers{"Authorization": "Bearer secret-token"},
		},
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/triggers?appId=%d&envId=%d", app.ID, env.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	str := response.String()

	s.Equal(http.StatusOK, response.Code)
	s.NotContains(str, "secret-token")
	s.Contains(str, `"Authorization":""`)
}

func TestHandlerTrigger(t *testing.T) {
	suite.Run(t, &HandlerTriggerFunctionGetSuite{})
}
