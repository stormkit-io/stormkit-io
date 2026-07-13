package adminhandlers_test

import (
	"context"
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
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type HandlerRuntimesSuite struct {
	suite.Suite
	*factory.Factory

	conn    databasetest.TestDB
	mise    *mocks.MiseInterface
	service *mocks.MicroServiceInterface
}

func (s *HandlerRuntimesSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.service = &mocks.MicroServiceInterface{}
	s.mise = &mocks.MiseInterface{}
	mise.DefaultMise = s.mise
	rediscache.DefaultService = s.service
}

func (s *HandlerRuntimesSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
	mise.DefaultMise = nil
	rediscache.DefaultService = nil
}

func (s *HandlerRuntimesSuite) Test_Success() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})

	vc, err := admin.Store().Config(context.Background())
	s.NoError(err)
	s.NotNil(vc)

	vc.SystemConfig = nil

	s.service.On("Broadcast", rediscache.EventInvalidateAdminCache).Return(nil).Once()
	s.NoError(admin.Store().UpsertConfig(context.Background(), vc))

	type Case struct {
		SystemConfig     *admin.SystemConfig
		ExpectedResponse string
	}

	cases := []Case{
		{
			SystemConfig:     nil,
			ExpectedResponse: `{ "runtimes": [], "autoInstall": true, "status": "ok", "installed": null }`,
		},
		{
			SystemConfig: &admin.SystemConfig{
				Runtimes:    []string{"node@18", "go@latest"},
				AutoInstall: false,
			},
			ExpectedResponse: `{ "runtimes": ["node@18", "go@latest"], "autoInstall": false, "installed": null, "status": "ok" }`,
		},
		{
			SystemConfig: &admin.SystemConfig{
				Runtimes:    []string{"python@3.10"},
				AutoInstall: true,
			},
			ExpectedResponse: `{ "runtimes": ["python@3.10"], "autoInstall": true, "status": "ok", "installed": null }`,
		},
	}

	for _, c := range cases {
		vc.SystemConfig = c.SystemConfig

		s.mise.On("ListGlobal", mock.Anything).Return(nil, nil).Once()
		s.service.On("Broadcast", rediscache.EventInvalidateAdminCache).Return(nil).Once()
		s.service.On("GetAll", rediscache.KEY_RUNTIMES_STATUS, []string{"hosting", "workerserver"}).Return(nil, nil).Once()

		s.NoError(admin.Store().UpsertConfig(context.Background(), vc))

		response := shttptest.RequestWithHeaders(
			shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
			shttp.MethodGet,
			"/admin/system/runtimes",
			nil,
			map[string]string{
				"Authorization": usertest.Authorization(usr.ID),
			},
		)

		s.Equal(http.StatusOK, response.Code)
		s.JSONEq(c.ExpectedResponse, response.String())
	}
}

func (s *HandlerRuntimesSuite) Test_NonAdmin() {
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

func TestHandlerRuntimesSuite(t *testing.T) {
	suite.Run(t, &HandlerRuntimesSuite{})
}
