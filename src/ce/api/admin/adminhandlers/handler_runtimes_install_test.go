package adminhandlers_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
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

type HandlerRuntimesInstallSuite struct {
	suite.Suite
	*factory.Factory

	conn    databasetest.TestDB
	mise    *mocks.MiseInterface
	service *mocks.MicroServiceInterface
}

func (s *HandlerRuntimesInstallSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mise = &mocks.MiseInterface{}
	s.service = &mocks.MicroServiceInterface{}
	mise.DefaultMise = s.mise
	rediscache.DefaultService = s.service
}

func (s *HandlerRuntimesInstallSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	mise.DefaultMise = nil
	rediscache.DefaultService = nil
}

func (s *HandlerRuntimesInstallSuite) Test_Success() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})
	services := []string{
		rediscache.ServiceHosting,
		rediscache.ServiceWorkerserver,
	}

	s.service.On("SetAll", rediscache.KEY_RUNTIMES_STATUS, rediscache.StatusSent, services).Return(nil).Once()
	s.service.On("Broadcast", rediscache.EventRuntimesInstall).Return(nil).Once()
	s.service.On("Broadcast", rediscache.EventInvalidateAdminCache).Return(nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/system/runtimes",
		map[string]any{
			"runtimes":    []string{"node@18", "go@latest"},
			"autoInstall": true,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	vc, err := admin.Store().Config(context.Background())
	s.NoError(err)
	s.Equal([]string{"node@18", "go@latest"}, vc.SystemConfig.Runtimes)
	s.True(vc.SystemConfig.AutoInstall)
}

func (s *HandlerRuntimesInstallSuite) Test_BroadcastError() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})
	expectedError := errors.New("broadcast failed")

	s.service.On("SetAll", "runtimes_status", rediscache.StatusSent, []string{"hosting", "workerserver"}).Return(nil).Once()
	s.service.On("Broadcast", rediscache.EventRuntimesInstall).Return(expectedError).Once()
	s.service.On("Broadcast", rediscache.EventInvalidateAdminCache).Return(nil).Once()
	s.service.On("SetAll", "runtimes_status", rediscache.StatusErr, []string{"hosting", "workerserver"}).Return(nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/system/runtimes",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusInternalServerError, response.Code)
}

func (s *HandlerRuntimesInstallSuite) Test_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/system/runtimes",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerRuntimesInstallSuite(t *testing.T) {
	suite.Run(t, &HandlerRuntimesInstallSuite{})
}
