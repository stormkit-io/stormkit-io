package schemahandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/schemahandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type HandlerSchemaDeleteSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
	usr  *factory.MockUser
	app  *factory.MockApp
}

func (s *HandlerSchemaDeleteSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	// Create test user and app
	s.usr = s.MockUser(nil)
	s.app = s.MockApp(s.usr, nil)
}

func (s *HandlerSchemaDeleteSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerSchemaDeleteSuite) Test_Success() {
	admin.ResetMockLicense()
	config.SetIsSelfHosted(true)
	defer config.SetIsSelfHosted(false)

	env := s.MockEnv(s.app, map[string]any{
		"SchemaConf": &buildconf.SchemaConf{
			SchemaName: "some_schema",
		},
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(schemahandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/schema?envId=%d", env.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	// Verify schema configuration was cleared
	updatedEnv, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)
	s.NoError(err)
	s.Nil(updatedEnv.SchemaConf)

	// Should not create audit logs for non-enterprise
	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.NoError(err)
	s.Len(audits, 0)
}

func (s *HandlerSchemaDeleteSuite) Test_Success_AuditLogs() {
	admin.SetMockLicense()
	defer admin.ResetMockLicense()

	env := s.MockEnv(s.app, map[string]any{
		"SchemaConf": &buildconf.SchemaConf{
			SchemaName: "some_schema",
		},
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(schemahandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/schema?envId=%d", env.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.NoError(err)
	s.Len(audits, 1)
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "DELETE:SCHEMA",
		EnvName:     env.Name,
		EnvID:       env.ID,
		AppID:       s.app.ID,
		TeamID:      s.app.TeamID,
		UserID:      s.usr.ID,
		UserDisplay: s.usr.Display(),
		Diff: &audit.Diff{
			Old: audit.DiffFields{
				SchemaName: env.SchemaConf.SchemaName,
			},
		},
	}, audits[0])
}

func (s *HandlerSchemaDeleteSuite) Test_Forbidden_NoWriteAccess() {
	// Create a viewer user (no write access)
	viewerUser := s.MockUser(nil)

	s.NoError(team.NewStore().AddMemberToTeam(context.Background(), &team.Member{
		TeamID: s.app.TeamID,
		UserID: viewerUser.ID,
		Role:   team.ROLE_DEVELOPER,
		Status: true,
	}))

	env := s.MockEnv(s.app, map[string]any{
		"SchemaConf": &buildconf.SchemaConf{
			SchemaName: "some_schema",
		},
	})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(schemahandlers.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/schema?envId=%d", env.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(viewerUser.ID),
		},
	)

	s.Equal(http.StatusForbidden, response.Code)

	// Verify schema configuration still exists
	existingEnv, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)
	s.NoError(err)
	s.NotNil(existingEnv.SchemaConf)
}

func TestSchemaDeleteHandler(t *testing.T) {
	suite.Run(t, &HandlerSchemaDeleteSuite{})
}
