package authhandlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/authhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type HandlerAdminLoginSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerAdminLoginSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.NoError(admin.Store().DeleteConfig(context.Background()))
}

func (s *HandlerAdminLoginSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAdminLoginSuite) Test_Admin_Login_Success() {
	usr := s.MockUser(map[string]any{
		"Emails": []oauth.Email{
			{Address: "hello@stormkit.io", IsVerified: false, IsPrimary: false},
			{Address: "test@admin.com", IsVerified: true, IsPrimary: true},
		},
	})

	s.NotNil(usr)

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
		"/auth/admin/login",
		map[string]string{
			"email":    "test@admin.com",
			"password": "password",
		},
		nil,
	)

	expected := struct {
		User         *user.User `json:"user"`
		SessionToken string     `json:"sessionToken"`
	}{}

	s.Equal(http.StatusOK, response.Code)
	s.NoError(json.Unmarshal(response.Byte(), &expected))
	s.NotEmpty(expected.SessionToken)
}

func (s *HandlerAdminLoginSuite) Test_Admin_Login_InvalidPassword() {
	usr := s.MockUser(map[string]any{
		"Emails": []oauth.Email{
			{Address: "test@admin.com", IsVerified: true, IsPrimary: true},
		},
	})

	s.NotNil(usr)

	s.NoError(admin.Store().UpsertConfig(context.Background(), admin.InstanceConfig{
		AuthConfig: &admin.AuthConfig{},
		AdminUserConfig: &admin.AdminUserConfig{
			Email:    "test@admin.com",
			Password: utils.EncryptToString("Password"),
		},
	}))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth/admin/login",
		map[string]string{
			"email":    "test@admin.com",
			"password": "password", // Notice the case difference
		},
		nil,
	)

	s.Equal(http.StatusUnauthorized, response.Code)
}

func (s *HandlerAdminLoginSuite) Test_Admin_NotAllowed_ProvidersAlreadySet() {
	cnf := admin.MustConfig()
	cnf.AuthConfig = &admin.AuthConfig{
		Github: admin.GithubConfig{
			ClientID:     "my-client-id",
			ClientSecret: "my-secret",
			Account:      "my-account",
			AppID:        123456,
			PrivateKey:   "my-private-key",
		},
	}

	s.NoError(admin.Store().UpsertConfig(context.Background(), cnf))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth/admin/login",
		map[string]string{
			"email":    "test@admin.com",
			"password": "password",
		},
		nil,
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func TestHandlerAdminLoginSuite(t *testing.T) {
	suite.Run(t, &HandlerAdminLoginSuite{})
}
