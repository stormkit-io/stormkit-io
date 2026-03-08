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

func (s *HandlerAppListSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerAppListSuite) userKey() (string, *factory.MockApp) {
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

// Test_Forbidden verifies that requests without a valid user-scoped API key
// are rejected with 403.
func (s *HandlerAppListSuite) Test_Forbidden() {
	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/apps",
		nil,
		map[string]string{
			"Authorization": "SK_invalid-key",
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_InvalidFrom verifies that a negative 'from' value is rejected.
func (s *HandlerAppListSuite) Test_InvalidFrom() {
	keyValue, _ := s.userKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/apps?from=-1",
		nil,
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

// Test_InvalidFrom_NonInteger verifies that a non-integer 'from' value is rejected.
func (s *HandlerAppListSuite) Test_InvalidFrom_NonInteger() {
	keyValue, _ := s.userKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/apps?from=abc",
		nil,
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

// Test_MissingTeamId verifies that omitting the required teamId parameter returns 400.
func (s *HandlerAppListSuite) Test_MissingTeamId() {
	keyValue, _ := s.userKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/apps",
		nil,
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

// Test_Success verifies that an authenticated user can retrieve their apps.
func (s *HandlerAppListSuite) Test_Success() {
	keyValue, a := s.userKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/apps?teamId=%s", a.TeamID.String()),
		nil,
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	var body map[string]any
	s.NoError(json.Unmarshal([]byte(response.String()), &body))

	apps, _ := body["apps"].([]any)

	s.False(body["hasNextPage"].(bool))
	s.Len(apps, 1)

	first := apps[0].(map[string]any)
	s.Equal(a.ID.String(), first["id"])
	s.Equal(a.DisplayName, first["displayName"])
	s.Equal(a.Repo, first["repo"])
	s.Equal(a.Repo == "", first["isBare"])
	s.Equal(a.UserID.String(), first["userId"])
	s.Equal(a.TeamID.String(), first["teamId"])
}

// Test_FilterByRepo verifies that the repo query parameter restricts results to exact matches.
func (s *HandlerAppListSuite) Test_FilterByRepo() {
	keyValue, a := s.userKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/apps?teamId=%s&repo=%s", a.TeamID.String(), a.Repo),
		nil,
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	var body map[string]any
	s.NoError(json.Unmarshal([]byte(response.String()), &body))

	apps, _ := body["apps"].([]any)
	s.Len(apps, 1)
	s.Equal(a.ID.String(), apps[0].(map[string]any)["id"])
}

// Test_FilterByDisplayName verifies that the displayName query parameter restricts results to exact matches.
func (s *HandlerAppListSuite) Test_FilterByDisplayName() {
	keyValue, a := s.userKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/apps?teamId=%s&displayName=%s", a.TeamID.String(), a.DisplayName),
		nil,
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	var body map[string]any
	s.NoError(json.Unmarshal([]byte(response.String()), &body))

	apps, _ := body["apps"].([]any)
	s.Len(apps, 1)
	s.Equal(a.ID.String(), apps[0].(map[string]any)["id"])
}

// Test_InvalidTeamId_NonInteger verifies that a non-integer teamId is rejected.
func (s *HandlerAppListSuite) Test_InvalidTeamId_NonInteger() {
	keyValue, _ := s.userKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/apps?teamId=abc",
		nil,
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

// Test_InvalidTeamId_Negative verifies that a negative teamId is rejected.
func (s *HandlerAppListSuite) Test_InvalidTeamId_Negative() {
	keyValue, _ := s.userKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/apps?teamId=-1",
		nil,
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

// Test_InvalidRepo_Format verifies that a repo without a valid vcs prefix is rejected.
func (s *HandlerAppListSuite) Test_InvalidRepo_Format() {
	keyValue, _ := s.userKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		"/v1/apps?repo=not-a-valid-repo",
		nil,
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

// Test_FilterByTeamId verifies that results are scoped to the given teamId.
func (s *HandlerAppListSuite) Test_FilterByTeamId() {
	usr := s.MockUser()
	a := s.MockApp(usr)
	// A second user with their own team and app — should not appear in results.
	usr2 := s.MockUser()
	s.MockApp(usr2)

	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr.ID,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
		"Scope":  apikey.SCOPE_USER,
	})

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/apps?teamId=%s", usr.DefaultTeamID.String()),
		nil,
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusOK, response.Code)

	var body map[string]any
	s.NoError(json.Unmarshal([]byte(response.String()), &body))

	apps, _ := body["apps"].([]any)
	s.Len(apps, 1)
	s.Equal(a.ID.String(), apps[0].(map[string]any)["id"])
}

// Test_Forbidden_WhenNotTeamMember verifies that a user cannot access apps scoped
// to a team they do not belong to.
func (s *HandlerAppListSuite) Test_Forbidden_WhenNotTeamMember() {
	usr := s.MockUser()
	// A second user — usr is not a member of this user's default team.
	otherUsr := s.MockUser()

	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr.ID,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
		"Scope":  apikey.SCOPE_USER,
	})

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/apps?teamId=%s", otherUsr.DefaultTeamID.String()),
		nil,
		map[string]string{
			"Authorization": key.Value,
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

// Test_FilterNoMatch verifies that non-matching filters return an empty list.
func (s *HandlerAppListSuite) Test_FilterNoMatch() {
	keyValue, a := s.userKey()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/apps?teamId=%s&repo=github/does-not/exist", a.TeamID.String()),
		nil,
		map[string]string{
			"Authorization": keyValue,
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{"apps": [], "hasNextPage": false}`, response.String())
}

func TestHandlerAppListSuite(t *testing.T) {
	suite.Run(t, new(HandlerAppListSuite))
}
