package jobs

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"go.uber.org/zap"
)

type staleSchema struct {
	EnvID      types.ID
	SchemaName string
}

// RemoveStaleEnvironments marks soft deleted environment's deployments soft deleted,
// drops the per-environment Postgres schema (and its roles) when one was provisioned,
// and removes volume files (physical bytes + DB rows) that belong to those environments.
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

	if err := removeStaleVolumes(ctx, store); err != nil {
		slog.Errorf("error while removing stale volumes: %v", err)
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

// removeStaleVolumes deletes volume files (physical bytes + DB rows) that belong
// to soft-deleted environments. App/environment deletion is soft-only, so the
// volumes_env_id_fkey ON DELETE CASCADE never fires; without this cleanup public
// files for deleted apps would keep being served and leak on disk/S3 forever.
// Per-env failures are logged and skipped so one broken volume backend does not
// block the rest of the batch; anything left behind retries on the next run.
func removeStaleVolumes(ctx context.Context, store *Store) error {
	cfg, err := admin.Store().Config(ctx)

	if err != nil {
		return err
	}

	if cfg.VolumesConfig == nil {
		return nil
	}

	rows, err := store.Query(ctx, stmt.selectStaleVolumeEnvs)

	if err != nil {
		return err
	}

	defer rows.Close()

	var envIDs []types.ID

	for rows.Next() {
		var envID types.ID

		if err := rows.Scan(&envID); err != nil {
			return err
		}

		envIDs = append(envIDs, envID)
	}

	if err := rows.Err(); err != nil {
		return err
	}

	for _, envID := range envIDs {
		if err := purgeEnvVolumes(ctx, purgeEnvVolumesParams{config: cfg.VolumesConfig, envID: envID}); err != nil {
			slog.Errorf("failed to purge volumes for env %d: %v", envID, err)
			continue
		}

		slog.Debug(slog.LogOpts{
			Msg:     "removed stale volumes",
			Level:   slog.DL1,
			Payload: []zap.Field{zap.Int64("env_id", int64(envID))},
		})
	}

	return nil
}

type purgeEnvVolumesParams struct {
	config *admin.VolumesConfig
	envID  types.ID
}

// purgeEnvVolumes removes every volume file of a single environment, one page at
// a time. Rows are only deleted from the database once their bytes are gone, so
// a failing physical delete surfaces as an error and the surviving rows retry on
// the next run rather than looping forever.
func purgeEnvVolumes(ctx context.Context, p purgeEnvVolumesParams) error {
	const batchSize = 100

	store := volumes.Store()

	for {
		files, err := store.SelectFiles(ctx, volumes.SelectFilesArgs{EnvID: p.envID, Limit: batchSize})

		if err != nil {
			return err
		}

		if len(files) == 0 {
			return nil
		}

		removed, removeErr := volumes.RemoveFiles(p.config, files)

		if len(removed) > 0 {
			if err := store.RemoveFiles(ctx, removed, p.envID); err != nil {
				return err
			}
		}

		if removeErr != nil {
			return removeErr
		}

		if len(files) < batchSize {
			return nil
		}
	}
}
