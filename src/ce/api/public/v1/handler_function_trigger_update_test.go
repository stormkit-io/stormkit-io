package publicapiv1_test

import (
	"context"
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
	"github.com/stretchr/testify/suite"
)

type HandlerFunctionTriggerUpdateSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerFunctionTriggerUpdateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerFunctionTriggerUpdateSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerFunctionTriggerUpdateSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	tf := s.MockTriggerFunction(env)

	triggerRequestPayload := `{
		"window": {
			"title": "Sample Konfabulator Widget",
			"name":"main_window"
		}
	}`

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPatch,
		"/v1/trigger",
		map[string]any{
			"id":     tf.ID.String(),
			"appId":  app.ID.String(),
			"envId":  env.ID.String(),
			"cron":   "5 5 * * *",
			"status": true,
			"options": map[string]any{
				"method":  "GET",
				"url":     "https://test.com/",
				"payload": triggerRequestPayload,
				"headers": map[string]string{
					"name":    "joe",
					"surname": "doe",
				},
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	record, err := functiontrigger.NewStore().ByID(context.Background(), tf.ID)

	s.NoError(err)
	s.Equal(http.StatusOK, response.Code)
	s.True(record.Status)
	s.Equal(record.Options.URL, "https://test.com/")
	s.Equal(record.Options.Headers.String(), "name:joe;surname:doe")
	s.True(record.NextRunAt.Valid)
	s.Equal(string(record.Options.Payload), triggerRequestPayload)
	s.Equal(string(record.Cron), "5 5 * * *")
}

func (s *HandlerFunctionTriggerUpdateSuite) Test_Permission() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	app2 := s.MockApp(usr)
	env2 := s.MockEnv(app2)
	tf2 := s.MockTriggerFunction(env2)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPatch,
		"/v1/trigger",
		map[string]any{
			"id":     tf2.ID.String(),
			"appId":  app.ID.String(),
			"envId":  env.ID.String(),
			"cron":   "5 5 * * *",
			"status": true,
			"options": map[string]any{
				"method": "GET",
				"url":    "https://test.com/",
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func (s *HandlerFunctionTriggerUpdateSuite) Test_FailValidation() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	tf := s.MockTriggerFunction(env)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPatch,
		"/v1/trigger",
		map[string]any{
			"id":     tf.ID.String(),
			"appId":  app.ID.String(),
			"envId":  env.ID.String(),
			"cron":   "X * * * *",
			"status": true,
			"options": map[string]any{
				"method": "GET",
				"url":    "not-a-url",
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func TestHandlerUpdateTrigger(t *testing.T) {
	suite.Run(t, &HandlerFunctionTriggerUpdateSuite{})
}
