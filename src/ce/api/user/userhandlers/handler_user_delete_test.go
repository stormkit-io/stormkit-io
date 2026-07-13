package userhandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/userhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerUserDeleteSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerUserDeleteSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerUserDeleteSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerUserDeleteSuite) Test_Success() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		"/user",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	deletedUser, _ := user.NewStore().UserByID(usr.ID)
	s.Nil(deletedUser)
}

func TestHandlerUserDeleteSuite(t *testing.T) {
	suite.Run(t, &HandlerUserDeleteSuite{})
}
