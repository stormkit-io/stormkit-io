package adminhandlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerAdminDomainsSetSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerAdminDomainsSetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	config.SetIsStormkitCloud(true)
}

func (s *HandlerAdminDomainsSetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	config.SetIsStormkitCloud(false)
}

func (s *HandlerAdminDomainsSetSuite) Test_Set_Success() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})

	requestData := map[string]any{
		"dev": "https://dev.example.org",
		"app": "https://app.example.org",
		"api": "https://api.example.org",
	}

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/domains",
		requestData,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	data := map[string]any{}

	s.Equal(http.StatusOK, resp.Code)
	s.NoError(json.Unmarshal(resp.Byte(), &data))
	s.Contains(data, "domains")

	domains, ok := data["domains"].(map[string]any)
	s.True(ok, "domains should be a map")
	s.Equal("https://dev.example.org", domains["dev"])
	s.Equal("https://app.example.org", domains["app"])
	s.Equal("https://api.example.org", domains["api"])
}

func (s *HandlerAdminDomainsSetSuite) Test_Set_InvalidDevDomain() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})

	requestData := map[string]any{
		"dev": "invalid-domain",
		"app": "https://app.example.org",
		"api": "https://api.example.org",
	}

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/domains",
		requestData,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	data := map[string]any{}

	s.Equal(http.StatusBadRequest, resp.Code)
	s.NoError(json.Unmarshal(resp.Byte(), &data))
	s.Contains(data, "error")
	s.Equal("Dev domain is invalid", data["error"])
}

func (s *HandlerAdminDomainsSetSuite) Test_Set_InvalidAppDomain() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})

	requestData := map[string]any{
		"dev": "https://dev.example.org",
		"app": "invalid-domain",
		"api": "https://api.example.org",
	}

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/domains",
		requestData,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	data := map[string]any{}

	s.Equal(http.StatusBadRequest, resp.Code)
	s.NoError(json.Unmarshal(resp.Byte(), &data))
	s.Contains(data, "error")
	s.Equal("App domain is invalid", data["error"])
}

func (s *HandlerAdminDomainsSetSuite) Test_Set_InvalidAPIDomain() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})

	requestData := map[string]any{
		"dev": "https://dev.example.org",
		"app": "https://app.example.org",
		"api": "invalid-domain",
	}

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/domains",
		requestData,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	data := map[string]any{}

	s.Equal(http.StatusBadRequest, resp.Code)
	s.NoError(json.Unmarshal(resp.Byte(), &data))
	s.Contains(data, "error")
	s.Equal("API domain is invalid", data["error"])
}

func (s *HandlerAdminDomainsSetSuite) Test_Set_Unauthorized_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	requestData := map[string]any{
		"dev": "https://dev.example.org",
		"app": "https://app.example.org",
		"api": "https://api.example.org",
	}

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/domains",
		requestData,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, resp.Code)
}

func TestHandlerAdminDomainsSetSuite(t *testing.T) {
	suite.Run(t, &HandlerAdminDomainsSetSuite{})
}
