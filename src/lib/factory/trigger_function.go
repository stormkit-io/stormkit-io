package factory

import (
	"encoding/json"
	"fmt"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type MockFunctionTrigger struct {
	*functiontrigger.FunctionTrigger
	*Factory
}

func (tf MockFunctionTrigger) Insert(conn databasetest.TestDB) error {
	opts, err := json.Marshal(tf.Options)

	if err != nil {
		panic(err)
	}

	insertQuery := `
		INSERT INTO skitapi.function_triggers
			(env_id, cron, next_run_at, trigger_options, trigger_status, updated_at, created_at)
		VALUES
			($1, $2, $3, $4, $5, $6, $7)
		RETURNING
			trigger_id;
	`

	return conn.PrepareOrPanic(insertQuery).QueryRow(
		tf.EnvID,
		tf.Cron,
		tf.NextRunAt,
		opts,
		tf.Status,
		tf.UpdatedAt,
		tf.CreatedAt,
	).Scan(&tf.ID)
}

func (f *Factory) MockTriggerFunction(env *MockEnv, overwrites ...map[string]any) *MockFunctionTrigger {
	if env == nil {
		env = f.GetEnv()
	}

	tf := &functiontrigger.FunctionTrigger{
		Cron:      "*/1 * * * *",
		EnvID:     env.ID,
		NextRunAt: utils.NewUnix(),
		CreatedAt: utils.NewUnix(),
		Options:   functiontrigger.Options{},
		Status:    true,
	}

	for _, o := range overwrites {
		merge(tf, o)
	}

	mock := f.newObject(MockFunctionTrigger{
		FunctionTrigger: tf,
		Factory:         f,
	}).(MockFunctionTrigger)

	err := mock.Insert(f.conn)

	if err != nil {
		fmt.Printf("Error inserting Triggerfunction %s", err.Error())
	}

	return &mock
}
