package jobs

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"go.uber.org/zap"
)

type staleSchema struct {
	EnvID      types.ID
	SchemaName string
}

// RemoveStaleEnvironments marks soft deleted environment's deployments soft deleted and
// drops the per-environment Postgres schema (and its roles) when one was provisioned.
func RemoveStaleEnvironments(ctx context.Context) error {
	store := NewStore()

	// if environment is soft deleted, mark deployments belonging to those environments soft
	// deleted as well
	_, err := store.Exec(ctx, stmt.markDeploymentsSoftDeleted)

	if err != nil {
		slog.Errorf("error while marking deployments soft deleted %v", err)
		return err
	}

	if err := dropStaleSchemas(ctx, store); err != nil {
		slog.Errorf("error while dropping stale schemas: %v", err)
		return err
	}

	return nil
}

// dropStaleSchemas finds soft-deleted environments that still have a schema
// provisioned and drops the schema along with its app/migration roles.
// Per-env failures are logged and skipped so a single broken schema does not
// block the rest of the batch; schema_conf is cleared only after the drop
// succeeds, so transient failures retry on the next run.
func dropStaleSchemas(ctx context.Context, store *Store) error {
	rows, err := store.Query(ctx, stmt.selectStaleSchemas)

	if err != nil {
		return err
	}

	defer rows.Close()

	var stale []staleSchema

	for rows.Next() {
		var (
			envID   types.ID
			appID   types.ID
			rawConf []byte
		)

		if err := rows.Scan(&envID, &appID, &rawConf); err != nil {
			return err
		}

		schemaName := buildconf.SchemaName(appID, envID)

		var conf buildconf.SchemaConf

		if err := utils.ByteaScan(rawConf, &conf); err != nil {
			slog.Errorf("failed to decode schema_conf for env %d, falling back to derived name: %v", envID, err)
		} else if conf.SchemaName != "" {
			schemaName = conf.SchemaName
		}

		stale = append(stale, staleSchema{EnvID: envID, SchemaName: schemaName})
	}

	if err := rows.Err(); err != nil {
		return err
	}

	for _, s := range stale {
		if err := buildconf.SchemaStore().DropSchema(ctx, s.SchemaName); err != nil {
			slog.Errorf("failed to drop schema %s for env %d: %v", s.SchemaName, s.EnvID, err)
			continue
		}

		if _, err := store.Exec(ctx, stmt.clearSchemaConf, s.EnvID); err != nil {
			slog.Errorf("failed to clear schema_conf for env %d: %v", s.EnvID, err)
			continue
		}

		slog.Debug(slog.LogOpts{
			Msg:   "dropped stale schema",
			Level: slog.DL1,
			Payload: []zap.Field{
				zap.String("schema", s.SchemaName),
				zap.Int64("env_id", int64(s.EnvID)),
			},
		})
	}

	return nil
}
