package apphandlers_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/testutils"
	"github.com/stretchr/testify/suite"
	null "gopkg.in/guregu/null.v3"
)

type OutboundWebhooksSampleSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *OutboundWebhooksSampleSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *OutboundWebhooksSampleSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *OutboundWebhooksSampleSuite) TestOutboundWebhookSample_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)

	ms := testutils.MockServer()
	mr := testutils.MockResponse{
		Status:   200,
		Method:   shttp.MethodGet,
		DataText: "Success",
		Expect: func(req *http.Request) {
			b, _ := io.ReadAll(req.Body)
			post := string(b)

			s.Equal("application/json", req.Header.Get("Content-Type"))
			s.JSONEq(`{"deployment_id": "1", "env_name": "production"}`, post)
		},
	}

	ms.NewResponse("/", &mr)
	defer ms.Close()

	err := app.NewStore().InsertOutboundWebhook(context.Background(), appl.ID, app.OutboundWebhook{
		TriggerWhen:    app.TriggerOnPublish,
		RequestURL:     ms.URL(),
		RequestMethod:  shttp.MethodGet,
		RequestPayload: null.NewString(`{ "deployment_id": "$SK_DEPLOYMENT_ID", "env_name": "$SK_ENVIRONMENT" }`, true),
		RequestHeaders: map[string]string{
			"Content-Type": "application/json",
		},
	})

	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/app/%s/outbound-webhooks/1/trigger", appl.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(appl.UserID),
		},
	)

	expected := `{
		"error": "",
		"result": {
			"body": "Success",
			"status": 200
		}
	}`

	s.Nil(err)
	s.Equal(response.Code, http.StatusOK)
	s.JSONEq(expected, response.String())
	s.Equal(mr.NumberOfCalls, 1)
}

func TestOutboundWebhooksSample(t *testing.T) {
	suite.Run(t, &OutboundWebhooksSampleSuite{})
}
