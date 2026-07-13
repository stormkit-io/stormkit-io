package apphandlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerAppSettingsSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerAppSettingsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerAppSettingsSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAppSettingsSuite) Test_Success() {
	appl := s.MockApp(nil)
	env := s.MockEnv(appl)
	dt := "4918AvvjzfkADxmczoedDAdvczvz"

	_, err := s.conn.Exec("UPDATE apps SET deploy_trigger = $1 WHERE app_id = 1", dt)
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/app/%s/settings", appl.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(appl.UserID),
		},
	)

	expected := fmt.Sprintf(`{"deployTrigger":"%s","runtime":"%s","envs":["%s"]}`, dt, config.DefaultNodeRuntime, env.Name)
	s.Equal(expected, response.String())
	s.Equal(http.StatusOK, response.Code)
}

func TestHandlerAppSettings(t *testing.T) {
	suite.Run(t, &HandlerAppSettingsSuite{})
}
