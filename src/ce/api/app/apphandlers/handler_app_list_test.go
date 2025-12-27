package apphandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerAppListSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerAppListSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerAppListSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAppListSuite) Test_WithItems() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	app1 := s.MockApp(usr1)
	app2 := s.MockApp(usr1)
	env1 := s.MockEnv(app1)
	s.MockApp(usr2)

	defaultTeamID, err := team.NewStore().DefaultTeamID(context.Background(), usr1.ID)
	s.NoError(err)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/apps?teamId=%d", defaultTeamID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr1.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"apps": [{
		  "id": "%s",
		  "userId": "%s",
		  "teamId": "%d",
		  "defaultEnv": "production",
		  "defaultEnvId": "%s",
		  "createdAt": "1700489144",
		  "displayName": "%s",
		  "repo": "github/svedova/react-minimal",
		  "isBare": false
		}, {
		  "id": "%s",
		  "userId": "%s",
		  "teamId": "%d",
		  "defaultEnv": "production",
		  "defaultEnvId": "0",
		  "createdAt": "1700489144",
		  "displayName": "%s",
		  "repo": "github/svedova/react-minimal",
		  "isBare": false
		}],
		"hasNextPage": false
	}`,
		app1.ID.String(),
		usr1.ID.String(),
		usr1.DefaultTeamID,
		env1.ID.String(),
		app1.DisplayName,
		app2.ID.String(),
		usr1.ID.String(),
		usr1.DefaultTeamID,
		app2.DisplayName,
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerAppListSuite) Test_WithItems_Filtering() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	app1 := s.MockApp(usr1)
	s.MockApp(usr1)
	s.MockApp(usr2)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/apps?filter=%s", app1.DisplayName),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr1.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"apps": [{
		  "id": "%s",
		  "userId": "%s",
		  "teamId": "%d",
		  "defaultEnv": "production",
		  "defaultEnvId": "0",
		  "createdAt": "1700489144",
		  "displayName": "%s",
		  "repo": "github/svedova/react-minimal",
		  "isBare": false
		}],
		"hasNextPage": false
	}`,
		app1.ID.String(),
		usr1.ID.String(),
		usr1.DefaultTeamID,
		app1.DisplayName,
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerAppListSuite) Test_WithItems_Pagination() {
	usr1 := s.MockUser()
	s.MockApp(usr1)
	app2 := s.MockApp(usr1)
	s.MockApp(usr1)

	originalLimit := apphandlers.AppListLimit
	apphandlers.AppListLimit = 1

	defer func() {
		apphandlers.AppListLimit = originalLimit
	}()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/apps?from=1",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr1.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"apps": [{
		  "id": "%s",
		  "userId": "%s",
		  "teamId": "%d",
		  "defaultEnv": "production",
		  "defaultEnvId": "0",
		  "createdAt": "1700489144",
		  "displayName": "%s",
		  "repo": "github/svedova/react-minimal",
		  "isBare": false
		}],
		"hasNextPage": true
	}`,
		app2.ID.String(),
		usr1.ID.String(),
		usr1.DefaultTeamID,
		app2.DisplayName,
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerAppListSuite) Test_WithoutItems() {
	usr1 := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/apps",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr1.ID),
		},
	)

	str := response.String()
	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{ "apps": [], "hasNextPage": false }`, str)
}

func TestHandlerAppList(t *testing.T) {
	suite.Run(t, &HandlerAppListSuite{})
}
