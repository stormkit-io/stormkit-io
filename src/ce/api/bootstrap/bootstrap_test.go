package bootstrap_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/bootstrap"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/mocks"
)

type BootstrapSuite struct {
	suite.Suite
	*factory.Factory

	service *mocks.MicroServiceInterface
	conn    databasetest.TestDB
}

func (s *BootstrapSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.service = &mocks.MicroServiceInterface{}
	s.service.On("Broadcast", rediscache.EventInvalidateAdminCache).Return(nil)
	rediscache.DefaultService = s.service

	s.NoError(admin.Store().DeleteConfig(context.Background()))
}

func (s *BootstrapSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	rediscache.DefaultService = nil
}

func (s *BootstrapSuite) Test_Run_CreatesAdminAndMintsOwnerKey() {
	res, err := bootstrap.Run(context.Background(), bootstrap.Params{
		AdminEmail:    "agent@example.com",
		AdminPassword: "supersecret123",
	})

	s.NoError(err)
	s.True(res.Created)
	s.True(strings.HasPrefix(res.APIKey, "SK_"))

	cfg, err := admin.Store().Config(context.Background())
	s.NoError(err)
	s.NotNil(cfg.AdminUserConfig)
	s.Equal("agent@example.com", cfg.AdminUserConfig.Email)

	// The minted key resolves at user scope with a real user, which is what
	// the /v1/mcp endpoint requires.
	key, err := apikey.NewStore().APIKey(context.Background(), res.APIKey)
	s.NoError(err)
	s.Equal(apikey.SCOPE_USER, key.Scope)
	s.True(key.UserID > 0)
}

func (s *BootstrapSuite) Test_Run_UsesProvidedKey() {
	provided := "SK_" + utils.RandomToken(62)

	res, err := bootstrap.Run(context.Background(), bootstrap.Params{
		AdminEmail:    "agent@example.com",
		AdminPassword: "supersecret123",
		AgentAPIKey:   provided,
	})

	s.NoError(err)
	s.True(res.Created)
	s.Equal(provided, res.APIKey)

	key, err := apikey.NewStore().APIKey(context.Background(), provided)
	s.NoError(err)
	s.Equal(apikey.SCOPE_USER, key.Scope)
}

func (s *BootstrapSuite) Test_Run_IsIdempotent() {
	first, err := bootstrap.Run(context.Background(), bootstrap.Params{
		AdminEmail:    "agent@example.com",
		AdminPassword: "supersecret123",
	})

	s.NoError(err)
	s.True(first.Created)

	second, err := bootstrap.Run(context.Background(), bootstrap.Params{
		AdminEmail:    "someone-else@example.com",
		AdminPassword: "anothersecret123",
	})

	s.NoError(err)
	s.False(second.Created)
	s.Empty(second.APIKey)

	// The original admin is untouched.
	cfg, err := admin.Store().Config(context.Background())
	s.NoError(err)
	s.Equal("agent@example.com", cfg.AdminUserConfig.Email)
}

func (s *BootstrapSuite) Test_Run_NoOpWhenUsersExistWithoutAdminConfig() {
	// Simulate an instance first provisioned via OAuth/magic link: a user
	// exists but AdminUserConfig was never set.
	s.MockUser()

	res, err := bootstrap.Run(context.Background(), bootstrap.Params{
		AdminEmail:    "agent@example.com",
		AdminPassword: "supersecret123",
	})

	s.NoError(err)
	s.False(res.Created)
	s.Empty(res.APIKey)

	cfg, err := admin.Store().Config(context.Background())
	s.NoError(err)
	s.Nil(cfg.AdminUserConfig)
}

func (s *BootstrapSuite) Test_Run_RetriesAfterPartialFailure() {
	// Simulate a prior partial run: the admin user exists but AdminUserConfig
	// was never written (a later step failed). The retry must finish the flow
	// rather than short-circuit into a permanently unconfigured instance.
	_, err := user.NewStore().MustUser(oauth.NewAdminUser("agent@example.com"))
	s.NoError(err)

	res, err := bootstrap.Run(context.Background(), bootstrap.Params{
		AdminEmail:    "agent@example.com",
		AdminPassword: "supersecret123",
	})

	s.NoError(err)
	s.True(res.Created)
	s.True(strings.HasPrefix(res.APIKey, "SK_"))

	cfg, err := admin.Store().Config(context.Background())
	s.NoError(err)
	s.NotNil(cfg.AdminUserConfig)
	s.Equal("agent@example.com", cfg.AdminUserConfig.Email)

	key, err := apikey.NewStore().APIKey(context.Background(), res.APIKey)
	s.NoError(err)
	s.Equal(apikey.SCOPE_USER, key.Scope)
}

func (s *BootstrapSuite) Test_Run_RetryDoesNotDuplicateStoredKey() {
	provided := "SK_" + utils.RandomToken(62)

	// Prior partial run: user and its key are stored, but AdminUserConfig was
	// not yet written. The retry must reuse the stored key, not add a second.
	usr, err := user.NewStore().MustUser(oauth.NewAdminUser("agent@example.com"))
	s.NoError(err)

	s.NoError(apikey.NewStore().AddAPIKey(context.Background(), &apikey.Token{
		UserID: usr.ID,
		Name:   "agent",
		Scope:  apikey.SCOPE_USER,
		Value:  provided,
	}))

	res, err := bootstrap.Run(context.Background(), bootstrap.Params{
		AdminEmail:    "agent@example.com",
		AdminPassword: "supersecret123",
		AgentAPIKey:   provided,
	})

	s.NoError(err)
	s.True(res.Created)
	s.Equal(provided, res.APIKey)

	keys, err := apikey.NewStore().APIKeys(context.Background(), usr.ID, apikey.SCOPE_USER)
	s.NoError(err)
	s.Len(keys, 1)
}

func (s *BootstrapSuite) Test_Run_RejectsShortPassword() {
	res, err := bootstrap.Run(context.Background(), bootstrap.Params{
		AdminEmail:    "agent@example.com",
		AdminPassword: "elevenchar1",
	})

	s.Error(err)
	s.Nil(res)
}

func (s *BootstrapSuite) Test_Run_RejectsBadKeyPrefix() {
	res, err := bootstrap.Run(context.Background(), bootstrap.Params{
		AdminEmail:    "agent@example.com",
		AdminPassword: "supersecret123",
		AgentAPIKey:   "not-a-stormkit-key",
	})

	s.Error(err)
	s.Nil(res)
}

func TestBootstrapSuite(t *testing.T) {
	suite.Run(t, &BootstrapSuite{})
}
