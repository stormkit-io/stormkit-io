package file

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

type RsyncArgs struct {
	Context     context.Context
	Source      string // Relative path
	Destination string // Relative path
	WorkDir     string // Absolute path
}

// Rsync copies files using rsync (Unix) or robocopy (Windows).
func Rsync(args RsyncArgs) error {
	if config.IsWindows {
		return rsyncWindows(args)
	}

	return rsyncUnix(args)
}

// rsyncWindows copies files using robocopy on Windows.
// robocopy flags:
// /E - Copy subdirectories, including empty ones
// /DCOPY:DAT - Copy directory timestamps, attributes, and security
// /R:0 - Retry 0 times on failed copies
// /W:0 - Wait 0 seconds between retries
func rsyncWindows(args RsyncArgs) error {
	sourceDir := filepath.Dir(args.Source)
	sourceFile := filepath.Base(args.Source)

	info, err := os.Stat(filepath.Join(args.WorkDir, args.Source))

	if err != nil {
		return err
	}

	cmdArgs := []string{
		filepath.Join(args.WorkDir, sourceDir),
		filepath.Join(args.WorkDir, args.Destination, sourceDir),
	}

	if info.IsDir() {
		cmdArgs = append(cmdArgs, "/E", "/DCOPY:DAT", "/R:0", "/W:0")
	} else {
		cmdArgs = append(cmdArgs, sourceFile, "/DCOPY:DAT", "/R:0", "/W:0")
	}

	cmd := sys.Command(args.Context, sys.CommandOpts{
		Name: "robocopy",
		Args: cmdArgs,
	})

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			// Exit codes 0-7 are considered successful for robocopy
			if exitCode < 8 {
				return nil
			}
		}

		return err
	}

	return nil
}

// rsyncUnix copies files using the rsync command on Unix systems.
func rsyncUnix(args RsyncArgs) error {
	cmd := sys.Command(args.Context, sys.CommandOpts{
		Name: "rsync",
		Args: []string{"-a", "-R", args.Source, args.Destination},
		Dir:  args.WorkDir,
	})

	return cmd.Run()
}
