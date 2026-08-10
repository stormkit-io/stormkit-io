package jobs

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/adhocore/gronx"
	"github.com/hibiken/asynq"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/tasks"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type FunctionTriggerMessage struct {
	ID        types.ID      `json:"id,string"`
	Payload   []byte        `json:"payload"`
	Headers   shttp.Headers `json:"headers"`
	Method    string        `json:"method"`
	URL       string        `json:"url"`
	NextRunAt utils.Unix    `json:"nextRunAt"`
}

// InvokeDueFunctionTriggers fetches function triggers from the database that are due date. Matching
// function triggers will be prepared and sent to the queue for execution.
func InvokeDueFunctionTriggers(ctx context.Context) error {
	tfs, err := functiontrigger.NewStore().DueTriggers(ctx)

	if err != nil {
		slog.Errorf("error while selecting due function trigger: %v", err)
		return err
	}

	messages := []FunctionTriggerMessage{}

	// Cache each environment's variables so a batch that shares an environment
	// resolves references with a single lookup. A nil entry is cached too, so a
	// missing/failed environment is not retried for every trigger.
	varsByEnv := map[types.ID]map[string]string{}

	for _, tf := range tfs {
		vars, ok := varsByEnv[tf.EnvID]

		if !ok {
			env, err := buildconf.NewStore().EnvironmentByID(ctx, tf.EnvID)

			if err != nil {
				slog.Errorf("error while fetching environment %s for trigger interpolation: %v", tf.EnvID, err)
			}

			if env != nil && env.Data != nil {
				vars = env.Data.Vars
			}

			varsByEnv[tf.EnvID] = vars
		}

		// Resolve $VAR references against the environment's own variables so
		// secrets live in the env config, not in the stored trigger.
		opts := tf.Options.Interpolate(vars)

		nextRunAt, err := gronx.NextTickAfter(tf.Cron, time.Now().UTC(), false)

		if err != nil {
			slog.Errorf("error while calculating next tick: %s", err.Error())
		}

		messages = append(messages, FunctionTriggerMessage{
			URL:       opts.URL,
			Payload:   opts.Payload,
			Headers:   opts.Headers,
			Method:    opts.Method,
			ID:        tf.ID,
			NextRunAt: utils.UnixFrom(nextRunAt),
		})
	}

	if len(messages) == 0 {
		return nil
	}

	if _, err := tasks.Enqueue(ctx, tasks.TriggerFunctionHttp, messages, nil); err != nil {
		slog.Errorf("error occurred while enqueuing task %s", err.Error())
		return err
	}

	return nil
}

// HandleFunctionTrigger handles triggering a function trigger.
func HandleFunctionTrigger(ctx context.Context, t *asynq.Task) error {
	tfs := []FunctionTriggerMessage{}

	if err := json.Unmarshal(t.Payload(), &tfs); err != nil {
		slog.Errorf("HandleTriggerFunction cannot unmarshal payload information: %v", err)
		return err
	}

	logs := []functiontrigger.TriggerLog{}
	updates := map[types.ID]utils.Unix{}

	for _, tf := range tfs {
		log, err := functiontrigger.Run(functiontrigger.RunParams{
			TriggerID: tf.ID,
			Method:    tf.Method,
			URL:       tf.URL,
			Headers:   tf.Headers,
			Payload:   tf.Payload,
		})

		logs = append(logs, log)

		if err != nil {
			slog.Errorf("trigger function request failed %v", err)
			continue
		}

		// A 503 means the target never got to run the job — it was warming up
		// or otherwise unavailable. Leave nextRunAt untouched so the next sweep
		// picks the trigger up again instead of skipping to the following tick.
		if log.ResponseCode() == http.StatusServiceUnavailable {
			slog.Errorf("trigger function %d unavailable, will retry on the next sweep", tf.ID)
			continue
		}

		updates[tf.ID] = tf.NextRunAt
	}

	store := functiontrigger.NewStore()

	if err := store.InsertLogs(ctx, logs); err != nil {
		slog.Errorf("error while inserting function trigger logs: %s", err.Error())
	}

	if err := store.SetNextRunAt(ctx, updates); err != nil {
		slog.Errorf("error while inserting function trigger batch updates: %s", err.Error())
	}

	return nil
}
