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

type TriggerFunctionModelSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *TriggerFunctionModelSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *TriggerFunctionModelSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

// This is a tech debt that we have to carry until all triggers have
// been migrated. Initially we used to store headers as a string in the db.
// Then we added shttp.Headers to support this use case. Using the Marshaler
// interface we do handle both cases. This spec tests that use case.
func (s *TriggerFunctionModelSuite) Test_FetchingStringHeadersFromDB() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	opts := `{ "headers": "name:joe;surname:doe", "url": "https://example.com" }`
	tid := types.ID(0)

	err := s.conn.QueryRow(`
		INSERT INTO
			function_triggers (env_id, trigger_status, trigger_options, cron)
		VALUES
			($1, $2, $3, $4)
		RETURNING
			trigger_id;
	`, env.ID, true, []byte(opts), "* * * * *").Scan(&tid)

	s.NoError(err)
	s.Greater(int(tid), 0)

	tf, err := functiontrigger.NewStore().ByID(context.Background(), tid)

	s.NoError(err)
	s.NotNil(tf)
	s.Equal(tf.Options.Headers.String(), "name:joe;surname:doe")
}

func TestHandlerTrigger(t *testing.T) {
	suite.Run(t, &TriggerFunctionModelSuite{})
}
