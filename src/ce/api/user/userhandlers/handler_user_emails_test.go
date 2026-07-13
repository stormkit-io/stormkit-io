package userhandlers_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user/userhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerUserEmailsSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerUserEmailsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerUserEmailsSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerUserEmailsSuite) Test_Success() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/user/emails",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"emails": [{
			"address": "%s", 
			"verified": true,
			"primary": true
		}]
	}`, usr.PrimaryEmail())

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerUserEmailsSuite(t *testing.T) {
	suite.Run(t, &HandlerUserEmailsSuite{})
}
