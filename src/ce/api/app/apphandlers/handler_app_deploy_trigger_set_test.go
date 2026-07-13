package apphandlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerAppDeployTriggerSetSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerAppDeployTriggerSetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerAppDeployTriggerSetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAppDeployTriggerSetSuite) Test_Success() {
	appl := s.MockApp(nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/app/deploy-trigger",
		map[string]any{
			"appId": appl.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(appl.UserID),
		},
	)

	s.Equal(http.StatusCreated, response.Code)

	res := map[string]string{}
	_ = json.Unmarshal([]byte(response.String()), &res)
	hashFromRes := res["hash"]

	var hashFromDB string
	_ = s.conn.QueryRow("SELECT deploy_trigger FROM apps WHERE app_id = 1").Scan(&hashFromDB)
	s.Equal(hashFromDB, hashFromRes)
}

func TestHandlerAppDeployTriggerSet(t *testing.T) {
	suite.Run(t, &HandlerAppDeployTriggerSetSuite{})
}
