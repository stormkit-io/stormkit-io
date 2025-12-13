package schemahandlers_test

import (
	"context"
	"fmt"
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

type HandlerSchemaGetSuite struct {
	suite.Suite
	*factory.Factory
	conn       databasetest.TestDB
	usr        *factory.MockUser
	app        *factory.MockApp
	env        *factory.MockEnv
	schemaName string
}

func (s *HandlerSchemaGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	// Create test user, app, and environment
	s.usr = s.MockUser(nil)
	s.app = s.MockApp(s.usr, nil)
	s.env = s.MockEnv(s.app, nil)

	// Create schema
	s.schemaName = buildconf.SchemaName(s.app.ID, s.env.ID)

	if _, err := s.conn.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", s.schemaName)); err != nil {
		panic(err)
	}
}

func (s *HandlerSchemaGetSuite) AfterTest(_, _ string) {
	if _, err := s.conn.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", s.schemaName)); err != nil {
		panic(err)
	}

	s.conn.CloseTx()
}

func (s *HandlerSchemaGetSuite) Test_Success_WithTables() {
	// Create a test table
	_, err := s.conn.Exec(fmt.Sprintf(`CREATE TABLE %s.test_table ( id SERIAL PRIMARY KEY, name TEXT NOT NULL )`, s.schemaName))

	s.NoError(err)

	// Insert some test data
	_, err = s.conn.Exec(fmt.Sprintf(`
		INSERT INTO %s.test_table (name) VALUES ('test1'), ('test2'), ('test3')
	`, s.schemaName))

	s.NoError(err)

	// Update schema configuration
	s.env.SchemaConf = &buildconf.SchemaConf{
		MigrationsEnabled: true,
		MigrationsPath:    "/migrations",
	}

	s.NoError(buildconf.NewStore().SaveSchemaConf(context.Background(), s.env.ID, s.env.SchemaConf))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(schemahandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/schema?envId=%d", s.env.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	// rows is 0 because ANALYZE has not been run yet
	// even if we run it, since we are in a transaction, the stats won't be updated
	expected := fmt.Sprintf(`{
		"schema": {
			"name": "%s",
			"migrationsEnabled": true,
			"migrationsPath": "/migrations",
			"tables": [
				{
					"name": "test_table",
					"size": 8192,
					"rows": 0
				}
			]
		}
	}`, s.schemaName)

	s.JSONEq(expected, response.String())
	s.Equal(http.StatusOK, response.Code)
}

func (s *HandlerSchemaGetSuite) Test_Success_EmptySchema() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(schemahandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/schema?envId=%d", s.env.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"schema": {
			"name": "%s",
			"tables": []
		}
	}`, s.schemaName)

	s.JSONEq(expected, response.String())
	s.Equal(http.StatusOK, response.Code)
}

func (s *HandlerSchemaGetSuite) Test_MissingSchema() {
	// Create a new environment without creating its schema
	newEnv := s.MockEnv(s.app, map[string]any{
		"Name": "test-dev",
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(schemahandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/schema?envId=%d", newEnv.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	s.JSONEq(`{ "schema": null }`, response.String())
	s.Equal(http.StatusOK, response.Code)
}

func (s *HandlerSchemaGetSuite) Test_MissingEnvId() {
	usr := s.MockUser(nil)
	app := s.MockApp(usr, nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(schemahandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/schema?appId=%d", app.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func TestHandlerSchemaGet(t *testing.T) {
	suite.Run(t, &HandlerSchemaGetSuite{})
}
