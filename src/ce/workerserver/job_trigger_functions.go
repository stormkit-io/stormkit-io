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

// triggerRetryWindow is how long a trigger stays due after a run that never
// happened. The sweep runs every minute, so this is roughly five attempts.
const triggerRetryWindow = 5 * time.Minute

type FunctionTriggerMessage struct {
	ID      types.ID      `json:"id,string"`
	Payload []byte        `json:"payload"`
	Headers shttp.Headers `json:"headers"`
	Method  string        `json:"method"`
	URL     string        `json:"url"`
	// NextRunAt is the tick this run advances the trigger to once it succeeds.
	NextRunAt utils.Unix `json:"nextRunAt"`
	// ScheduledAt is the overdue tick this run was fired for. It bounds how long
	// a failing trigger is allowed to stay due.
	ScheduledAt utils.Unix `json:"scheduledAt"`
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
			URL:         opts.URL,
			Payload:     opts.Payload,
			Headers:     opts.Headers,
			Method:      opts.Method,
			ID:          tf.ID,
			NextRunAt:   utils.UnixFrom(nextRunAt),
			ScheduledAt: tf.NextRunAt,
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

		// The run never happened — the target was unreachable or still warming up.
		// Stay due for a short grace window so a cold start is picked up by the next
		// sweep, then fall through to the regular tick so one dead target cannot
		// re-fire every minute forever.
		if err != nil || log.ResponseCode() == http.StatusServiceUnavailable {
			retrying := time.Since(tf.ScheduledAt.Time) < triggerRetryWindow

			// A warming target is expected and sweeps once a minute, so it stays at
			// info; a transport failure is a genuine trigger error and keeps its
			// error level so log-based alerting still sees it.
			if err != nil {
				slog.Errorf("trigger function %d request failed, retrying: %t: %v", tf.ID, retrying, err)
			} else {
				slog.Infof("trigger function %d did not run, target responded with %d, retrying: %t", tf.ID, log.ResponseCode(), retrying)
			}

			if retrying {
				continue
			}
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
