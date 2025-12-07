package schemahandlers_test

import (
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

type HandlerSchemaSetSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	usr  *factory.MockUser
	app  *factory.MockApp
	env  *factory.MockEnv
}

func (s *HandlerSchemaSetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	// Create test user, app, and environment
	s.usr = s.MockUser(nil)
	s.app = s.MockApp(s.usr, nil)
	s.env = s.MockEnv(s.app, nil)
}

func (s *HandlerSchemaSetSuite) AfterTest(_, _ string) {
	// Clean up schema if it was created
	schemaName := buildconf.SchemaName(s.app.ID, s.env.ID)

	if _, err := s.conn.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName)); err != nil {
		panic(err)
	}

	s.conn.CloseTx()
}

func (s *HandlerSchemaSetSuite) Test_Success_CreateSchema() {
	schemaName := buildconf.SchemaName(s.app.ID, s.env.ID)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(schemahandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/schema",
		map[string]any{
			"envId": s.env.ID,
		},
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(fmt.Sprintf(`{"schema": "%s"}`, schemaName), response.String())

	// Verify schema was created by querying information_schema
	var exists bool

	s.NoError(s.conn.
		QueryRow(`SELECT EXISTS ( SELECT 1 FROM information_schema.schemata  WHERE schema_name = $1 )`, schemaName).
		Scan(&exists),
	)

	s.True(exists, "Schema should exist in database")
}

func (s *HandlerSchemaSetSuite) Test_Success_SchemaAlreadyExists() {
	schemaName := buildconf.SchemaName(s.app.ID, s.env.ID)

	// Create schema first
	_, err := s.conn.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName))
	s.NoError(err)

	// Try to create again - should succeed with IF NOT EXISTS
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(schemahandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/schema",
		map[string]any{
			"envId": s.env.ID,
		},
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(fmt.Sprintf(`{"schema": "%s"}`, schemaName), response.String())
}

func (s *HandlerSchemaSetSuite) Test_MissingEnvId() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(schemahandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/schema",
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
}

func TestHandlerSchemaSet(t *testing.T) {
	suite.Run(t, &HandlerSchemaSetSuite{})
}
