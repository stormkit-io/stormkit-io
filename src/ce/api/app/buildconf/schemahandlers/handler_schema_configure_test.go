package schemahandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/schemahandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerSchemaConfigureSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	usr  *factory.MockUser
	app  *factory.MockApp
	env  *factory.MockEnv
}

func (s *HandlerSchemaConfigureSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	// Create test user, app, and environment
	s.usr = s.MockUser(nil)
	s.app = s.MockApp(s.usr, nil)
	s.env = s.MockEnv(s.app, nil)
}

func (s *HandlerSchemaConfigureSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerSchemaConfigureSuite) Test_Success_EnableMigrations() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(schemahandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/schema/configure",
		map[string]any{
			"envId":             s.env.ID,
			"appId":             s.app.ID,
			"migrationsEnabled": true,
			"migrationsPath":    "/migrations",
		},
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	// Verify schema configuration was saved
	env, err := buildconf.NewStore().EnvironmentByID(context.Background(), s.env.ID)
	s.NoError(err)
	s.NotNil(env.SchemaConf, "SchemaConf should be set")
	s.True(env.SchemaConf.MigrationsEnabled, "MigrationsEnabled should be true")
	s.Equal("/migrations", env.SchemaConf.MigrationsPath, "MigrationsPath should match")
}

func (s *HandlerSchemaConfigureSuite) Test_Success_DisableMigrations() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(schemahandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/schema/configure",
		map[string]any{
			"envId":             s.env.ID,
			"appId":             s.app.ID,
			"migrationsEnabled": false,
			"migrationsPath":    "",
		},
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	// Verify schema configuration was saved
	env, err := buildconf.NewStore().EnvironmentByID(context.Background(), s.env.ID)
	s.NoError(err)
	s.NotNil(env.SchemaConf, "SchemaConf should be set")
	s.False(env.SchemaConf.MigrationsEnabled, "MigrationsEnabled should be false")
	s.Equal("", env.SchemaConf.MigrationsPath, "MigrationsPath should be empty")
}

func (s *HandlerSchemaConfigureSuite) Test_Success_UpdateMigrationsPath() {
	// First, set initial configuration
	schema := buildconf.SchemaConf{
		MigrationUserName: "user",
		MigrationPassword: "pass",
		MigrationsEnabled: false,
		MigrationsPath:    "/migrations",
	}

	s.NoError(buildconf.NewStore().SaveSchemaConf(context.Background(), s.env.ID, &schema))

	// Update migrations path
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(schemahandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/schema/configure",
		map[string]any{
			"envId":             s.env.ID,
			"appId":             s.app.ID,
			"migrationsEnabled": true,
			"migrationsPath":    "/app/db/migrations",
		},
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	// Verify schema configuration was updated
	env, err := buildconf.NewStore().EnvironmentByID(context.Background(), s.env.ID)
	s.NoError(err)
	s.NotNil(env.SchemaConf, "SchemaConf should be set")
	s.True(env.SchemaConf.MigrationsEnabled, "MigrationsEnabled should be true")
	s.Equal("/app/db/migrations", env.SchemaConf.MigrationsPath, "MigrationsPath should be updated")
	s.Equal("user", env.SchemaConf.MigrationUserName, "MigrationUserName should remain unchanged")
	s.Equal("pass", env.SchemaConf.MigrationPassword, "MigrationPassword should remain unchanged")
}

func (s *HandlerSchemaConfigureSuite) Test_MissingEnvId() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(schemahandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/schema/configure",
		map[string]any{
			"appId":             s.app.ID,
			"migrationsEnabled": true,
			"migrationsPath":    "/migrations",
		},
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func TestHandlerSchemaConfigure(t *testing.T) {
	suite.Run(t, &HandlerSchemaConfigureSuite{})
}
