package runner

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

// CloudMaxRepoSize is the default maximum on-disk size a checkout may reach on
// Stormkit Cloud, counting the working tree and the .git directory together
// since both consume the build host's disk.
const CloudMaxRepoSize int64 = 1 << 30 // 1GB

// MaxRepoSizeEnvVar overrides the checkout cap, in megabytes. It exists so an
// oversized customer can be unblocked by reconfiguring the build host rather
// than by cutting a runner release.
const MaxRepoSizeEnvVar = config.MaxRepoSizeEnvVar

// ErrRepoTooLarge is returned when a repository busts the size cap. The
// message is surfaced verbatim in the deployment logs.
type ErrRepoTooLarge struct {
	// size is the measured size of the repository, or 0 when the download was
	// stopped by the kernel before it could be measured.
	size  int64
	limit int64
}

func (e ErrRepoTooLarge) Error() string {
	if e.size <= 0 {
		//lint:ignore ST1005 This message is being consumed by the frontend
		return fmt.Sprintf(
			"Repository is larger than allowed.\n"+
				"The download exceeded the %s limit.\n"+
				"Reduce the size of the repository, or move large files out of git.",
			humanBytes(e.limit),
		)
	}

	//lint:ignore ST1005 This message is being consumed by the frontend
	return fmt.Sprintf(
		"Repository is larger than allowed.\n"+
			"The repository is %s while the limit is %s.\n"+
			"Reduce the size of the repository, or move large files out of git.",
		humanBytes(e.size), humanBytes(e.limit),
	)
}

// humanBytes formats a byte count for deployment logs, switching to GB once
// MB stops being readable. The uploader's megabytes() stays as-is because its
// limits are genuinely MB-scale.
func humanBytes(b int64) string {
	if b >= 1<<30 {
		return fmt.Sprintf("%.1fGB", float64(b)/(1<<30))
	}

	return megabytes(b)
}

// repoSizer measures a repository that has been cloned with --no-checkout,
// before its working tree is written to disk. Both halves are knowable at that
// point: .git is already on disk, and the size of every file the checkout will
// produce is recorded in the objects git just downloaded.
type repoSizer struct {
	dir  string
	vars map[string]string
}

// gitDirSize sums what the clone has already written under .git.
func (r repoSizer) gitDirSize() int64 {
	var total int64

	//nolint:errcheck // a partially walked tree is still a usable lower bound
	filepath.WalkDir(path.Join(r.dir, ".git"), func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		if info, err := d.Info(); err == nil {
			total += info.Size()
		}

		return nil
	})

	return total
}

// workTreeSize returns the exact number of bytes the checkout will write, read
// out of the downloaded objects rather than measured after the fact. Every
// blob reachable from HEAD is listed with its size, so a repository made of
// many small files is accounted for the same as one made of a few large ones.
func (r repoSizer) workTreeSize(ctx context.Context) (int64, error) {
	cmd := sys.Command(ctx, sys.CommandOpts{
		Name: "git",
		Args: []string{"ls-tree", "-r", "-l", "HEAD"},
		Dir:  r.dir,
		Env:  PrepareEnvVars(r.vars),
	})

	// Streamed rather than buffered: the repositories this cap exists to
	// reject are exactly the ones with enough paths for the full listing to
	// be a large allocation on an already constrained host.
	exe := cmd.Cmd()

	if exe == nil {
		return 0, fmt.Errorf("could not measure the repository: no command")
	}

	stdout, err := exe.StdoutPipe()

	if err != nil {
		return 0, fmt.Errorf("could not measure the repository: %w", err)
	}

	if err := exe.Start(); err != nil {
		return 0, fmt.Errorf("could not measure the repository: %w", err)
	}

	var total int64

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1<<20)

	for scanner.Scan() {
		// <mode> <type> <sha> <size>\t<path>; size is "-" for anything that is
		// not a blob, which -r should already have excluded.
		fields := bytes.Fields(scanner.Bytes())

		if len(fields) < 4 {
			continue
		}

		size, err := strconv.ParseInt(string(fields[3]), 10, 64)

		if err != nil {
			continue
		}

		total += size
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("could not measure the repository: %w", err)
	}

	if err := exe.Wait(); err != nil {
		return 0, fmt.Errorf("could not measure the repository: %w", err)
	}

	return total, nil
}

// total returns the disk the repository will occupy once checked out.
func (r repoSizer) total(ctx context.Context) (int64, error) {
	workTree, err := r.workTreeSize(ctx)

	if err != nil {
		return 0, err
	}

	return r.gitDirSize() + workTree, nil
}

// maxRepoSize returns the checkout size cap for the current edition. Only
// Stormkit Cloud is capped; self-hosted instances own their own disk and are
// left unlimited.
func maxRepoSize() int64 {
	if !config.IsStormkitCloud() {
		return 0
	}

	override := os.Getenv(MaxRepoSizeEnvVar)

	if override == "" {
		return CloudMaxRepoSize
	}

	mb, err := strconv.ParseInt(override, 10, 64)

	if err != nil || mb <= 0 {
		return CloudMaxRepoSize
	}

	return mb << 20
}
