package publicapiv1_test

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type WithAPIKeySuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *WithAPIKeySuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *WithAPIKeySuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *WithAPIKeySuite) invoke(token string, opts ...*publicapiv1.Opts) *shttp.Response {
	fn := publicapiv1.WithAPIKey(func(req *publicapiv1.RequestContext) *shttp.Response {
		return shttp.OK()
	}, opts...)

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+token)

	return fn(shttp.NewRequestContext(r))
}

func (s *WithAPIKeySuite) invokeAndCapture(method, url, body, contentType, token string) *publicapiv1.RequestContext {
	var captured *publicapiv1.RequestContext

	fn := publicapiv1.WithAPIKey(func(req *publicapiv1.RequestContext) *shttp.Response {
		captured = req
		return shttp.OK()
	})

	var bodyReader *strings.Reader

	if body != "" {
		bodyReader = strings.NewReader(body)
	} else {
		bodyReader = strings.NewReader("")
	}

	r := httptest.NewRequest(method, url, bodyReader)
	r.Header.Set("Authorization", "Bearer "+token)

	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}

	fn(shttp.NewRequestContext(r))

	return captured
}

// Test_EmptyToken verifies that an empty Authorization header is rejected.
func (s *WithAPIKeySuite) Test_EmptyToken() {
	resp := s.invoke("")
	s.Equal(http.StatusForbidden, resp.Status)
}

// Test_NoSKPrefix verifies that a token without the "SK_" prefix is immediately rejected,
// without hitting the database.
func (s *WithAPIKeySuite) Test_NoSKPrefix() {
	resp := s.invoke("not-a-valid-token")
	s.Equal(http.StatusForbidden, resp.Status)
}

// Test_KeyNotFound verifies that a properly prefixed token that doesn't exist in the
// database is rejected.
func (s *WithAPIKeySuite) Test_KeyNotFound() {
	resp := s.invoke("SK_doesnotexist")
	s.Equal(http.StatusForbidden, resp.Status)
}

// Test_KeyWithNoIDs verifies that a token with no associated user, app, or team is rejected.
func (s *WithAPIKeySuite) Test_KeyWithNoIDs() {
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": types.ID(0),
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
		"Scope":  apikey.SCOPE_APP,
	})

	resp := s.invoke(key.Value)
	s.Equal(http.StatusForbidden, resp.Status)
}

// Test_ScopeUser_WithoutUserID verifies that an app-scoped key is rejected for a
// SCOPE_USER endpoint.
func (s *WithAPIKeySuite) Test_ScopeUser_WithoutUserID() {
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": types.ID(0),
		"TeamID": types.ID(0),
		"Scope":  apikey.SCOPE_APP,
	})

	resp := s.invoke(key.Value, &publicapiv1.Opts{MinimumScope: apikey.SCOPE_USER})
	s.Equal(http.StatusForbidden, resp.Status)
}

// Test_ScopeUser_WithUserID verifies that a user-scoped key passes a SCOPE_USER endpoint.
func (s *WithAPIKeySuite) Test_ScopeUser_WithUserID() {
	usr := s.MockUser()
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr.ID,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
		"Scope":  apikey.SCOPE_USER,
	})

	resp := s.invoke(key.Value, &publicapiv1.Opts{MinimumScope: apikey.SCOPE_USER})
	s.Equal(http.StatusOK, resp.Status)
}

// Test_ScopeTeam_WithoutTeamOrUserID verifies that an app-only key is rejected for
// a SCOPE_TEAM endpoint.
func (s *WithAPIKeySuite) Test_ScopeTeam_WithoutTeamOrUserID() {
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": types.ID(0),
		"TeamID": types.ID(0),
		"Scope":  apikey.SCOPE_APP,
	})

	resp := s.invoke(key.Value, &publicapiv1.Opts{MinimumScope: apikey.SCOPE_TEAM})
	s.Equal(http.StatusForbidden, resp.Status)
}

// Test_ScopeTeam_WithUserID verifies that a user-scoped key satisfies a SCOPE_TEAM endpoint.
func (s *WithAPIKeySuite) Test_ScopeTeam_WithUserID() {
	usr := s.MockUser()
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr.ID,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
		"Scope":  apikey.SCOPE_USER,
	})

	resp := s.invoke(key.Value, &publicapiv1.Opts{MinimumScope: apikey.SCOPE_TEAM})
	s.Equal(http.StatusOK, resp.Status)
}

// Test_ScopeApp_Passes verifies that an app-scoped key passes a SCOPE_APP endpoint.
func (s *WithAPIKeySuite) Test_ScopeApp_Passes() {
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": types.ID(0),
		"TeamID": types.ID(0),
		"Scope":  apikey.SCOPE_APP,
	})

	resp := s.invoke(key.Value, &publicapiv1.Opts{MinimumScope: apikey.SCOPE_APP})
	s.Equal(http.StatusOK, resp.Status)
}

// Test_ScopeEnv_Passes verifies that an env-scoped key passes a SCOPE_ENV endpoint.
func (s *WithAPIKeySuite) Test_ScopeEnv_Passes() {
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": types.ID(0),
		"TeamID": types.ID(0),
		"Scope":  apikey.SCOPE_ENV,
	})

	resp := s.invoke(key.Value, &publicapiv1.Opts{MinimumScope: apikey.SCOPE_ENV})
	s.Equal(http.StatusOK, resp.Status)
}

// Test_TokenSetOnContext verifies that the resolved API token is attached to the
// RequestContext and forwarded to the handler.
func (s *WithAPIKeySuite) Test_TokenSetOnContext() {
	usr := s.MockUser()
	key := s.MockAPIKey(nil, nil, map[string]any{
		"UserID": usr.ID,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"TeamID": types.ID(0),
		"Scope":  apikey.SCOPE_USER,
	})

	var capturedToken *apikey.Token

	fn := publicapiv1.WithAPIKey(func(req *publicapiv1.RequestContext) *shttp.Response {
		capturedToken = req.Token
		return shttp.OK()
	})

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.Header.Set("Authorization", "Bearer "+key.Value)

	resp := fn(shttp.NewRequestContext(r))

	s.Equal(http.StatusOK, resp.Status)
	s.NotNil(capturedToken)
	s.Equal(key.ID, capturedToken.ID)
}

// Test_IDResolution_GET_FromQueryParams verifies that for GET requests the EnvID and AppID
// are populated from query parameters when the token carries no IDs of its own.
func (s *WithAPIKeySuite) Test_IDResolution_GET_FromQueryParams() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env, map[string]any{
		"EnvID":  types.ID(0),
		"AppID":  types.ID(0),
		"Scope":  apikey.SCOPE_USER,
		"UserID": usr.ID,
	})

	url := fmt.Sprintf("/?envId=%s&appId=%s", env.ID, appl.ID)
	ctx := s.invokeAndCapture(http.MethodGet, url, "", "", key.Value)

	s.NotNil(ctx)
	s.Equal(env.ID, ctx.EnvID)
	s.Equal(appl.ID, ctx.AppID)
}

// Test_IDResolution_DELETE_FromQueryParams verifies that DELETE requests also read
// EnvID and AppID from query parameters.
func (s *WithAPIKeySuite) Test_IDResolution_DELETE_FromQueryParams() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env, map[string]any{
		"EnvID":  types.ID(0),
		"AppID":  types.ID(0),
		"Scope":  apikey.SCOPE_USER,
		"UserID": usr.ID,
	})

	url := fmt.Sprintf("/?envId=%s&appId=%s", env.ID, appl.ID)
	ctx := s.invokeAndCapture(http.MethodDelete, url, "", "", key.Value)

	s.NotNil(ctx)
	s.Equal(env.ID, ctx.EnvID)
	s.Equal(appl.ID, ctx.AppID)
}

// Test_IDResolution_POST_FromJSONBody verifies that for POST requests the EnvID and AppID
// are parsed from the JSON request body when the token carries no IDs.
func (s *WithAPIKeySuite) Test_IDResolution_POST_FromJSONBody() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env, map[string]any{
		"EnvID":  types.ID(0),
		"AppID":  types.ID(0),
		"Scope":  apikey.SCOPE_USER,
		"UserID": usr.ID,
	})

	body := fmt.Sprintf(`{"envId":"%s","appId":"%s"}`, env.ID, appl.ID)
	ctx := s.invokeAndCapture(http.MethodPost, "/", body, "application/json", key.Value)

	s.NotNil(ctx)
	s.Equal(env.ID, ctx.EnvID)
	s.Equal(appl.ID, ctx.AppID)
}

// Test_IDResolution_POST_Multipart_FromFormValues verifies that multipart POST requests
// read EnvID and AppID from form fields rather than the JSON body.
func (s *WithAPIKeySuite) Test_IDResolution_POST_Multipart_FromFormValues() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env, map[string]any{
		"EnvID":  types.ID(0),
		"AppID":  types.ID(0),
		"Scope":  apikey.SCOPE_USER,
		"UserID": usr.ID,
	})

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("envId", env.ID.String())
	_ = w.WriteField("appId", appl.ID.String())
	w.Close()

	ctx := s.invokeAndCapture(http.MethodPost, "/", buf.String(), w.FormDataContentType(), key.Value)

	s.NotNil(ctx)
	s.Equal(env.ID, ctx.EnvID)
	s.Equal(appl.ID, ctx.AppID)
}

// Test_IDResolution_TokenIDTakesPriority verifies that the IDs embedded in the token
// take precedence over IDs supplied in query parameters.
func (s *WithAPIKeySuite) Test_IDResolution_TokenIDTakesPriority() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.MockAPIKey(appl, env) // token carries env.ID and appl.ID

	url := fmt.Sprintf("/?envId=9999&appId=9999")
	ctx := s.invokeAndCapture(http.MethodGet, url, "", "", key.Value)

	s.NotNil(ctx)
	s.Equal(env.ID, ctx.EnvID)
	s.Equal(appl.ID, ctx.AppID)
}

func TestWithAPIKey(t *testing.T) {
	suite.Run(t, new(WithAPIKeySuite))
}
