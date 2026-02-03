package app_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type AppSHTTPSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *AppSHTTPSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *AppSHTTPSuite) AfterTest(_, _ string) {
	admin.ResetMockLicense()
	s.conn.CloseTx()
}

func (s *AppSHTTPSuite) MockLicense() *admin.License {
	return &admin.License{
		Seats:   10,
		Key:     "abcd-efgh-1234-defg-5829-bnac-00",
		Version: admin.LicenseVersion20240610,
	}
}

func (s *AppSHTTPSuite) Test_WithAPIKey_AdminScopeFail() {
	key := s.MockAPIKey(nil, nil)

	fn := app.WithAPIKey(func(rc *app.RequestContext) *shttp.Response {
		return &shttp.Response{
			Status: http.StatusOK,
			Data: map[string]string{
				"appId": rc.App.ID.String(),
			},
		}
	}, &app.Opts{Admin: true})

	res := fn(&shttp.RequestContext{
		Request: &http.Request{
			Header: http.Header{
				"Authorization": []string{key.Value},
			},
		},
	})

	s.True(int64(0) < int64(key.ID))
	s.Equal(http.StatusForbidden, res.Status)
}

func (s *AppSHTTPSuite) Test_WithAPIKey_AdminScopeSuccess() {
	key := s.MockAPIKey(nil, nil)

	s.conn.Exec("UPDATE users SET is_admin = TRUE WHERE user_id = $1", key.UserID)

	fn := app.WithAPIKey(func(rc *app.RequestContext) *shttp.Response {
		return &shttp.Response{
			Status: http.StatusOK,
			Data: map[string]string{
				"appId": rc.App.ID.String(),
			},
		}
	}, &app.Opts{Admin: true})

	res := fn(&shttp.RequestContext{
		Request: &http.Request{
			Header: http.Header{
				"Authorization": []string{key.Value},
			},
		},
	})

	s.True(int64(0) < int64(key.ID))
	s.Equal(http.StatusOK, res.Status)
	s.Equal(key.AppID.String(), (res.Data.(map[string]string))["appId"])
}

func (s *AppSHTTPSuite) Test_WithAPIKey_AppAPIKey() {
	key := s.MockAPIKey(nil, nil)

	fn := app.WithAPIKey(func(rc *app.RequestContext) *shttp.Response {
		return &shttp.Response{
			Status: http.StatusOK,
			Data: map[string]string{
				"appId": rc.App.ID.String(),
			},
		}
	})

	res := fn(&shttp.RequestContext{
		Request: &http.Request{
			Header: http.Header{
				"Authorization": []string{key.Value},
			},
		},
	})

	s.True(int64(0) < int64(key.ID))
	s.Equal(http.StatusOK, res.Status)
	s.Equal(key.AppID.String(), (res.Data.(map[string]string))["appId"])
}

func (s *AppSHTTPSuite) Test_WithAPIKey_UserAuth() {
	usr := s.MockUser()
	appl := s.MockApp(usr)

	fn := app.WithAPIKey(func(rc *app.RequestContext) *shttp.Response {
		return &shttp.Response{
			Status: http.StatusOK,
			Data: map[string]string{
				"appId": rc.App.ID.String(),
				"envId": rc.EnvID.String(),
			},
		}
	})

	jsonData := fmt.Sprintf(`{"appId":"%s","envId":"1"}`, appl.ID.String())

	res := fn(&shttp.RequestContext{
		Request: &http.Request{
			Body: io.NopCloser(strings.NewReader(jsonData)),
			Header: http.Header{
				"Authorization": []string{usertest.Authorization(usr.ID)},
			},
		},
	})

	s.Equal(http.StatusOK, res.Status)
	s.Equal(appl.ID.String(), (res.Data.(map[string]string))["appId"])
	s.Equal("1", (res.Data.(map[string]string))["envId"])
}

func (s *AppSHTTPSuite) Test_WithAPIKey_APIKeyInvalid() {
	appl := s.MockApp(nil)
	s.MockEnv(appl)

	fn := app.WithAPIKey(func(rc *app.RequestContext) *shttp.Response {
		return &shttp.Response{
			Status: http.StatusOK,
			Data: map[string]string{
				"appId": rc.App.ID.String(),
			},
		}
	})

	res := fn(&shttp.RequestContext{
		Request: &http.Request{
			Header: http.Header{
				"Authorization": []string{"SK_some-random-token"},
			},
		},
	})

	s.Equal(http.StatusForbidden, res.Status)
}

func (s *AppSHTTPSuite) Test_WithAPIKey_UserAuthFailed() {
	appl := s.MockApp(nil)
	s.MockEnv(appl)

	fn := app.WithAPIKey(func(rc *app.RequestContext) *shttp.Response {
		return &shttp.Response{
			Status: http.StatusOK,
			Data: map[string]string{
				"appId": rc.App.ID.String(),
			},
		}
	})

	res := fn(&shttp.RequestContext{
		Request: &http.Request{
			Header: http.Header{
				"Authorization": []string{"user-auth"},
			},
		},
	})

	s.Equal(http.StatusUnauthorized, res.Status)
}

func (s *AppSHTTPSuite) Test_WithAPIKey_TeamAPIKey_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	key := s.MockAPIKey(nil, nil, map[string]any{
		"TeamID": usr.DefaultTeamID,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"UserID": types.ID(0),
		"Scope":  apikey.SCOPE_TEAM,
	})

	fn := app.WithAPIKey(func(rc *app.RequestContext) *shttp.Response {
		return &shttp.Response{
			Status: http.StatusOK,
			Data: map[string]string{
				"appId": rc.App.ID.String(),
			},
		}
	})

	body, err := json.Marshal(map[string]string{
		"appId": appl.ID.String(),
	})

	s.NoError(err)

	res := fn(&shttp.RequestContext{
		Request: &http.Request{
			Header: http.Header{
				"Content-Type":  []string{"application/json"},
				"Authorization": []string{key.Value},
			},
			Body: io.NopCloser(bytes.NewReader(body)),
		},
	})

	s.True(int64(0) < int64(key.ID))
	s.Equal(http.StatusOK, res.Status)
	s.Equal(appl.ID.String(), (res.Data.(map[string]string))["appId"])
}

func (s *AppSHTTPSuite) Test_WithAPIKey_TeamAPIKey_ErrPermission() {
	usr := s.MockUser()
	usr2 := s.MockUser()
	appl := s.MockApp(usr2)
	key := s.MockAPIKey(nil, nil, map[string]any{
		"TeamID": usr.DefaultTeamID,
		"AppID":  types.ID(0),
		"EnvID":  types.ID(0),
		"UserID": types.ID(0),
		"Scope":  apikey.SCOPE_TEAM,
	})

	fn := app.WithAPIKey(func(rc *app.RequestContext) *shttp.Response {
		return &shttp.Response{
			Status: http.StatusOK,
			Data: map[string]string{
				"appId": rc.App.ID.String(),
			},
		}
	})

	body, err := json.Marshal(map[string]string{
		"appId": appl.ID.String(),
	})

	s.NoError(err)

	res := fn(&shttp.RequestContext{
		Request: &http.Request{
			Header: http.Header{
				"Content-Type":  []string{"application/json"},
				"Authorization": []string{key.Value},
			},
			Body: io.NopCloser(bytes.NewReader(body)),
		},
	})

	s.True(int64(0) < int64(key.ID))
	s.Equal(http.StatusForbidden, res.Status)
}

func TestAppSHTTP(t *testing.T) {
	suite.Run(t, &AppSHTTPSuite{})
}
