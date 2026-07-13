package apphandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/testutils"
	"github.com/stretchr/testify/suite"
)

type AppProxySuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *AppProxySuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *AppProxySuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *AppProxySuite) TestProxy() {
	appl := s.MockApp(nil)

	ms := testutils.MockServer()
	mr := testutils.MockResponse{Status: 402, Method: shttp.MethodHead}
	ms.NewResponse("/", &mr)

	defer ms.Close()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(apphandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/app/proxy",
		map[string]string{
			"url":   ms.URL(),
			"appId": appl.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(appl.UserID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{ "status": 402 }`, response.String())
}

func TestAppProxy(t *testing.T) {
	suite.Run(t, &AppProxySuite{})
}
