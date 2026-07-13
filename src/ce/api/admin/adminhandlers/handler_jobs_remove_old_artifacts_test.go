package adminhandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerJobsRemoveOldArtifactsSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerJobsRemoveOldArtifactsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerJobsRemoveOldArtifactsSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerJobsRemoveOldArtifactsSuite) Test_Success() {
	usr := s.MockUser(map[string]any{"IsAdmin": true})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/jobs/remove-old-artifacts",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{"deleted": null}`, response.String()) // since we got no data
}

func (s *HandlerJobsRemoveOldArtifactsSuite) Test_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/jobs/remove-old-artifacts",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerJobsRemoveOldArtifactsSuite(t *testing.T) {
	suite.Run(t, &HandlerJobsRemoveOldArtifactsSuite{})
}
