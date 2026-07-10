package maintenancehandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/maintenance"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/maintenance/maintenancehandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerConfigGetSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerConfigGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerConfigGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerConfigGetSuite) Test_ConfigGet_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	s.NoError(maintenance.Store().SetConfig(context.Background(), env.ID, &maintenance.Config{
		Status: "on",
	}))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(maintenancehandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/maintenance/config?envId="+env.ID.String(),
		nil,
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{ "maintenance": "on" }`, response.String())
}

func (s *HandlerConfigGetSuite) Test_ConfigGet_SuccessEmptyState() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(maintenancehandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/maintenance/config?envId="+env.ID.String(),
		nil,
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{ "maintenance": "" }`, response.String())
}

func TestHandlerConfigGetSuite(t *testing.T) {
	suite.Run(t, &HandlerConfigGetSuite{})
}
