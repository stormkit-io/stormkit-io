package adminhandlers_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type HandlerUsersPendingSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerUsersPendingSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerUsersPendingSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerUsersPendingSuite) Test_Success() {
	admin := s.MockUser(map[string]any{
		"IsAdmin": true,
	})

	// Create some pending users
	s.MockUser(map[string]any{
		"IsApproved": null.Bool{},
		"Metadata":   user.UserMeta{},
		"Emails": []oauth.Email{
			{
				Address:    strings.ToLower(strings.TrimSpace("user1@pending.com")),
				IsPrimary:  true,
				IsVerified: true,
			},
		},
	})

	s.MockUser(map[string]any{
		"IsApproved": null.Bool{},
		"Metadata":   user.UserMeta{},
		"Emails": []oauth.Email{
			{
				Address:    strings.ToLower(strings.TrimSpace("user2@pending.com")),
				IsPrimary:  true,
				IsVerified: true,
			},
		},
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/users/pending",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(admin.ID),
		},
	)

	str := response.String()

	s.Equal(http.StatusOK, response.Code)
	s.Contains(str, "user1@pending.com")
	s.Contains(str, "user2@pending.com")
}

func (s *HandlerUsersPendingSuite) Test_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/users/pending",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerUsersPending(t *testing.T) {
	suite.Run(t, &HandlerUsersPendingSuite{})
}
