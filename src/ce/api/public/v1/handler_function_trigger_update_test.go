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

// Test_Partial verifies that PATCH applies only the fields the body carries:
// a request that mentions the cron alone keeps the documentation, status, URL
// and headers it never mentioned, and an explicit empty value still clears.
func (s *HandlerFunctionTriggerUpdateSuite) Test_Partial() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	tf := s.MockTriggerFunction(env)

	store := functiontrigger.NewStore()
	before, err := store.ByID(context.Background(), tf.ID)
	s.Require().NoError(err)

	before.Documentation = "Ping #data when this fails."
	before.Options = functiontrigger.Options{
		Method:  "POST",
		URL:     "https://test.com/rollup",
		Payload: []byte(`{"full":true}`),
		Headers: shttp.Headers{"name": "joe"},
	}

	s.Require().NoError(store.Update(context.Background(), before))

	handler := shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
	auth := map[string]string{"Authorization": usertest.Authorization(usr.ID)}

	response := shttptest.RequestWithHeaders(
		handler,
		shttp.MethodPatch,
		"/v1/trigger",
		map[string]any{
			"id":    tf.ID.String(),
			"appId": app.ID.String(),
			"envId": env.ID.String(),
			"cron":  "5 5 * * *",
		},
		auth,
	)

	s.Equal(http.StatusOK, response.Code)

	record, err := store.ByID(context.Background(), tf.ID)
	s.Require().NoError(err)
	s.Equal("5 5 * * *", record.Cron)
	s.Equal("Ping #data when this fails.", record.Documentation)
	s.Equal(before.Status, record.Status)
	s.Equal("POST", record.Options.Method)
	s.Equal("https://test.com/rollup", record.Options.URL)
	s.Equal(`{"full":true}`, string(record.Options.Payload))
	s.Equal("name:joe", record.Options.Headers.String())

	response = shttptest.RequestWithHeaders(
		handler,
		shttp.MethodPatch,
		"/v1/trigger",
		map[string]any{
			"id":            tf.ID.String(),
			"appId":         app.ID.String(),
			"envId":         env.ID.String(),
			"documentation": "",
		},
		auth,
	)

	s.Equal(http.StatusOK, response.Code)

	record, err = store.ByID(context.Background(), tf.ID)
	s.Require().NoError(err)
	s.Empty(record.Documentation)
	s.Equal("5 5 * * *", record.Cron)
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
