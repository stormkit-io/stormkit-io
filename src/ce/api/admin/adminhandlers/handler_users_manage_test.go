package adminhandlers_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type HandlerUsersManageSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerUsersManageSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerUsersManageSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerUsersManageSuite) Test_Success_Approve() {
	admin := s.MockUser(map[string]any{
		"IsAdmin": true,
	})

	pendingUser := s.MockUser(map[string]any{
		"IsApproved": null.Bool{},
	})

	s.False(pendingUser.IsApproved.Valid)
	s.False(pendingUser.IsApproved.ValueOrZero())

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/users/manage",
		map[string]any{
			"userIds": []string{pendingUser.ID.String()},
			"action":  "approve",
		},
		map[string]string{
			"Authorization": usertest.Authorization(admin.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	usr, err := user.NewStore().UserByID(pendingUser.ID)
	s.NoError(err)
	s.True(usr.IsApproved.ValueOrZero())
}

func (s *HandlerUsersManageSuite) Test_Success_Reject() {
	admin := s.MockUser(map[string]any{
		"IsAdmin": true,
	})

	pendingUser := s.MockUser(map[string]any{
		"IsApproved": null.Bool{},
	})

	s.False(pendingUser.IsApproved.Valid)
	s.False(pendingUser.IsApproved.ValueOrZero())

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/users/manage",
		map[string]any{
			"userIds": []string{pendingUser.ID.String()},
			"action":  "reject",
		},
		map[string]string{
			"Authorization": usertest.Authorization(admin.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	usr, err := user.NewStore().UserByID(pendingUser.ID)
	s.NoError(err)
	s.False(usr.IsApproved.ValueOrZero())
}

func (s *HandlerUsersManageSuite) Test_InvalidAction() {
	admin := s.MockUser(map[string]any{
		"IsAdmin": true,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/users/manage",
		map[string]any{
			"userIds": []string{"1"},
			"action":  "invalid",
		},
		map[string]string{
			"Authorization": usertest.Authorization(admin.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "Invalid action provided")
}

func (s *HandlerUsersManageSuite) Test_EmptyUserIDs() {
	admin := s.MockUser(map[string]any{
		"IsAdmin": true,
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/users/manage",
		map[string]any{
			"userIds": []string{},
			"action":  "approve",
		},
		map[string]string{
			"Authorization": usertest.Authorization(admin.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "No user IDs provided")
}

func (s *HandlerUsersManageSuite) Test_NonAdmin() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/admin/users/manage",
		map[string]any{
			"userIds": []string{"123"},
			"action":  "approve",
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerUsersManage(t *testing.T) {
	suite.Run(t, &HandlerUsersManageSuite{})
}
