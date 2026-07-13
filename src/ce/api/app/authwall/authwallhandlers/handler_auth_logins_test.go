package authwallhandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall/authwallhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthLoginsSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerAuthLoginsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerAuthLoginsSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerAuthLoginsSuite) Test_AuthList_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	aw := &authwall.AuthWall{
		LoginEmail:    "email@example.org",
		LoginPassword: "123pass",
		EnvID:         env.ID,
	}

	s.NoError(authwall.Store().CreateLogin(context.Background(), aw))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/auth-wall?envId=%s", env.ID.String()),
		nil,
		map[string]string{
			"authorization": usertest.Authorization(usr.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"logins": [
			{ "email": "email@example.org", "lastLogin": 0, "id": "%d" }
		]
	}`, aw.LoginID)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerAuthLoginsSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthLoginsSuite{})
}
