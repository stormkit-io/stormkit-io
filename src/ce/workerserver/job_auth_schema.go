package jobs

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type authSchemaTarget struct {
	EnvID types.ID
	Conf  buildconf.SchemaConf
}

// EnsureAuthSchemas reconciles the per-tenant auth tables for every auth-enabled
// environment, adding columns (e.g. metadata) introduced after a tenant first
// enabled auth. It runs off the request path so sign-in and /me never trigger
// migration-role DDL themselves.
//
// It is a backfill for tenants that enabled auth before a column was introduced;
// new tenants and auth-config edits are reconciled inline by saveProvider. Each
// call is idempotent — SchemaConf.EnsureAuthSchema no-ops on schemas already
// reconciled this process — so the recurring run converges to near-zero work.
// Per-env failures are logged and skipped so one broken schema never blocks the
// batch (and, being uncached, retry on the next run).
func EnsureAuthSchemas(ctx context.Context) error {
	store := NewStore()

	targets, err := collectAuthSchemaTargets(ctx, store)

	if err != nil {
		return err
	}

	for _, t := range targets {
		if err := t.Conf.EnsureAuthSchema(ctx); err != nil {
			slog.Errorf("ensure auth schema for env %d: %v", t.EnvID, err)
			continue
		}
	}

	return nil
}

// collectAuthSchemaTargets loads and decodes every non-deleted environment that
// has auth configured and a provisioned schema, keeping only those with auth
// enabled. auth_conf is an opaque bytea blob, so the enabled check happens in Go
// rather than SQL. The primary-DB cursor is drained fully before any per-tenant
// connection is opened, mirroring dropStaleSchemas.
func collectAuthSchemaTargets(ctx context.Context, store *Store) ([]authSchemaTarget, error) {
	rows, err := store.Query(ctx, stmt.selectAuthEnabledEnvs)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var targets []authSchemaTarget

	for rows.Next() {
		var (
			envID     types.ID
			rawAuth   []byte
			rawSchema []byte
		)

		if err := rows.Scan(&envID, &rawAuth, &rawSchema); err != nil {
			return nil, err
		}

		var authConf buildconf.SKAuthConf

		if err := utils.ByteaScan(rawAuth, &authConf); err != nil {
			slog.Errorf("failed to decode auth_conf for env %d: %v", envID, err)
			continue
		}

		if !authConf.Status {
			continue
		}

		var schemaConf buildconf.SchemaConf

		if err := utils.ByteaScan(rawSchema, &schemaConf); err != nil {
			slog.Errorf("failed to decode schema_conf for env %d: %v", envID, err)
			continue
		}

		targets = append(targets, authSchemaTarget{EnvID: envID, Conf: schemaConf})
	}

	return targets, rows.Err()
}
