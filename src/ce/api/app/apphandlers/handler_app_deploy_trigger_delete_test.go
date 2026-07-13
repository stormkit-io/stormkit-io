package apphandlers_test

import (
	"context"
	"fmt"
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
)

type HandlerDeleteTriggerSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerDeleteTriggerSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerDeleteTriggerSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerDeleteTriggerSuite) Test_RemovingDeployTrigger() {
	mockApp := s.Factory.MockApp(nil, map[string]any{
		"DeployTrigger": "12345",
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/app/%s/deploy-trigger", mockApp.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(mockApp.User().ID),
		},
	)

	s.Equal(response.Code, http.StatusOK)

	app, err := app.NewStore().AppByID(context.Background(), mockApp.ID)
	s.NoError(err)
	s.Empty(app.DeployTrigger)
}

func TestHandlerDeleteTrigger(t *testing.T) {
	suite.Run(t, &HandlerDeleteTriggerSuite{})
}
