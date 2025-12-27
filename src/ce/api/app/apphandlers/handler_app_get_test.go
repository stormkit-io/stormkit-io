package apphandlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
	null "gopkg.in/guregu/null.v3"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type AppGetSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *AppGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *AppGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *AppGetSuite) Test_Success() {
	user := s.MockUser()
	appl := s.MockApp(user, map[string]any{"AutoDeploy": null.NewString("commit", true)})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/app/%s", appl.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(user.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"app": {
		  "id": "%s",
		  "userId": "%s",
		  "teamId": "%d",
		  "createdAt": "1700489144",
		  "defaultEnv": "production",
		  "defaultEnvId": "0",
		  "displayName": "%s",
		  "repo": "github/svedova/react-minimal",
		  "isBare": false
		}
	  }`,
		appl.ID.String(),
		user.ID.String(),
		user.DefaultTeamID,
		appl.DisplayName,
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

// This should never occur because when an app is created, a production environment
// is always created with it. But since it happened once during testing that the
// environment was forgotton to be mocked, this spec was created to test that impractical case.
func (s *AppGetSuite) Test_FailEnvNotFound() {
	app := s.MockApp(nil, map[string]any{"AutoDeploy": null.NewString("commit", true)})
	usr := app.GetUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/app/%d", app.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusNotFound, response.Code)
}

func TestHandlerAppGet(t *testing.T) {
	suite.Run(t, &AppGetSuite{})
}
