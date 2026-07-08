package buildconf

import (
	"context"
	"fmt"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stretchr/testify/suite"
)

// SchemaModelSuite is an internal (package buildconf) suite so it can assert
// against the unexported schemaStoreCache and authSchemaEnsured maps. Migration
// roles are created with CONNECTION LIMIT 1, so a migration store lingering in
// the cache permanently occupies the role's only connection slot — these tests
// guard against reintroducing that leak.
type SchemaModelSuite struct {
	suite.Suite

	conn       databasetest.TestDB
	schemaName string
}

func (s *SchemaModelSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.schemaName = "schema_model_test"
}

func (s *SchemaModelSuite) AfterTest(_, _ string) {
	if err := SchemaStore().DropSchema(context.Background(), s.schemaName); err != nil {
		fmt.Printf("cleanup error (ignored): %v\n", err)
	}

	s.conn.CloseTx()
}

// migrationConf creates the schema and returns a SchemaConf wired to the test
// transaction, mirroring the helper in schema_store_test.go.
func (s *SchemaModelSuite) migrationConf(ctx context.Context) *SchemaConf {
	conf, err := SchemaStore().CreateSchema(ctx, s.schemaName)

	s.Require().NoError(err)
	s.Require().NotNil(conf)

	conf.MigrationUserName = database.Config.User
	conf.MigrationPassword = database.Config.Password
	conf.AppUserName = database.Config.User
	conf.AppPassword = database.Config.Password
	conf.DriverName = "txdb"

	return conf
}

func (s *SchemaModelSuite) Test_Store_MigrationsAccessNotCached() {
	ctx := context.Background()
	conf := s.migrationConf(ctx)

	first, err := conf.Store(SchemaAccessTypeMigrations)
	s.Require().NoError(err)
	defer first.Close()

	second, err := conf.Store(SchemaAccessTypeMigrations)
	s.Require().NoError(err)
	defer second.Close()

	s.NotSame(first, second, "migration stores must not be shared between callers")

	_, cached := schemaStoreCache.Load(conf.storeKey(SchemaAccessTypeMigrations))
	s.False(cached, "migration stores must never enter the cache")
}

func (s *SchemaModelSuite) Test_Store_AppUserAccessCached() {
	ctx := context.Background()
	conf := s.migrationConf(ctx)

	first, err := conf.Store(SchemaAccessTypeAppUser)
	s.Require().NoError(err)

	second, err := conf.Store(SchemaAccessTypeAppUser)
	s.Require().NoError(err)

	s.Same(first, second, "app-user stores are pooled via the cache")

	s.NoError(first.Close())

	_, cached := schemaStoreCache.Load(conf.storeKey(SchemaAccessTypeAppUser))
	s.False(cached, "Close must evict the store from the cache")
}

func (s *SchemaModelSuite) Test_EnsureAuthSchema_DoesNotLeakStore() {
	ctx := context.Background()
	conf := s.migrationConf(ctx)
	key := conf.storeKey(SchemaAccessTypeMigrations)

	s.Require().NoError(conf.EnsureAuthSchema(ctx))

	_, cached := schemaStoreCache.Load(key)
	s.False(cached, "EnsureAuthSchema must not leave a migration store behind")

	_, ensured := authSchemaEnsured.Load(key)
	s.True(ensured, "successful runs are marked so later calls no-op")

	s.NoError(conf.EnsureAuthSchema(ctx), "second call must no-op")
}

func TestSchemaModelSuite(t *testing.T) {
	suite.Run(t, &SchemaModelSuite{})
}
