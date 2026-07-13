package userhandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user/userhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type HandlerUpdatePersonalAccessTokenSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerUpdatePersonalAccessTokenSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerUpdatePersonalAccessTokenSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerUpdatePersonalAccessTokenSuite) Test_Update_Success() {
	usr := s.MockUser()

	myToken := "my-new-personal-access-token"

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/user/access-token",
		map[string]string{
			"token": myToken,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	var token []byte
	var query = "SELECT personal_access_token FROM user_access_tokens WHERE user_id = $1;"

	s.Equal(http.StatusOK, response.Code)
	s.Nil(s.conn.QueryRow(query, usr.ID).Scan(&token))

	decrypted, err := utils.Decrypt(token)

	s.NoError(err)
	s.Equal(string(decrypted), myToken)
}

func (s *HandlerUpdatePersonalAccessTokenSuite) Test_Delete_Success() {
	usr := s.MockUser()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(userhandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/user/access-token",
		map[string]string{
			"token": "",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	var token []byte
	var query = "SELECT personal_access_token FROM user_access_tokens WHERE user_id = $1;"

	s.Equal(http.StatusOK, response.Code)
	s.Nil(s.conn.QueryRow(query, usr.ID).Scan(&token))
	s.Nil(token)
}

func TestHandlerUpdatePersonalAccessTokenSuite(t *testing.T) {
	suite.Run(t, &HandlerUpdatePersonalAccessTokenSuite{})
}
