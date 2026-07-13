package adminhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerProxiesUpdateSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerProxiesUpdateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerProxiesUpdateSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerProxiesUpdateSuite) Test_Update_Success() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})

	vc, err := admin.Store().Config(context.Background())
	s.NoError(err)

	vc.ProxyConfig = &admin.ProxyConfig{
		Rules: map[string]*admin.ProxyRule{
			"api.example.org": {
				Target: "https://backend.internal/api",
				Headers: map[string]string{
					"X-Forwarded-Host": "api.example.org",
				},
			},
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.Background(), vc))

	payload := map[string]any{
		"proxies": map[string]any{
			"a.example.org": map[string]any{
				"target": "https://t",
				"headers": map[string]string{
					"X-Forwarded-Host": "a.example.org",
				},
			},
			"api.example.org": map[string]any{
				"target": "https://backend.internal/updated",
			},
		},
	}

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/admin/system/proxies",
		payload,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, resp.Code)

	expected := `{
		"proxies": {
			"rules": {
				"api.example.org": {
					"target": "https://backend.internal/updated"
				},
				"a.example.org": {
					"target": "https://t",
					"headers": {
						"X-Forwarded-Host": "a.example.org"
					}
				}
			}
		}
	}`

	vc, err = admin.Store().Config(context.Background())
	s.NoError(err)
	s.NotNil(vc.ProxyConfig)
	s.JSONEq(expected, resp.String())
}

func (s *HandlerProxiesUpdateSuite) Test_Update_Removal_Success() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})

	vc, err := admin.Store().Config(context.Background())
	s.NoError(err)

	vc.ProxyConfig = &admin.ProxyConfig{
		Rules: map[string]*admin.ProxyRule{
			"api.example.org": {
				Target: "https://backend.internal/api",
				Headers: map[string]string{
					"X-Forwarded-Host": "api.example.org",
				},
			},
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.Background(), vc))

	payload := map[string]any{
		"remove": []string{"api.example.org"},
	}

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/admin/system/proxies",
		payload,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, resp.Code)

	expected := `{ "proxies": { "rules": { } } }`

	vc, err = admin.Store().Config(context.Background())
	s.NoError(err)
	s.NotNil(vc.ProxyConfig)
	s.JSONEq(expected, resp.String())
}

func (s *HandlerProxiesUpdateSuite) Test_Update_Unauthorized_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/admin/system/proxies",
		map[string]any{
			"proxies": map[string]any{"rules": map[string]string{"a.example.org": "https://t"}},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, resp.Code)
}

func TestHandlerProxiesUpdateSuite(t *testing.T) {
	suite.Run(t, &HandlerProxiesUpdateSuite{})
}
