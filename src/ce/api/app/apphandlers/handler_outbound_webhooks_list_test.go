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
	"github.com/stretchr/testify/suite"
)

type OutboundWebhooksListSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *OutboundWebhooksListSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *OutboundWebhooksListSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *OutboundWebhooksListSuite) Test_HandlerOutboundWebhookList_Empty() {
	usr := s.MockUser(nil)
	appl := s.MockApp(usr)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/app/%s/outbound-webhooks", appl.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(appl.UserID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{ "webhooks": [] }`, response.String())
}

func (s *OutboundWebhooksListSuite) Test_HandlerOutboundWebhookList_NotEmpty() {
	usr := s.MockUser(nil)
	appl := s.MockApp(usr)

	app.NewStore().InsertOutboundWebhook(
		context.Background(), appl.ID, app.OutboundWebhook{
			RequestURL:    "https://discord.com/hooks/test",
			RequestMethod: "GET",
			TriggerWhen:   app.TriggerOnDeploySuccess,
		})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/app/%s/outbound-webhooks", appl.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(appl.UserID),
		},
	)

	expected := `{
		"webhooks": [{
			"id": "1",
			"requestUrl": "https://discord.com/hooks/test",
			"requestMethod": "GET",
			"requestPayload": null,
			"requestHeaders": null,
			"triggerWhen": "on_deploy_success"
		}]
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestOutboundWebhooksList(t *testing.T) {
	suite.Run(t, &OutboundWebhooksListSuite{})
}
