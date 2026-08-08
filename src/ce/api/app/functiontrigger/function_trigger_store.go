package functiontrigger

import (
	"context"
	"encoding/json"
	"strings"
	"text/template"
	"time"

	"github.com/adhocore/gronx"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var stmts = struct {
	selectTriggers    string
	selectTriggerLogs string
	deleteTrigger     string
	insertTrigger     string
	updateTrigger     string
	updateNextRunAt   string
	insertTriggerLogs string
}{
	selectTriggers: `
		SELECT
			trigger_id, env_id, cron, trigger_options,
			trigger_status, created_at, next_run_at, updated_at,
			COALESCE(documentation, '')
        FROM
			function_triggers
		WHERE
			{{ .where }}
		LIMIT
			1000;
	`,
	selectTriggerLogs: `
		SELECT
			request, response, created_at
		FROM
			function_trigger_logs
		WHERE
			trigger_id = $1
		ORDER BY
			ftl_id DESC
		LIMIT
			25;
	`,
	deleteTrigger: `
		DELETE FROM
			function_triggers
		WHERE
			trigger_id = $1;
	`,
	insertTrigger: `
		INSERT INTO function_triggers
		    (env_id, cron, next_run_at, trigger_options, trigger_status, documentation)
		VALUES
			($1, $2, $3, $4, $5, $6)
    	RETURNING
			trigger_id;
	`,
	updateTrigger: `
		UPDATE
			function_triggers
		SET
			cron = $1,
			trigger_options = $2,
			trigger_status = $3,
			next_run_at = $4,
			documentation = $5,
			updated_at = timezone('utc', now())
		WHERE
			trigger_id = $6;
	`,
	updateNextRunAt: `
		UPDATE
			function_triggers
		SET
			next_run_at = (v.next_run_at)::timestamp
		FROM
			(VALUES {{ generateValues 2 (len .) }}) AS v(trigger_id, next_run_at)
		WHERE
			(v.trigger_id)::integer = function_triggers.trigger_id;
	`,
	insertTriggerLogs: `
		INSERT INTO function_trigger_logs (
			trigger_id,
			request,
			response
		)
		VALUES
			{{ generateValues 3 (len .) }};
	`,
}

type Store struct {
	*database.Store
	selectStmt      *template.Template
	insertLogsStmt  *template.Template
	updateBatchStmt *template.Template
}

func NewStore() *Store {
	return &Store{
		Store: database.NewStore(),
		selectStmt: template.Must(
			template.New("select_triggers").
				Parse(stmts.selectTriggers),
		),
		updateBatchStmt: template.Must(
			template.New("batch_update_triggers").
				Funcs(template.FuncMap{"generateValues": utils.GenerateValues}).
				Parse(stmts.updateNextRunAt),
		),
		insertLogsStmt: template.Must(
			template.New("batch_insert_logs").
				Funcs(template.FuncMap{"generateValues": utils.GenerateValues}).
				Parse(stmts.insertTriggerLogs)),
	}
}

// List all function triggers that belong to the environment.
func (s *Store) List(ctx context.Context, envID types.ID) ([]*FunctionTrigger, error) {
	var qb strings.Builder

	data := map[string]any{
		"where": "env_id = $1",
	}

	if err := s.selectStmt.Execute(&qb, data); err != nil {
		return nil, err
	}

	return s.selectRows(ctx, qb.String(), envID)
}

// ByID returns the function trigger with the given ID.
func (s *Store) ByID(ctx context.Context, triggerID types.ID) (*FunctionTrigger, error) {
	var qb strings.Builder

	data := map[string]any{
		"where": "trigger_id = $1",
	}

	if err := s.selectStmt.Execute(&qb, data); err != nil {
		return nil, err
	}

	rows, err := s.selectRows(ctx, qb.String(), triggerID)

	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return nil, nil
	}

	return rows[0], nil
}

// DueTriggers return all function triggers that needs to be executed in the
// next batch.
func (s *Store) DueTriggers(ctx context.Context) ([]*FunctionTrigger, error) {
	var qb strings.Builder

	data := map[string]any{
		"where": "next_run_at < timezone('utc', now()) AND trigger_status = true",
	}

	if err := s.selectStmt.Execute(&qb, data); err != nil {
		return nil, err
	}

	return s.selectRows(ctx, qb.String())
}

// Insert the given trigger function into the database.
func (s *Store) Insert(ctx context.Context, ft *FunctionTrigger) error {
	opts, err := json.Marshal(ft.Options)

	if err != nil {
		slog.Errorf("error while marshaling function trigger options: %s", err.Error())
		return err
	}

	if ft.Status {
		nextRunAt, err := gronx.NextTickAfter(ft.Cron, time.Now().UTC(), true)

		if err == nil {
			ft.NextRunAt = utils.UnixFrom(nextRunAt)
		}
	}

	row, err := s.QueryRow(
		ctx,
		stmts.insertTrigger,
		ft.EnvID,
		ft.Cron,
		ft.NextRunAt,
		opts,
		ft.Status,
		ft.Documentation,
	)

	if err != nil {
		return err
	}

	if err := row.Scan(&ft.ID); err != nil {
		return err
	}

	return nil
}

// InsertLogs inserts given logs in a batch operation.
func (s *Store) InsertLogs(ctx context.Context, logs []TriggerLog) error {
	var qb strings.Builder

	if err := s.insertLogsStmt.Execute(&qb, logs); err != nil {
		slog.Errorf("error executing batch query template: %v", err)
		return err
	}

	params := []any{}

	for _, log := range logs {
		req, _ := json.Marshal(log.Request)
		res, _ := json.Marshal(log.Response)
		params = append(params, log.TriggerID, req, res)
	}

	_, err := s.Exec(ctx, qb.String(), params...)
	return err
}

// Update updates the given trigger function.
func (s *Store) Update(ctx context.Context, ft *FunctionTrigger) error {
	opts, err := json.Marshal(ft.Options)

	if err != nil {
		slog.Errorf("error while marshaling function trigger options: %s", err.Error())
		return err
	}

	if ft.Status {
		nextRunAt, err := gronx.NextTick(ft.Cron, true)

		if err == nil {
			ft.NextRunAt = utils.UnixFrom(nextRunAt)
		}
	} else {
		ft.NextRunAt = utils.Unix{Valid: false}
	}

	_, err = s.Exec(ctx, stmts.updateTrigger, ft.Cron, opts, ft.Status, ft.NextRunAt, ft.Documentation, ft.ID)
	return err
}

// SetNextRunAt is a batch operation to update the nextRunAt of the given trigger ids.
func (s *Store) SetNextRunAt(ctx context.Context, values map[types.ID]utils.Unix) error {
	if len(values) == 0 {
		return nil
	}

	var qb strings.Builder

	if err := s.updateBatchStmt.Execute(&qb, values); err != nil {
		slog.Errorf("error executing batch query template: %v", err)
		return err
	}

	params := []any{}

	for id, nextRunAt := range values {
		params = append(params, id, nextRunAt)
	}

	_, err := s.Exec(ctx, qb.String(), params...)

	return err
}

func (s *Store) Delete(ctx context.Context, id types.ID) error {
	_, err := s.Exec(ctx, stmts.deleteTrigger, id)
	return err
}

// Logs return the last 25 trigger logs for the given trigger ID.
func (s *Store) Logs(ctx context.Context, id types.ID) ([]TriggerLog, error) {
	rows, err := s.Query(ctx, stmts.selectTriggerLogs, id)

	if rows == nil || err != nil {
		return nil, err
	}

	defer rows.Close()

	logs := []TriggerLog{}

	for rows.Next() {
		tmp := TriggerLog{TriggerID: id}
		err := rows.Scan(&tmp.Request, &tmp.Response, &tmp.CreatedAt)

		if err != nil {
			slog.Errorf("error while scanning trigger log: %s", err.Error())
			continue
		}

		logs = append(logs, tmp)
	}

	return logs, nil
}

func (s *Store) selectRows(ctx context.Context, query string, args ...any) ([]*FunctionTrigger, error) {
	rows, err := s.Query(ctx, query, args...)

	if rows == nil || err != nil {
		return nil, err
	}

	defer rows.Close()

	tfs := []*FunctionTrigger{}

	for rows.Next() {
		tmp := &FunctionTrigger{}
		err := rows.Scan(
			&tmp.ID, &tmp.EnvID, &tmp.Cron,
			&tmp.Options, &tmp.Status, &tmp.CreatedAt,
			&tmp.NextRunAt, &tmp.UpdatedAt, &tmp.Documentation,
		)

		if err != nil {
			slog.Errorf("error while scanning trigger: %s", err.Error())
			continue
		}

		tfs = append(tfs, tmp)
	}

	return tfs, nil
}
