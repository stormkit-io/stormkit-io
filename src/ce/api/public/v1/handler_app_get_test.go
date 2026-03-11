package publicapiv1_test

import (
	"encoding/json"
	"fmt"
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

// Test_Forbidden verifies that requests without a valid API key are rejected with 403.
func (s *HandlerAppGetSuite) Test_Forbidden() {
	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/apps/1",
		nil,
		map[string]string{
			"Authorization": "SK_invalid-key",
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_NotFound verifies that requesting a non-existent app returns 404.
func (s *HandlerAppGetSuite) Test_NotFound() {
	keyValue, _ := s.userKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/apps/999999999",
		nil,
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusNotFound, response.Code)
}

// Test_Forbidden_OtherTeam verifies that a user cannot retrieve an app belonging
// to a team they are not a member of.
func (s *HandlerAppGetSuite) Test_Forbidden_OtherTeam() {
	// Create the app owned by one user.
	_, a := s.userKey()

	// Create a second unrelated user who has no access to that app.
	usr2 := s.MockUser()
	key2 := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr2.ID,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
		"Scope":  apikey.SCOPE_USER,
	})

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/apps/%s", a.ID.String()),
		nil,
		map[string]string{
			"Authorization": key2.Value,
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_Success verifies that an authenticated member can retrieve their app by ID.
func (s *HandlerAppGetSuite) Test_Success() {
	keyValue, a := s.userKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/apps/%s", a.ID.String()),
		nil,
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	var body map[string]any
	s.NoError(json.Unmarshal([]byte(response.String()), &body))

	got, ok := body["app"].(map[string]any)
	s.True(ok)
	s.Equal(a.ID.String(), got["id"])
	s.Equal(a.DisplayName, got["displayName"])
	s.Equal(a.Repo, got["repo"])
	s.Equal(a.Repo == "", got["isBare"])
	s.Equal(a.UserID.String(), got["userId"])
	s.Equal(a.TeamID.String(), got["teamId"])
}

// Test_InvalidAppId_NonInteger verifies that a non-integer appId path param is rejected.
func (s *HandlerAppGetSuite) Test_InvalidAppId_NonInteger() {
	keyValue, _ := s.userKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/apps/not-an-id",
		nil,
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func TestHandlerAppGet(t *testing.T) {
	suite.Run(t, new(HandlerAppGetSuite))
}
