package adminhandlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerSystemSettingsUpdateSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerSystemSettingsUpdateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerSystemSettingsUpdateSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerSystemSettingsUpdateSuite) Test_Update_Success() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/admin/system/settings",
		map[string]any{
			"artifactRetentionDays": 60,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	data := map[string]any{}

	s.Equal(http.StatusOK, resp.Code)
	s.NoError(json.Unmarshal(resp.Byte(), &data))
	s.Equal(float64(60), data["artifactRetentionDays"])
}

func (s *HandlerSystemSettingsUpdateSuite) Test_Update_InvalidRetentionDays() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/admin/system/settings",
		map[string]any{
			"artifactRetentionDays": 0,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	data := map[string]any{}

	s.Equal(http.StatusBadRequest, resp.Code)
	s.NoError(json.Unmarshal(resp.Byte(), &data))
	s.Contains(data, "error")
}

func (s *HandlerSystemSettingsUpdateSuite) Test_Update_Unauthorized_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/admin/system/settings",
		map[string]any{
			"artifactRetentionDays": 60,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, resp.Code)
}

func TestHandlerSystemSettingsUpdateSuite(t *testing.T) {
	suite.Run(t, &HandlerSystemSettingsUpdateSuite{})
}
