package jobs

import (
	"context"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type JobAuthSchemaSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *JobAuthSchemaSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *JobAuthSchemaSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

// Test_CollectAuthSchemaTargets_SelectsOnlyEnabled proves the sweep only picks
// up non-deleted environments that have auth enabled and a provisioned schema —
// disabled-auth, unconfigured, and deleted envs are skipped. This is the
// selection logic that keeps the backfill off unrelated tenants.
func (s *JobAuthSchemaSuite) Test_CollectAuthSchemaTargets_SelectsOnlyEnabled() {
	app := s.MockApp(nil)

	enabled := s.MockEnv(app, map[string]any{
		"Name":       "enabled",
		"AuthConf":   &buildconf.SKAuthConf{Status: true},
		"SchemaConf": &buildconf.SchemaConf{SchemaName: "schema_enabled"},
	})

	// Auth configured but disabled — nothing to reconcile.
	s.MockEnv(app, map[string]any{
		"Name":       "disabled",
		"AuthConf":   &buildconf.SKAuthConf{Status: false},
		"SchemaConf": &buildconf.SchemaConf{SchemaName: "schema_disabled"},
	})

	// Auth enabled but soft-deleted env — must be ignored.
	s.MockEnv(app, map[string]any{
		"Name":       "deleted",
		"AuthConf":   &buildconf.SKAuthConf{Status: true},
		"SchemaConf": &buildconf.SchemaConf{SchemaName: "schema_deleted"},
		"DeletedAt":  utils.Unix{Time: time.Now(), Valid: true},
	})

	// No auth configured at all — filtered out by the query.
	s.MockEnv(app, map[string]any{
		"Name":       "no_auth",
		"SchemaConf": &buildconf.SchemaConf{SchemaName: "schema_no_auth"},
	})

	targets, err := collectAuthSchemaTargets(context.Background(), NewStore())
	s.Require().NoError(err)

	ids := make([]types.ID, 0, len(targets))
	for _, t := range targets {
		ids = append(ids, t.EnvID)
	}

	s.Equal([]types.ID{enabled.ID}, ids)
	s.Require().Len(targets, 1)
	s.Equal("schema_enabled", targets[0].Conf.SchemaName)
}

func TestJobAuthSchemaSuite(t *testing.T) {
	suite.Run(t, &JobAuthSchemaSuite{})
}
