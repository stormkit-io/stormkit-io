package apphandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type OutboundWebhooksSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *OutboundWebhooksSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *OutboundWebhooksSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

// Payload not nil, headers nil
func (s *OutboundWebhooksSuite) Test_Success() {
	triggerWhen := map[string]string{
		app.TriggerOnCachePurge:    "on_cache_purge",
		app.TriggerOnPublish:       "on_publish",
		app.TriggerOnDeployFailed:  "on_deploy_failed",
		app.TriggerOnDeploySuccess: "on_deploy_success",
		"on_deploy":                "on_deploy_success", // Backwards compatibility
	}

	for tw, expected := range triggerWhen {
		mockApp := s.MockApp(nil)

		response := shttptest.RequestWithHeaders(
			shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
			shttp.MethodPost,
			"/app/outbound-webhooks",
			map[string]string{
				"appId":          mockApp.ID.String(),
				"requestUrl":     "https://api.discord.com/hooks/575VN415XU",
				"requestMethod":  "POST",
				"requestPayload": `{"message":"deployed $SK_DEPLOYMENT_ID"}`,
				"triggerWhen":    tw,
			},
			map[string]string{
				"Authorization": usertest.Authorization(mockApp.User().ID),
			},
		)

		s.Equal(response.Code, http.StatusOK)

		whs := app.NewStore().OutboundWebhooks(context.Background(), mockApp.ID)
		s.Equal(len(whs), 1)
		s.Nil(whs[0].RequestHeaders)
		s.Equal(whs[0].RequestMethod, "POST")
		s.Equal(whs[0].RequestPayload.ValueOrZero(), `{"message":"deployed $SK_DEPLOYMENT_ID"}`)
		s.Equal(whs[0].RequestPayload.Valid, true)
		s.Equal(whs[0].TriggerWhen, expected)
		s.Equal(whs[0].RequestURL, "https://api.discord.com/hooks/575VN415XU")
	}
}

func (s *OutboundWebhooksSuite) Test_BadRequest() {
	mockApp := s.MockApp(nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/outbound-webhooks",
		map[string]any{
			"appId":         mockApp.ID.String(),
			"requestUrl":    "invalid_url",
			"triggerWhen":   "invalid_value",
			"requestMethod": "invalid_method",
		},
		map[string]string{
			"Authorization": usertest.Authorization(mockApp.User().ID),
		},
	)

	expected := `{
		"errors": {
			"requesUrl": "parse \"invalid_url\": invalid URI for request",
			"requestMethod":"Invalid requestMethod value. Accepted values are: POST | GET | HEAD",
			"triggerWhen":"Invalid triggerWhen value. Accepted values are: on_deploy_success | on_deploy_failed | on_publish | on_cache_purge"
		}
	}`

	s.Equal(response.Code, http.StatusBadRequest)
	s.JSONEq(response.String(), expected)

	whs := app.NewStore().OutboundWebhooks(context.Background(), types.ID(1))
	s.Equal(len(whs), 0)
}

func (s *OutboundWebhooksSuite) Test_PreventLoop() {
	mockApp := s.MockApp(nil, map[string]any{
		"DeployTrigger": "zlhuejysosgxe7",
	})

	cnf := admin.MustConfig()
	requestURL := cnf.ApiURL(fmt.Sprintf("/hooks/app/%s/deploy/zlhuejysosgxe7/sar34fdsa", mockApp.ID.String()))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/outbound-webhooks",
		map[string]any{
			"appId":         mockApp.ID.String(),
			"requestUrl":    requestURL,
			"triggerWhen":   "on_publish",
			"requestMethod": "HEAD",
			"requestHeaders": map[string]string{
				"Content-Type": "application/json",
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(mockApp.User().ID),
		},
	)
	expected := `{
		"error":"Can't use Trigger deploy link as outbound request","code":""
	}`

	s.Equal(response.Code, http.StatusBadRequest)
	s.JSONEq(response.String(), expected)
}

func TestOutboundWebhooks(t *testing.T) {
	suite.Run(t, &OutboundWebhooksSuite{})
}
