package deployhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/mocks"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy/deployhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type HandlerDeployDeleteSuite struct {
	suite.Suite
	*factory.Factory

	conn      databasetest.TestDB
	mockCache *mocks.CacheInterface
}

func (s *HandlerDeployDeleteSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockCache = &mocks.CacheInterface{}
	appcache.DefaultCacheService = s.mockCache
}

func (s *HandlerDeployDeleteSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	appcache.DefaultCacheService = nil
}

func (s *HandlerDeployDeleteSuite) Test_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	depl := s.MockDeployment(env)

	s.mockCache.On("Reset", env.ID).Return(nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		"/app/deploy",
		map[string]any{
			"appId":        appl.ID.String(),
			"deploymentId": depl.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	a := assert.New(s.T())
	a.Equal(http.StatusOK, response.Code)

	d, err := deploy.NewStore().DeploymentByID(context.Background(), depl.ID)
	a.Nil(d)
	a.NoError(err)
}

func (s *HandlerDeployDeleteSuite) Test_Fail204() {
	usr := s.MockUser()
	appl := s.MockApp(usr)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(deployhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		"/app/deploy",
		map[string]interface{}{
			"appId":        appl.ID.String(),
			"deploymentId": appl.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	a := assert.New(s.T())
	a.Equal(http.StatusNoContent, response.Code)

	d, err := deploy.NewStore().DeploymentByID(context.Background(), types.ID(1))
	a.Nil(d)
	a.NoError(err)
}

func TestHandlerDeployDelete(t *testing.T) {
	suite.Run(t, &HandlerDeployDeleteSuite{})
}
