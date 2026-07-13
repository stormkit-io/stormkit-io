package apphandlers_test

import (
	"context"
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

type OutboundWebhooksDeleteHandlerSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *OutboundWebhooksDeleteHandlerSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *OutboundWebhooksDeleteHandlerSuite) TearDownSuite() {
	s.conn.CloseTx()
}

func (s *OutboundWebhooksDeleteHandlerSuite) TestSuccess() {
	store := app.NewStore()
	appl := s.MockApp(nil)
	err := store.InsertOutboundWebhook(
		context.Background(), appl.ID, app.OutboundWebhook{
			RequestURL:    admin.MustConfig().AppURL("/"),
			RequestMethod: "POST",
			TriggerWhen:   app.TriggerOnDeploySuccess,
		})

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		"/app/outbound-webhooks",
		map[string]string{
			"appId": appl.ID.String(),
			"whId":  "1",
		},
		map[string]string{
			"Authorization": usertest.Authorization(appl.UserID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	whs := store.OutboundWebhooks(context.Background(), types.ID(1))
	s.Len(whs, 0)
}

func TestOutboundWebhooksDeleteHandler(t *testing.T) {
	suite.Run(t, &OutboundWebhooksDeleteHandlerSuite{})
}
