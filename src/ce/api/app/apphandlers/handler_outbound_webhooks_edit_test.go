package apphandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
	null "gopkg.in/guregu/null.v3"
)

type OutboundWebhooksUpdateHandlerSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *OutboundWebhooksUpdateHandlerSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *OutboundWebhooksUpdateHandlerSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *OutboundWebhooksUpdateHandlerSuite) Test_Success() {
	appl := s.MockApp(nil)
	store := app.NewStore()
	wh := app.OutboundWebhook{
		WebhookID:      1, // This will be generated on the fly
		RequestURL:     "https://api.discord.com/hooks/575VN415XU",
		RequestMethod:  "POST",
		RequestPayload: null.NewString(`{"message":"deployed $SK_DEPLOYMENT_ID"}`, true),
		TriggerWhen:    "on_deploy",
	}

	s.NoError(store.InsertOutboundWebhook(context.Background(), appl.ID, wh))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/app/outbound-webhooks",
		map[string]string{
			"appId":         appl.ID.String(),
			"whId":          wh.WebhookID.String(),
			"requestMethod": "HEAD",
			"requestUrl":    "https://api.example.org/hooks/575VN415XU",
			"triggerWhen":   "on_publish",
		},
		map[string]string{
			"Authorization": usertest.Authorization(appl.UserID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	whUpdated := app.NewStore().OutboundWebhook(context.Background(), appl.ID, wh.WebhookID)
	s.Nil(whUpdated.RequestHeaders)
	s.Equal("HEAD", whUpdated.RequestMethod)
	s.Equal("", whUpdated.RequestPayload.ValueOrZero())
	s.Equal(whUpdated.RequestPayload.Valid, false)
	s.Equal(whUpdated.TriggerWhen, "on_publish")
	s.Equal(whUpdated.RequestURL, "https://api.example.org/hooks/575VN415XU")
}

func (s *OutboundWebhooksUpdateHandlerSuite) Test_404() {
	appl := s.MockApp(nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/app/outbound-webhooks",
		map[string]any{
			"appId":         appl.ID.String(),
			"whId":          "1",
			"requestUrl":    "https://api.discord.com/hooks/575VN415XU",
			"triggerWhen":   "on_publish",
			"requestMethod": "HEAD",
			"requestHeaders": map[string]string{
				"Content-Type": "application/json",
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(appl.UserID),
		},
	)

	s.Equal(response.Code, http.StatusNotFound)
}

func TestOutboundWebhooksEdit(t *testing.T) {
	suite.Run(t, &OutboundWebhooksUpdateHandlerSuite{})
}
