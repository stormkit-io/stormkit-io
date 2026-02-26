package adminhandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/mocks"
)

type HandlerAdminAppDeleteSuite struct {
	suite.Suite
	*factory.Factory

	conn             databasetest.TestDB
	app              *factory.MockApp
	env              *factory.MockEnv
	admin            *factory.MockUser
	mockCacheService *mocks.CacheInterface
	mockRequest      *mocks.RequestInterface
}

func (s *HandlerAdminAppDeleteSuite) SetupSuite() {
	s.conn = databasetest.InitTx("handler_admin_app_delete_suite")
	s.Factory = factory.New(s.conn)
}

func (s *HandlerAdminAppDeleteSuite) BeforeTest(_, _ string) {
	s.mockCacheService = &mocks.CacheInterface{}
	s.mockRequest = &mocks.RequestInterface{}

	s.admin = s.MockUser(map[string]any{"IsAdmin": true})
	s.app = s.MockApp(s.admin)
	s.env = s.MockEnv(s.app)

	shttp.DefaultRequest = s.mockRequest
	appcache.DefaultCacheService = s.mockCacheService
	config.SetIsStormkitCloud(true)
	config.Get().Reporting.DiscordProductionChannel = "https://my-discord.endpoint.com"
}

func (s *HandlerAdminAppDeleteSuite) TearDownSuite() {
	s.conn.CloseTx()
	appcache.DefaultCacheService = nil
	shttp.DefaultRequest = nil
	config.SetIsStormkitCloud(false)
	config.Get().Reporting.DiscordProductionChannel = ""
}

func (s *HandlerAdminAppDeleteSuite) Test_DeleteAppAndUser_Success() {
	headers := make(http.Header)
	headers.Add("Content-Type", "application/json")

	s.mockCacheService.On("Reset", s.env.ID).Return(nil).Once()
	s.mockRequest.On("SetOpts").Return(s.mockRequest).Once()
	s.mockRequest.On("URL", config.Get().Reporting.DiscordProductionChannel).Return(s.mockRequest).Once()
	s.mockRequest.On("Method", http.MethodPost).Return(s.mockRequest).Once()
	s.mockRequest.On("Headers", headers).Return(s.mockRequest).Once()
	s.mockRequest.On("Payload", mock.Anything).Return(s.mockRequest).Once()
	s.mockRequest.On("Do").Return(nil, nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/admin/cloud/app?appId=%s", s.app.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.admin.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	a, err := app.NewStore().AppByID(context.Background(), s.app.ID)
	s.NoError(err)
	s.Nil(a)

	u, err := user.NewStore().UserByID(s.admin.ID)
	s.NoError(err)
	s.Nil(u)
}

func (s *HandlerAdminAppGetSuite) Test_Unauthorized_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	resp := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		"/admin/cloud/app?appId=4",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, resp.Code)
}

func TestHandlerAdminAppDeleteSuite(t *testing.T) {
	suite.Run(t, &HandlerAdminAppDeleteSuite{})
}
