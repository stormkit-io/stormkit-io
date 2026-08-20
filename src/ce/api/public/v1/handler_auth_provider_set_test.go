package publicapiv1_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthProviderSetSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
	app  *factory.MockApp
	env  *factory.MockEnv
	key  *factory.MockAPIKey
}

func (s *HandlerAuthProviderSetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.app = s.MockApp(nil)

	s.env = s.MockEnv(s.app, map[string]any{
		"SchemaConf": &buildconf.SchemaConf{
			Host:              s.conn.Cfg.Host,
			Port:              s.conn.Cfg.Port,
			DBName:            s.conn.Cfg.DBName,
			SchemaName:        s.conn.Cfg.Schema,
			AppUserName:       s.conn.Cfg.User,
			AppPassword:       s.conn.Cfg.Password,
			MigrationPassword: s.conn.Cfg.Password,
			MigrationUserName: s.conn.Cfg.User,
			MigrationsEnabled: true,
		},
	})

	s.key = s.MockAPIKey(nil, s.env, map[string]any{
		"Scope": apikey.SCOPE_ENV,
		"EnvID": s.env.ID,
	})
}

func (s *HandlerAuthProviderSetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAuthProviderSetSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerAuthProviderSetSuite) auth() map[string]string {
	return map[string]string{"Authorization": s.key.Value}
}

func (s *HandlerAuthProviderSetSuite) Test_Forbidden_NoAPIKey() {
	for _, method := range []string{shttp.MethodGet, shttp.MethodPost} {
		response := shttptest.RequestWithHeaders(
			s.handler(),
			method,
			"/v1/auth/providers",
			nil,
			map[string]string{},
		)

		s.Equal(http.StatusForbidden, response.Code)
	}
}

func (s *HandlerAuthProviderSetSuite) Test_Set_StoresProvider() {
	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/auth/providers",
		map[string]any{
			"envId":        s.env.ID,
			"providerName": "google",
			"clientId":     "client-id",
			"clientSecret": "client-secret",
			"status":       true,
		},
		s.auth(),
	)

	s.Equal(http.StatusOK, response.Code)

	// The secret must never be echoed back on the write path either.
	s.NotContains(response.String(), "client-secret")

	provider, err := skauth.NewStore().Provider(context.Background(), s.env.ID, skauth.ProviderGoogle)

	s.Require().NoError(err)
	s.Require().NotNil(provider)
	s.True(provider.Status)
	s.Equal("client-id", provider.Data.ClientID)
	s.Equal("client-secret", provider.Data.ClientSecret)
}

// Test_Set_ValidationErrorIsBadRequest covers the ProviderValidationError
// branch: magiclink has no from address, so the upsert must be rejected with a
// 400 carrying the message rather than a generic 500.
func (s *HandlerAuthProviderSetSuite) Test_Set_ValidationErrorIsBadRequest() {
	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/auth/providers",
		map[string]any{
			"envId":        s.env.ID,
			"providerName": "magiclink",
		},
		s.auth(),
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "From address is required")
}

func (s *HandlerAuthProviderSetSuite) Test_Set_IsAudited() {
	admin.SetMockLicense()
	defer admin.ResetMockLicense()

	response := shttptest.RequestWithHeaders(
		s.handler(),
		shttp.MethodPost,
		"/v1/auth/providers",
		map[string]any{
			"envId":        s.env.ID,
			"providerName": "magiclink",
			"fromAddress":  "noreply@acme.com",
		},
		s.auth(),
	)

	s.Equal(http.StatusOK, response.Code)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: s.env.ID,
	})

	s.Require().NoError(err)
	s.Require().Len(audits, 1)
	s.Equal("UPDATE:AUTHPROVIDER", audits[0].Action)
}

func TestHandlerAuthProviderSetSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthProviderSetSuite{})
}
