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

type TriggerDescriptionSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *TriggerDescriptionSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *TriggerDescriptionSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

// Rows written before the column existed read as an empty description rather
// than failing the scan.
func (s *TriggerDescriptionSuite) Test_ExistingTriggersHaveNoDescription() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	tid := types.ID(0)

	err := s.conn.QueryRow(`
		INSERT INTO
			function_triggers (env_id, trigger_status, trigger_options, cron)
		VALUES
			($1, true, $2, '* * * * *')
		RETURNING
			trigger_id;
	`, env.ID, []byte(`{"url": "https://example.com"}`)).Scan(&tid)

	s.Require().NoError(err)

	tf, err := functiontrigger.NewStore().ByID(context.Background(), tid)

	s.Require().NoError(err)
	s.Require().NotNil(tf)
	s.Empty(tf.Description)
}

func (s *TriggerDescriptionSuite) Test_DescriptionRoundTrips() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	store := functiontrigger.NewStore()

	record := &functiontrigger.FunctionTrigger{
		EnvID:       env.ID,
		Cron:        "* * * * *",
		Description: "Autofill weekly newsletter",
		Status:      true,
		Options:     functiontrigger.Options{URL: "https://example.com/api/cron"},
	}

	s.Require().NoError(store.Insert(context.Background(), record))

	stored, err := store.ByID(context.Background(), record.ID)

	s.Require().NoError(err)
	s.Equal("Autofill weekly newsletter", stored.Description)

	stored.Description = "Autofill the newsletter draft"

	s.Require().NoError(store.Update(context.Background(), stored))

	updated, err := store.ByID(context.Background(), record.ID)

	s.Require().NoError(err)
	s.Equal("Autofill the newsletter draft", updated.Description)
}

// The scheduler reads its own column list, so a field the display paths return
// must reach the batch that actually fires the trigger too.
func (s *TriggerDescriptionSuite) Test_DueTriggersCarryDescription() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	tid := types.ID(0)

	err := s.conn.QueryRow(`
		INSERT INTO
			function_triggers (env_id, trigger_status, trigger_options, cron, next_run_at, description)
		VALUES
			($1, true, $2, '* * * * *', timezone('utc', now()) - interval '1 hour', 'Nightly rollup')
		RETURNING
			trigger_id;
	`, env.ID, []byte(`{"url": "https://example.com"}`)).Scan(&tid)

	s.Require().NoError(err)

	due, err := functiontrigger.NewStore().DueTriggers(context.Background())

	s.Require().NoError(err)
	s.Require().NotEmpty(due)

	for _, tf := range due {
		if tf.ID == tid {
			s.Equal("Nightly rollup", tf.Description)
			return
		}
	}

	s.Fail("the due trigger was not returned")
}

func TestTriggerDescription(t *testing.T) {
	suite.Run(t, &TriggerDescriptionSuite{})
}
