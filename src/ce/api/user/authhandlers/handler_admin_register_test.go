package authhandlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/authhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type HandlerAdminRegisterSuite struct {
	suite.Suite
	*factory.Factory

	service *mocks.MicroServiceInterface
	conn    databasetest.TestDB
}

func (s *HandlerAdminRegisterSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.service = &mocks.MicroServiceInterface{}
	s.NoError(admin.Store().DeleteConfig(context.Background()))

	rediscache.DefaultService = s.service
}

func (s *HandlerAdminRegisterSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	rediscache.DefaultService = nil
}

func (s *HandlerAdminRegisterSuite) Test_Admin_Register_Success() {
	s.service.On("Broadcast", rediscache.EventInvalidateAdminCache).Return(nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth/admin/register",
		map[string]string{
			"email":    "test@admin.com",
			"password": "password",
		},
		nil,
	)

	s.Equal(http.StatusOK, response.Code)

	// Verify the admin config was created
	cfg, err := admin.Store().Config(context.Background())
	s.NoError(err)
	s.NotNil(cfg.AdminUserConfig)
	s.Equal("test@admin.com", cfg.AdminUserConfig.Email)
	s.Equal("password", utils.DecryptToString(cfg.AdminUserConfig.Password))

	// Verify the response contains user and session token
	var responseData map[string]any
	s.NoError(json.Unmarshal(response.Byte(), &responseData))

	// Check that user data is returned
	s.Contains(responseData, "user")
	s.Contains(responseData, "sessionToken")

	userData, ok := responseData["user"].(map[string]any)
	s.True(ok, "User data should be a map")

	// Verify user properties (based on user.JSON() method)
	s.Contains(userData, "id")
	s.Contains(userData, "email")
	s.Contains(userData, "isAdmin")
	s.Contains(userData, "displayName")
	s.Contains(userData, "fullName")
	s.Equal(true, userData["isAdmin"])

	// Verify email in user object (it's a single string field)
	email, ok := userData["email"].(string)
	s.True(ok, "Email should be a string")
	s.Equal("test@admin.com", email)

	// Verify session token is not empty
	sessionToken, ok := responseData["sessionToken"].(string)
	s.True(ok, "Session token should be a string")
	s.NotEmpty(sessionToken)
}

func (s *HandlerAdminRegisterSuite) Test_Admin_Register_FailAdminUserAlreadyExists() {
	s.service.On("Broadcast", rediscache.EventInvalidateAdminCache).Return(nil).Once()

	s.NoError(admin.Store().UpsertConfig(context.Background(), admin.InstanceConfig{
		AuthConfig: &admin.AuthConfig{},
		AdminUserConfig: &admin.AdminUserConfig{
			Email:    "test@admin.com",
			Password: utils.EncryptToString("password"),
		},
	}))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth/admin/register",
		map[string]string{
			"email":    "test@admin.com",
			"password": "password",
		},
		nil,
	)

	s.Equal(http.StatusConflict, response.Code)
}

func (s *HandlerAdminRegisterSuite) Test_Admin_Register_NotAllowed_ProvidersAlreadySet() {
	s.service.On("Broadcast", rediscache.EventInvalidateAdminCache).Return(nil).Once()
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{
		Github: admin.GithubConfig{
			ClientID:     "my-id",
			ClientSecret: "my-secret",
		},
		Gitlab: admin.GitlabConfig{
			ClientID:     "my-client-id",
			ClientSecret: "my-secret",
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.Background(), cnf))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth/admin/register",
		map[string]string{
			"email":    "test@admin.com",
			"password": "password",
		},
		nil,
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func (s *HandlerAdminRegisterSuite) Test_Admin_Register_FailInvalidPassword() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth/admin/register",
		map[string]string{
			"email":    "test@admin.com",
			"password": "pass",
		},
		nil,
	)

	expected := `{ "error": "Password must be at least 6 characters long." }`

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerAdminRegisterSuite(t *testing.T) {
	suite.Run(t, &HandlerAdminRegisterSuite{})
}
