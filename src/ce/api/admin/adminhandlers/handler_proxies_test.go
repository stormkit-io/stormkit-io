package adminhandlers_test

import (
	"context"
	"encoding/json"
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

type HandlerProxiesSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerProxiesSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerProxiesSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerProxiesSuite) Test_Get_Success() {
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

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/system/proxies",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, resp.Code)

	var payload struct {
		Proxies admin.ProxyConfig `json:"proxies"`
	}

	s.NoError(json.Unmarshal([]byte(resp.String()), &payload))
	s.Equal(vc.ProxyConfig.Rules, payload.Proxies.Rules)
}

func (s *HandlerProxiesSuite) Test_Get_Unauthorized_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/system/proxies",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, resp.Code)
}

func TestHandlerProxiesSuite(t *testing.T) {
	suite.Run(t, &HandlerProxiesSuite{})
}
