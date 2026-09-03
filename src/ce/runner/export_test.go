package runner

import "context"

// DiskPreflightCheck exposes diskPreflight.check. The test lives in the
// external package because src/mocks imports this one.
func DiskPreflightCheck(ctx context.Context, dir string, reporter *ReporterModel) error {
	return diskPreflight{dir: dir, reporter: reporter}.check(ctx)
}

// DiskPreflightMinFreeBytes exposes diskPreflight.minFreeBytes.
func DiskPreflightMinFreeBytes() uint64 {
	return diskPreflight{}.minFreeBytes()
}

// HumanBytes exposes humanBytes.
func HumanBytes(b uint64) string {
	return humanBytes(b)
}
