package adminhandlers_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/mise"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type HandlerMiseSuite struct {
	suite.Suite
	*factory.Factory

	mise *mocks.MiseInterface
	conn databasetest.TestDB
}

func (s *HandlerMiseSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mise = &mocks.MiseInterface{}
	mise.DefaultMise = s.mise
}

func (s *HandlerMiseSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	mise.DefaultMise = nil
}

func (s *HandlerMiseSuite) Test_Success() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})
	expectedVersion := "v2024.1.0"

	s.mise.On("Version").Return(expectedVersion, nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/system/mise",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	var data map[string]any
	err := json.Unmarshal(response.Body.Bytes(), &data)
	s.NoError(err)
	s.Equal(expectedVersion, data["version"])
}

func (s *HandlerMiseSuite) Test_VersionError() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})
	expectedError := errors.New("mise not found")

	s.mise.On("Version").Return("", expectedError).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/system/mise",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusInternalServerError, response.Code)
}

func (s *HandlerMiseSuite) Test_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/system/mise",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerMiseSuite(t *testing.T) {
	suite.Run(t, &HandlerMiseSuite{})
}
