package jobs_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type JobEnvironmentsSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *JobEnvironmentsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *JobEnvironmentsSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *JobEnvironmentsSuite) Test_RemoveStateEnvironments() {
	app := s.MockApp(nil)
	env := s.MockEnv(app, map[string]any{
		"DeletedAt": utils.Unix{
			Time:  time.Now(),
			Valid: true,
		},
	})

	s.MockDeployments(10, env)

	env2 := s.MockEnv(app, map[string]any{
		"DeletedAt": utils.Unix{
			Time:  time.Now(),
			Valid: true,
		},
	})

	s.MockDeployments(10, env2)

	err := jobs.RemoveStaleEnvironments(context.TODO())
	s.NoError(err)

	tmp, _ := s.conn.Prepare("SELECT deleted_at FROM deployments;")
	res, _ := tmp.Query()

	defer res.Close()

	for res.Next() {
		var deletedAt utils.Unix
		err := res.Scan(&deletedAt)
		s.NoError(err)
		s.NotEmpty(deletedAt)
	}
}

func (s *JobEnvironmentsSuite) Test_RemoveStaleEnvironments_DropsSchemas() {
	app := s.MockApp(nil)

	envWithSchema := s.MockEnv(app, map[string]any{
		"DeletedAt": utils.Unix{Time: time.Now(), Valid: true},
	})

	schemaName := buildconf.SchemaName(app.ID, envWithSchema.ID)

	_, err := s.conn.Exec(
		"UPDATE apps_build_conf SET schema_conf = $1 WHERE env_id = $2",
		&buildconf.SchemaConf{SchemaName: schemaName},
		envWithSchema.ID,
	)
	s.NoError(err)

	_, err = s.conn.Exec(fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %s", schemaName))
	s.NoError(err)

	defer func() {
		_, _ = s.conn.Exec(fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schemaName))
	}()

	// A live environment with a schema must not be touched.
	envLive := s.MockEnv(app, map[string]any{
		"SchemaConf": &buildconf.SchemaConf{
			SchemaName: "should_not_be_touched",
		},
	})

	s.NoError(jobs.RemoveStaleEnvironments(context.TODO()))

	var exists bool

	s.NoError(s.conn.
		QueryRow(`SELECT EXISTS (SELECT 1 FROM information_schema.schemata WHERE schema_name = $1)`, schemaName).
		Scan(&exists),
	)

	s.False(exists, "schema for deleted env should be dropped")

	var staleConf []byte
	s.NoError(s.conn.
		QueryRow("SELECT schema_conf FROM apps_build_conf WHERE env_id = $1", envWithSchema.ID).
		Scan(&staleConf),
	)
	s.Nil(staleConf, "schema_conf should be cleared on the deleted env")

	var liveConf []byte
	s.NoError(s.conn.
		QueryRow("SELECT schema_conf FROM apps_build_conf WHERE env_id = $1", envLive.ID).
		Scan(&liveConf),
	)
	s.NotEmpty(liveConf, "schema_conf on live env must be preserved")
}

func TestJobEnvironmentsSuite(t *testing.T) {
	suite.Run(t, &JobEnvironmentsSuite{})
}
