package apphandlers_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

// TODO: write more test here. It requires refactoring with interfaces to Github Type
type HandleOneClickDeploySuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandleOneClickDeploySuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandleOneClickDeploySuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandleOneClickDeploySuite) Test_UnauthorizedUser() {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/deploy?template=https%3A%2F%2Fgithub.com%2Fstormkit-io%2Fmonorepo-template-react",
		nil,
	)

	s.Equal(http.StatusFound, response.Result().StatusCode)
}

func TestHandleOneClickDeployInsert(t *testing.T) {
	suite.Run(t, &HandleOneClickDeploySuite{})
}
