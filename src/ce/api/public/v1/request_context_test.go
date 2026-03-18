package publicapiv1_test

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type RequestContextSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *RequestContextSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *RequestContextSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *RequestContextSuite) invoke(method, target, body, contentType, token string, opts ...*publicapiv1.Opts) *shttp.Response {
	fn := publicapiv1.WithAPIKey(func(req *publicapiv1.RequestContext) *shttp.Response {
		return shttp.OK()
	}, opts...)

	r := httptest.NewRequest(method, target, strings.NewReader(body))
	r.Header.Set("Authorization", "Bearer "+token)

	if contentType != "" {
		r.Header.Set("Content-Type", contentType)
	}

	return fn(shttp.NewRequestContext(r))
}

func (s *RequestContextSuite) Test_ForbiddenOrNotFound() {
	usr := s.MockUser()
	appOnlyKey := s.MockAPIKey(nil, nil, map[string]any{"EnvID": types.ID(0)})
	userOnlyKey := s.MockAPIKey(nil, nil, map[string]any{"UserID": usr.ID, "AppID": types.ID(0), "EnvID": types.ID(0)})

	tests := []struct {
		name   string
		token  string
		status int
		opts   *publicapiv1.Opts
	}{
		{"empty token", "", http.StatusForbidden, nil},
		{"token without SK_ prefix", "not-a-valid-token", http.StatusForbidden, nil},
		{"unknown SK_ token", "SK_unknowntoken", http.StatusForbidden, nil},
		{"SCOPE_USER key has no UserID", appOnlyKey.Value, http.StatusForbidden, &publicapiv1.Opts{MinimumScope: apikey.SCOPE_USER}},
		{"SCOPE_TEAM key has no TeamID and no teamId in request", appOnlyKey.Value, http.StatusForbidden, &publicapiv1.Opts{MinimumScope: apikey.SCOPE_TEAM}},
		{"SCOPE_APP key has no AppID", userOnlyKey.Value, http.StatusNotFound, &publicapiv1.Opts{MinimumScope: apikey.SCOPE_APP}},
		{"SCOPE_ENV key has no EnvID and no envId in request", appOnlyKey.Value, http.StatusNotFound, &publicapiv1.Opts{MinimumScope: apikey.SCOPE_ENV}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			var opts []*publicapiv1.Opts
			if tt.opts != nil {
				opts = append(opts, tt.opts)
			}
			resp := s.invoke(http.MethodGet, "/", "", "", tt.token, opts...)
			s.Equal(tt.status, resp.Status)
		})
	}
}

func (s *RequestContextSuite) Test_Success_WithTokenScope() {
	usr := s.MockUser()
	envKey := s.MockAPIKey(nil, nil)
	appKey := s.MockAPIKey(nil, nil)
	userKey := s.MockAPIKey(nil, nil, map[string]any{"UserID": usr.ID, "AppID": types.ID(0), "EnvID": types.ID(0)})
	teamKey := s.MockAPIKey(nil, nil, map[string]any{"TeamID": usr.DefaultTeamID, "AppID": types.ID(0), "EnvID": types.ID(0)})

	tests := []struct {
		name  string
		token string
		opts  *publicapiv1.Opts
	}{
		{"no scope", envKey.Value, nil},
		{"SCOPE_ENV satisfied by key", envKey.Value, &publicapiv1.Opts{MinimumScope: apikey.SCOPE_ENV}},
		{"SCOPE_APP satisfied by key", appKey.Value, &publicapiv1.Opts{MinimumScope: apikey.SCOPE_APP}},
		{"SCOPE_USER satisfied by key", userKey.Value, &publicapiv1.Opts{MinimumScope: apikey.SCOPE_USER}},
		{"SCOPE_TEAM satisfied by key", teamKey.Value, &publicapiv1.Opts{MinimumScope: apikey.SCOPE_TEAM}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			resp := s.invoke(http.MethodGet, "/", "", "", tt.token, tt.opts)
			s.Equal(http.StatusOK, resp.Status)
		})
	}
}

func (s *RequestContextSuite) Test_Success_WithQueryParams() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	key := s.MockAPIKey(nil, nil, map[string]any{"UserID": usr.ID, "AppID": types.ID(0), "EnvID": types.ID(0)})

	tests := []struct {
		name   string
		target string
		opts   *publicapiv1.Opts
	}{
		{"SCOPE_ENV from envId query param", fmt.Sprintf("/?envId=%d", env.ID), &publicapiv1.Opts{MinimumScope: apikey.SCOPE_ENV}},
		{"SCOPE_APP from appId query param", fmt.Sprintf("/?appId=%d", app.ID), &publicapiv1.Opts{MinimumScope: apikey.SCOPE_APP}},
		{"SCOPE_TEAM from teamId query param", fmt.Sprintf("/?teamId=%d", usr.DefaultTeamID), &publicapiv1.Opts{MinimumScope: apikey.SCOPE_TEAM}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			resp := s.invoke(http.MethodGet, tt.target, "", "", key.Value, tt.opts)
			s.Equal(http.StatusOK, resp.Status)
		})
	}
}

func (s *RequestContextSuite) Test_Success_WithMultipartForm() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	key := s.MockAPIKey(nil, nil, map[string]any{"UserID": usr.ID, "AppID": types.ID(0), "EnvID": types.ID(0)})

	tests := []struct {
		name  string
		field string
		value string
		opts  *publicapiv1.Opts
	}{
		{"SCOPE_ENV from envId multipart field", "envId", fmt.Sprintf("%d", env.ID), &publicapiv1.Opts{MinimumScope: apikey.SCOPE_ENV}},
		{"SCOPE_APP from appId multipart field", "appId", fmt.Sprintf("%d", app.ID), &publicapiv1.Opts{MinimumScope: apikey.SCOPE_APP}},
		{"SCOPE_TEAM from teamId multipart field", "teamId", fmt.Sprintf("%d", usr.DefaultTeamID), &publicapiv1.Opts{MinimumScope: apikey.SCOPE_TEAM}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			fn := publicapiv1.WithAPIKey(func(req *publicapiv1.RequestContext) *shttp.Response {
				return shttp.OK()
			}, tt.opts)

			buf := &bytes.Buffer{}
			mw := multipart.NewWriter(buf)
			_ = mw.WriteField(tt.field, tt.value)
			_ = mw.Close()

			r := httptest.NewRequest(http.MethodPost, "/", buf)
			r.Header.Set("Authorization", "Bearer "+key.Value)
			r.Header.Set("Content-Type", mw.FormDataContentType())

			resp := fn(shttp.NewRequestContext(r))
			s.Equal(http.StatusOK, resp.Status)
		})
	}
}

func (s *RequestContextSuite) Test_Success_WithRequestBody() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	key := s.MockAPIKey(nil, nil, map[string]any{"UserID": usr.ID, "AppID": types.ID(0), "EnvID": types.ID(0)})

	tests := []struct {
		name string
		body string
		opts *publicapiv1.Opts
	}{
		{"SCOPE_ENV from envId body field", fmt.Sprintf(`{"envId":"%d"}`, env.ID), &publicapiv1.Opts{MinimumScope: apikey.SCOPE_ENV}},
		{"SCOPE_APP from appId body field", fmt.Sprintf(`{"appId":"%d"}`, app.ID), &publicapiv1.Opts{MinimumScope: apikey.SCOPE_APP}},
		{"SCOPE_TEAM from teamId body field", fmt.Sprintf(`{"teamId":"%d"}`, usr.DefaultTeamID), &publicapiv1.Opts{MinimumScope: apikey.SCOPE_TEAM}},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			resp := s.invoke(http.MethodPost, "/", tt.body, "application/json", key.Value, tt.opts)
			s.Equal(http.StatusOK, resp.Status)
		})
	}
}

func (s *RequestContextSuite) Test_License() {
	config.SetIsSelfHosted(false)

	usr := s.MockUser(map[string]any{"Metadata": user.UserMeta{
		SeatsPurchased: 10,
		PackageName:    config.PackageUltimate,
	}})
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	tkn := &apikey.Token{Name: "Default", EnvID: env.ID}

	// Should return the license from the environment
	req := &publicapiv1.RequestContext{
		RequestContext: &shttp.RequestContext{
			Request: &http.Request{},
		},
		Token: tkn,
	}

	licenseFromEnv := req.License()

	s.NotNil(licenseFromEnv)
	s.True(licenseFromEnv.Ultimate)
	s.Equal(10, licenseFromEnv.Seats)

	// Let's check based on AppID
	req.Token.EnvID = 0
	req.Token.AppID = app.ID

	licenseFromApp := req.License()
	s.Equal(licenseFromEnv, licenseFromApp)

	// Let's check based on TeamID
	req.Token.AppID = 0
	req.Token.TeamID = usr.DefaultTeamID

	licenseFromTeam := req.License()
	s.Equal(licenseFromEnv, licenseFromTeam)

	// Let's check based on user
	req.Token.TeamID = 0
	req.Token.UserID = usr.ID

	licenseFromUser := req.License()
	s.Equal(licenseFromEnv, licenseFromUser)

	// Let's mock the license to imitate self-hosted
	admin.SetMockLicense()
	licenseFromSelfHosted := req.License()

	s.NotNil(licenseFromSelfHosted)
	s.NotEqual(licenseFromTeam, licenseFromSelfHosted)

	admin.ResetMockLicense()
	config.SetIsSelfHosted(false)

	// Let's now test nil cases
	req.Token = nil
	s.Nil(req.License())
}

func (s *RequestContextSuite) Test_GetAuditData() {
	baseReq := shttp.NewRequestContext(httptest.NewRequest(http.MethodGet, "/", nil))
	appID := types.ID(10)
	appTeamID := types.ID(20)
	envID := types.ID(30)
	directTeamID := types.ID(40)

	tests := []struct {
		name     string
		req      *publicapiv1.RequestContext
		expected audit.AuditData
	}{
		{
			name: "empty request — only ctx set",
			req: &publicapiv1.RequestContext{
				RequestContext: baseReq,
			},
			expected: audit.AuditData{
				Ctx: baseReq.Context(),
			},
		},
		{
			name: "token name populated",
			req: &publicapiv1.RequestContext{
				RequestContext: baseReq,
				Token:          &apikey.Token{Name: "ci-token"},
			},
			expected: audit.AuditData{
				Ctx:       baseReq.Context(),
				TokenName: "ci-token",
			},
		},
		{
			name: "app ID and team ID populated",
			req: &publicapiv1.RequestContext{
				RequestContext: baseReq,
				App:            &app.App{ID: appID, TeamID: appTeamID},
			},
			expected: audit.AuditData{
				Ctx:    baseReq.Context(),
				AppID:  appID,
				TeamID: appTeamID,
			},
		},
		{
			name: "direct TeamID overrides app TeamID",
			req: &publicapiv1.RequestContext{
				RequestContext: baseReq,
				App:            &app.App{ID: appID, TeamID: appTeamID},
				TeamID:         directTeamID,
			},
			expected: audit.AuditData{
				Ctx:    baseReq.Context(),
				AppID:  appID,
				TeamID: directTeamID,
			},
		},
		{
			name: "env ID populated",
			req: &publicapiv1.RequestContext{
				RequestContext: baseReq,
				Env:            &buildconf.Env{ID: envID},
			},
			expected: audit.AuditData{
				Ctx:   baseReq.Context(),
				EnvID: envID,
			},
		},
		{
			name: "all fields set",
			req: &publicapiv1.RequestContext{
				RequestContext: baseReq,
				Token:          &apikey.Token{Name: "ci-token"},
				App:            &app.App{ID: appID, TeamID: appTeamID},
				Env:            &buildconf.Env{ID: envID, AppID: appID},
				TeamID:         directTeamID,
			},
			expected: audit.AuditData{
				Ctx:       baseReq.Context(),
				TokenName: "ci-token",
				AppID:     appID,
				TeamID:    directTeamID,
				EnvID:     envID,
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			s.Equal(tt.expected, tt.req.GetAuditData())
		})
	}
}

func TestRequestContext(t *testing.T) {
	suite.Run(t, new(RequestContextSuite))
}
