package adminhandlers_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerImpersonateSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerImpersonateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	config.SetIsStormkitCloud(true)
}

func (s *HandlerImpersonateSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	config.SetIsStormkitCloud(false)
}

func (s *HandlerImpersonateSuite) Test_Success() {
	admin := s.MockUser(map[string]any{"IsAdmin": true})
	targetUser := s.MockUser(map[string]any{"IsAdmin": false})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/cloud/impersonate",
		map[string]any{
			"userId": targetUser.ID,
		},
		map[string]string{
			"Authorization": usertest.Authorization(admin.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	var responseData map[string]any
	err := json.Unmarshal(response.Body.Bytes(), &responseData)
	s.NoError(err)

	s.Contains(responseData, "token")
	s.NotEmpty(responseData["token"])

	// Verify the token is valid and contains the correct user ID
	tokenString, ok := responseData["token"].(string)
	s.True(ok, "Token should be a string")

	claims := user.ParseJWT(&user.ParseJWTArgs{
		Bearer: tokenString,
	})

	s.Equal(targetUser.ID.String(), claims["uid"])
}

func (s *HandlerImpersonateSuite) Test_NonAdmin() {
	nonAdmin := s.MockUser(map[string]any{"IsAdmin": false})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/cloud/impersonate",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(nonAdmin.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerImpersonateSuite) Test_MissingUserID() {
	admin := s.MockUser(map[string]any{"IsAdmin": true})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/cloud/impersonate",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(admin.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"userId is required"}`, response.Body.String())
}

func (s *HandlerImpersonateSuite) Test_ImpersonateAdmin() {
	admin1 := s.MockUser(map[string]any{"IsAdmin": true})
	admin2 := s.MockUser(map[string]any{"IsAdmin": true})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/cloud/impersonate",
		map[string]any{
			"userId": admin2.ID,
		},
		map[string]string{
			"Authorization": usertest.Authorization(admin1.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerImpersonateSuite(t *testing.T) {
	suite.Run(t, &HandlerImpersonateSuite{})
}
