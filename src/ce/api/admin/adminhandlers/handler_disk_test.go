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
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/mocks"
)

type HandlerDiskSuite struct {
	suite.Suite
	*factory.Factory
	conn        databasetest.TestDB
	mockService *mocks.MicroServiceInterface
}

func (s *HandlerDiskSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockService = &mocks.MicroServiceInterface{}
	rediscache.DefaultService = s.mockService
}

func (s *HandlerDiskSuite) AfterTest(_, _ string) {
	rediscache.DefaultService = nil
	s.conn.CloseTx()
}

func (s *HandlerDiskSuite) request(method, path string) shttptest.Response {
	usr := s.MockUser(map[string]any{"IsAdmin": true})

	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		method,
		path,
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)
}

func (s *HandlerDiskSuite) Test_Get_Success() {
	services := []string{rediscache.ServiceHosting, rediscache.ServiceWorkerserver}

	s.mockService.On("List", services).Return([]*rediscache.MicroService{
		{ID: "host-1", Name: rediscache.ServiceHosting},
	}, nil).Once()

	s.mockService.On("GetAll", rediscache.KEY_DISK_USAGE, services).Return(map[string]string{
		"host-1": `{"root":{"path":"/","totalBytes":100,"freeBytes":25,"usedBytes":75,"usedPercent":75}}`,
	}, nil).Once()

	resp := s.request(shttp.MethodGet, "/admin/system/disk")

	data := map[string]any{}

	s.Equal(http.StatusOK, resp.Code)
	s.NoError(json.Unmarshal(resp.Byte(), &data))

	services_, ok := data["services"].([]any)
	s.Require().True(ok)
	s.Require().Len(services_, 1)

	first := services_[0].(map[string]any)
	s.Equal("hosting", first["serviceName"])
	s.Equal(float64(75), first["root"].(map[string]any)["usedPercent"])
}

// Nothing registered yet must render as an empty list, not null: the UI
// iterates over it.
func (s *HandlerDiskSuite) Test_Get_NoServices() {
	services := []string{rediscache.ServiceHosting, rediscache.ServiceWorkerserver}

	s.mockService.On("List", services).Return([]*rediscache.MicroService{}, nil).Once()
	s.mockService.On("GetAll", rediscache.KEY_DISK_USAGE, services).Return(map[string]string{}, nil).Once()

	resp := s.request(shttp.MethodGet, "/admin/system/disk")

	data := map[string]any{}

	s.Equal(http.StatusOK, resp.Code)
	s.NoError(json.Unmarshal(resp.Byte(), &data))
	s.Equal([]any{}, data["services"])
}

func (s *HandlerDiskSuite) Test_Get_Unauthorized_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/system/disk",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, resp.Code)
}

func (s *HandlerDiskSuite) Test_Cleanup_Broadcasts() {
	s.mockService.On("Broadcast", rediscache.EventDiskCleanup).Return(nil).Once()

	resp := s.request(shttp.MethodPost, "/admin/system/disk/cleanup")

	s.Equal(http.StatusOK, resp.Code)
	s.mockService.AssertExpectations(s.T())
}

func (s *HandlerDiskSuite) Test_Cleanup_Unauthorized_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/system/disk/cleanup",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, resp.Code)
	s.mockService.AssertNotCalled(s.T(), "Broadcast", rediscache.EventDiskCleanup)
}

func TestHandlerDiskSuite(t *testing.T) {
	suite.Run(t, new(HandlerDiskSuite))
}
