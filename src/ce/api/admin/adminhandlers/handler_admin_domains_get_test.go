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

type HandlerAdminDomainsGetSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerAdminDomainsGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	config.SetIsStormkitCloud(true)
}

func (s *HandlerAdminDomainsGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	config.SetIsStormkitCloud(false)
}

func (s *HandlerAdminDomainsGetSuite) Test_Get_Success() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/domains",
		nil,
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
	s.Contains(domains, "dev")
	s.Contains(domains, "app")
	s.Contains(domains, "api")
}

func (s *HandlerAdminDomainsGetSuite) Test_Get_Unauthorized_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/domains",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, resp.Code)
}

func TestHandlerAdminDomainsGetSuite(t *testing.T) {
	suite.Run(t, &HandlerAdminDomainsGetSuite{})
}
