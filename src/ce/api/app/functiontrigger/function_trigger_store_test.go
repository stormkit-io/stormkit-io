package functiontrigger_test

import (
	"context"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type TriggerFunctionStoreSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *TriggerFunctionStoreSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *TriggerFunctionStoreSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

// insertDueTrigger inserts a trigger that is already due, optionally with
// documentation. Rows written before the documentation column existed have it
// as NULL, which is what a nil value reproduces here.
func (s *TriggerFunctionStoreSuite) insertDueTrigger(envID types.ID, documentation any) types.ID {
	tid := types.ID(0)

	err := s.conn.QueryRow(`
		INSERT INTO
			function_triggers (env_id, trigger_status, trigger_options, cron, next_run_at, documentation)
		VALUES
			($1, true, $2, '* * * * *', timezone('utc', now()) - interval '1 hour', $3)
		RETURNING
			trigger_id;
	`, envID, []byte(`{"url": "https://example.com"}`), documentation).Scan(&tid)

	s.Require().NoError(err)

	return tid
}

// Test_NullDocumentationReadsAsEmpty covers rows written before the column
// existed: they hold NULL and must scan into an empty string rather than
// failing, which would drop the trigger from every listing.
func (s *TriggerFunctionStoreSuite) Test_NullDocumentationReadsAsEmpty() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	tid := s.insertDueTrigger(env.ID, nil)

	tf, err := functiontrigger.NewStore().ByID(context.Background(), tid)

	s.Require().NoError(err)
	s.Require().NotNil(tf)
	s.Empty(tf.Documentation)

	list, err := functiontrigger.NewStore().List(context.Background(), env.ID)

	s.Require().NoError(err)
	s.Len(list, 1)
}

// Test_DueTriggersCarryDocumentation verifies the scheduler's rows
// carry the notes the same way the display paths do, so the column list and the
// scan stay identical across every select.
func (s *TriggerFunctionStoreSuite) Test_DueTriggersCarryDocumentation() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	tid := s.insertDueTrigger(env.ID, "Ping #data when this fails.")

	store := functiontrigger.NewStore()
	due, err := store.DueTriggers(context.Background())

	s.Require().NoError(err)
	s.Require().NotEmpty(due)

	for _, tf := range due {
		s.Equal("Ping #data when this fails.", tf.Documentation)
	}

	tf, err := store.ByID(context.Background(), tid)

	s.Require().NoError(err)
	s.Equal("Ping #data when this fails.", tf.Documentation)
}

func TestTriggerFunctionStore(t *testing.T) {
	suite.Run(t, &TriggerFunctionStoreSuite{})
}
