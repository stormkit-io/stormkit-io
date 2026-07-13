package jobs

import (
	"context"
)

// RemoveOldLogs removes logs older than 30 days.
func RemoveOldLogs(ctx context.Context) error {
	return NewStore().RemoveOldLogs(ctx)
}
