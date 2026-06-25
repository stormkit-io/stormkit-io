package publicapiv1_test

import (
	"context"
	"fmt"
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

type HandlerFunctionTriggerDeleteSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerFunctionTriggerDeleteSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerFunctionTriggerDeleteSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerFunctionTriggerDeleteSuite) Test_Delete() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	tf := s.MockTriggerFunction(env)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/v1/trigger?triggerId=%d&envId=%d&appId=%d", tf.ID, env.ID, app.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
}

func (s *HandlerFunctionTriggerDeleteSuite) Test_Permission() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	app2 := s.MockApp(usr)
	env2 := s.MockEnv(app2)
	tf2 := s.MockTriggerFunction(env2)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/v1/trigger?triggerId=%d&envId=%d&appId=%d", tf2.ID, env.ID, app.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	f, err := functiontrigger.NewStore().ByID(context.Background(), tf2.ID)

	s.Equal(http.StatusNotFound, response.Code)
	s.NoError(err)
	s.NotNil(f)
}

func TestHandlerDeleteTrigger(t *testing.T) {
	suite.Run(t, &HandlerFunctionTriggerDeleteSuite{})
}
