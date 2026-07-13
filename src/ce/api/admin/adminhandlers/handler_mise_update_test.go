package adminhandlers_test

import (
	"errors"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/mise"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type HandlerMiseUpdateSuite struct {
	suite.Suite
	*factory.Factory

	conn    databasetest.TestDB
	mise    *mocks.MiseInterface
	service *mocks.MicroServiceInterface
}

func (s *HandlerMiseUpdateSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mise = &mocks.MiseInterface{}
	s.service = &mocks.MicroServiceInterface{}
	mise.DefaultMise = s.mise
	rediscache.DefaultService = s.service
}

func (s *HandlerMiseUpdateSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	mise.DefaultMise = nil
	rediscache.DefaultService = nil
}

func (s *HandlerMiseUpdateSuite) Test_Success() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})
	services := []string{
		rediscache.ServiceHosting,
		rediscache.ServiceWorkerserver,
	}

	s.service.On("SetAll", "mise_update", rediscache.StatusSent, services).Return(nil).Once()
	s.service.On("Broadcast", rediscache.EventMiseUpdate).Return(nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/system/mise",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
}

func (s *HandlerMiseUpdateSuite) Test_Success_Abort() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})
	services := []string{
		rediscache.ServiceHosting,
		rediscache.ServiceWorkerserver,
	}

	s.service.On("DelAll", "mise_update", services).Return(nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/system/mise",
		map[string]any{
			"abort": true,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
}

func (s *HandlerMiseUpdateSuite) Test_BroadcastError() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})
	expectedError := errors.New("broadcast failed")

	s.service.On("SetAll", "mise_update", rediscache.StatusSent, []string{"hosting", "workerserver"}).Return(nil).Once()
	s.service.On("Broadcast", rediscache.EventMiseUpdate).Return(expectedError).Once()
	s.service.On("SetAll", "mise_update", rediscache.StatusErr, []string{"hosting", "workerserver"}).Return(nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/system/mise",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusInternalServerError, response.Code)
}

func (s *HandlerMiseUpdateSuite) Test_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/system/mise",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerMiseUpdateSuite(t *testing.T) {
	suite.Run(t, &HandlerMiseUpdateSuite{})
}
