package jobs

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

func CleanupDeletedTeams(ctx context.Context) error {
	store := NewStore()

	_, err := store.Exec(ctx, stmt.markStaleAppsAndEnvsSoftDeleted)

	if err != nil {
		slog.Errorf("error while soft deleting team content: %v", err)
		return err
	}

	return nil
}
