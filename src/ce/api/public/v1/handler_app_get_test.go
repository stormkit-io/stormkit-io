package publicapiv1_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type HandlerAppGetSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerAppGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerAppGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAppGetSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerAppGetSuite) userKey() (string, *factory.MockApp) {
	usr := s.MockUser()
	a := s.MockApp(usr)
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr.ID,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
		"Scope":  apikey.SCOPE_USER,
	})

	return key.Value, a
}

func (s *HandlerAppGetSuite) appKey() (string, *factory.MockApp) {
	usr := s.MockUser()
	a := s.MockApp(usr)
	key := s.MockAPIKey(nil, nil, map[string]any{
		"AppID":  a.ID,
		"UserID": types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
		"Scope":  apikey.SCOPE_APP,
	})

	return key.Value, a
}

// Test_Forbidden verifies that requests without a valid API key are rejected with 403.
func (s *HandlerAppGetSuite) Test_Forbidden() {
	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/app",
		nil,
		map[string]string{
			"Authorization": "SK_invalid-key",
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_Success verifies that an authenticated member can retrieve their app by ID.
func (s *HandlerAppGetSuite) Test_Success() {
	keyValueUser, app1 := s.userKey()
	keyValueApp, app2 := s.appKey()

	params := []struct {
		token string
		query string
		app   *factory.MockApp
	}{
		{token: keyValueUser, query: app1.ID.String(), app: app1},
		{token: keyValueApp, app: app2},
	}

	for _, p := range params {
		response := shttptest.RequestWithHeaders(
			s.handler(),
			shttp.MethodGet,
			"/v1/app?appId="+p.query,
			nil,
			map[string]string{
				"Authorization": p.token,
			},
		)

		s.Equal(http.StatusOK, response.Code)

		body := response.Map()

		got, ok := body["app"].(map[string]any)
		s.True(ok)
		s.Equal(p.app.ID.String(), got["id"])
		s.Equal(p.app.DisplayName, got["displayName"])
		s.Equal(p.app.Repo, got["repo"])
		s.Equal(p.app.Repo == "", got["isBare"])
		s.Equal(p.app.UserID.String(), got["userId"])
		s.Equal(p.app.TeamID.String(), got["teamId"])
	}
}

func TestHandlerAppGet(t *testing.T) {
	suite.Run(t, new(HandlerAppGetSuite))
}
