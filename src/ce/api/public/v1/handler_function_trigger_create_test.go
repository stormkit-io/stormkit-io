package publicapiv1_test

import (
	"context"
	"net/http"
	"strings"
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

type HandlerFunctionTriggerCreateSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerFunctionTriggerCreateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerFunctionTriggerCreateSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerFunctionTriggerCreateSuite) Test_Success_Enabled() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	triggerRequestPayload := `
		"window": {
	        "title": "Sample Konfabulator Widget",
	        "name": "main_window",
	        "width": 500,
	        "height": 500
	    },
	`

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/trigger",

		map[string]any{
			"appId":  env.AppID.String(),
			"envId":  env.ID.String(),
			"cron":   "*/1 * * * *",
			"status": true,
			"options": map[string]any{
				"method": "GET",
				"headers": map[string]string{
					"name":    "joe",
					"surname": "doe",
				},
				"url":     "https://test.com/",
				"payload": triggerRequestPayload,
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(env.GetApp().UserID),
		},
	)

	s.Equal(http.StatusCreated, response.Code)

	tfs, err := functiontrigger.NewStore().List(context.Background(), env.ID)
	s.NoError(err)
	s.Len(tfs, 1)
	s.True(tfs[0].Status)
	s.False(tfs[0].UpdatedAt.Valid)
	s.True(tfs[0].NextRunAt.Valid) // Because status is true
	s.True(tfs[0].CreatedAt.Valid)
	s.Equal(tfs[0].Options.URL, "https://test.com/")
	s.Equal(tfs[0].Options.Headers.String(), "name:joe;surname:doe")
	s.Equal(string(tfs[0].Options.Payload), triggerRequestPayload)
}

func (s *HandlerFunctionTriggerCreateSuite) Test_Success_Documentation() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	documentation := "# Nightly rollup\n\nPings `/cron/rollup`. Ask **#data** if it fails."

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/trigger",

		map[string]any{
			"appId":         env.AppID.String(),
			"envId":         env.ID.String(),
			"cron":          "*/1 * * * *",
			"status":        true,
			"documentation": documentation,
			"options": map[string]any{
				"method": "GET",
				"url":    "https://test.com/",
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(env.GetApp().UserID),
		},
	)

	s.Equal(http.StatusCreated, response.Code)

	tfs, err := functiontrigger.NewStore().List(context.Background(), env.ID)
	s.NoError(err)
	s.Len(tfs, 1)
	s.Equal(documentation, tfs[0].Documentation)
}

// Test_Fail_DocumentationTooLong verifies that the free-form notes are bounded:
// the column is plain text and every listing returns it in full.
func (s *HandlerFunctionTriggerCreateSuite) Test_Fail_DocumentationTooLong() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/trigger",

		map[string]any{
			"appId":         env.AppID.String(),
			"envId":         env.ID.String(),
			"cron":          "*/1 * * * *",
			"status":        true,
			"documentation": strings.Repeat("a", 64*1024+1),
			"options": map[string]any{
				"method": "GET",
				"url":    "https://test.com/",
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(env.GetApp().UserID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "documentation")

	tfs, err := functiontrigger.NewStore().List(context.Background(), env.ID)
	s.Require().NoError(err)
	s.Empty(tfs)
}

// Test_Success_NoDocumentation verifies that a trigger created without
// documentation reads back as an empty string rather than failing the scan.
func (s *HandlerFunctionTriggerCreateSuite) Test_Success_NoDocumentation() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/trigger",

		map[string]any{
			"appId":  env.AppID.String(),
			"envId":  env.ID.String(),
			"cron":   "*/1 * * * *",
			"status": true,
			"options": map[string]any{
				"method": "GET",
				"url":    "https://test.com/",
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(env.GetApp().UserID),
		},
	)

	s.Equal(http.StatusCreated, response.Code)

	tfs, err := functiontrigger.NewStore().List(context.Background(), env.ID)
	s.NoError(err)
	s.Len(tfs, 1)
	s.Empty(tfs[0].Documentation)
}

func (s *HandlerFunctionTriggerCreateSuite) Test_Success_Disabled() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/trigger",

		map[string]any{
			"appId":  env.AppID.String(),
			"envId":  env.ID.String(),
			"cron":   "*/1 * * * *",
			"status": false,
			"options": map[string]any{
				"method":  "HEAD",
				"headers": nil,
				"url":     "https://test.com/",
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(env.GetApp().UserID),
		},
	)

	s.Equal(http.StatusCreated, response.Code)

	tfs, err := functiontrigger.NewStore().List(context.Background(), env.ID)
	s.NoError(err)
	s.Len(tfs, 1)
	s.False(tfs[0].Status)
	s.False(tfs[0].UpdatedAt.Valid)
	s.False(tfs[0].NextRunAt.Valid) // Because status is false
	s.True(tfs[0].CreatedAt.Valid)
	s.Equal(tfs[0].Options.URL, "https://test.com/")
	s.Nil(tfs[0].Options.Headers)
	s.Nil(tfs[0].Options.Payload)
}

func (s *HandlerFunctionTriggerCreateSuite) Test_Success_HeadersAsString() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	triggerRequestPayload := `
		"window": {
	        "title": "Sample Konfabulator Widget",
	        "name": "main_window",
	        "width": 500,
	        "height": 500
	    },
	`

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/trigger",

		map[string]any{
			"appId":  env.AppID.String(),
			"envId":  env.ID.String(),
			"cron":   "*/1 * * * *",
			"status": true,
			"options": map[string]any{
				"method":  "GET",
				"headers": "name:joe;surname:doe",
				"url":     "https://test.com/",
				"payload": triggerRequestPayload,
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(env.GetApp().UserID),
		},
	)

	s.Equal(http.StatusCreated, response.Code)

	tfs, err := functiontrigger.NewStore().List(context.Background(), env.ID)
	s.NoError(err)
	s.Len(tfs, 1)
	s.True(tfs[0].Status)
	s.False(tfs[0].UpdatedAt.Valid)
	s.True(tfs[0].NextRunAt.Valid) // Because status is true
	s.True(tfs[0].CreatedAt.Valid)
	s.Equal(tfs[0].Options.URL, "https://test.com/")
	s.Equal(tfs[0].Options.Headers.String(), "name:joe;surname:doe")
	s.Equal(string(tfs[0].Options.Payload), triggerRequestPayload)
}

func (s *HandlerFunctionTriggerCreateSuite) Test_FailValidation() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPost,
		"/v1/trigger",
		map[string]any{
			"appId":    env.AppID.String(),
			"envId":    env.ID.String(),
			"cron":     "X * * * *",
			"timeZone": "Europe/Dublin",
			"options": map[string]any{
				"method":  "GET",
				"headers": "name=can;surname=eldem",
				"url":     "https://can.com/",
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(env.GetApp().UserID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func TestHandlerCreateTrigger(t *testing.T) {
	suite.Run(t, &HandlerFunctionTriggerCreateSuite{})
}
