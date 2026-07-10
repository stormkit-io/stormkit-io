package maintenancehandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/maintenance"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/maintenance/maintenancehandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type HandlerConfigSetSuite struct {
	suite.Suite
	*factory.Factory
	conn             databasetest.TestDB
	mockCacheService *mocks.CacheInterface
}

func (s *HandlerConfigSetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockCacheService = &mocks.CacheInterface{}
	appcache.DefaultCacheService = s.mockCacheService
	admin.SetMockLicense()
}

func (s *HandlerConfigSetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	appcache.DefaultCacheService = nil
	admin.ResetMockLicense()
}

func (s *HandlerConfigSetSuite) Test_ConfigSet_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	s.mockCacheService.On("Reset", types.ID(env.ID)).Return(nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(maintenancehandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/maintenance/config",
		map[string]any{
			"envId":       env.ID.String(),
			"maintenance": "on",
		},
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	config, err := maintenance.Store().Config(context.Background(), env.ID)
	s.NoError(err)
	s.NotNil(config)
	s.Equal("on", config.Status)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.NoError(err)
	s.Len(audits, 1)
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "UPDATE:MAINTENANCE",
		EnvName:     env.Name,
		EnvID:       env.ID,
		AppID:       app.ID,
		TeamID:      app.TeamID,
		UserID:      usr.ID,
		UserDisplay: usr.Display(),
		Diff: &audit.Diff{
			Old: audit.DiffFields{
				MaintenanceStatus: "off",
			},
			New: audit.DiffFields{
				MaintenanceStatus: maintenance.StatusOn,
			},
		},
	}, audits[0])
}

func (s *HandlerConfigSetSuite) Test_ConfigSet_ResetSuccess() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	s.NoError(maintenance.Store().SetConfig(context.Background(), env.ID, &maintenance.Config{
		Status: "on",
	}))

	s.mockCacheService.On("Reset", types.ID(env.ID)).Return(nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(maintenancehandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/maintenance/config",
		map[string]any{
			"envId":       env.ID.String(),
			"maintenance": "",
		},
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	config, err := maintenance.Store().Config(context.Background(), env.ID)
	s.NoError(err)
	s.NotNil(config)
	s.Equal("", config.Status)
}

func (s *HandlerConfigSetSuite) Test_ConfigSet_InvalidOption() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(maintenancehandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/maintenance/config",
		map[string]any{
			"envId":       env.ID.String(),
			"maintenance": "not-allowed",
		},
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := `{ "error": "Invalid maintenance status. Available options are: on | ''" }`

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerConfigSetSuite(t *testing.T) {
	suite.Run(t, &HandlerConfigSetSuite{})
}
