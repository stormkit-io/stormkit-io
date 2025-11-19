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
	Source      string
	Destination string
	WorkDir     string
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
	source := args.Source
	destination := filepath.Join(args.WorkDir, args.Destination)

	info, err := os.Stat(filepath.Join(args.WorkDir, source))
	if err != nil {
		return err
	}

	if info.IsDir() {
		// Source is a directory - copy all contents
		source = "*.*"
	}

	cmd := sys.Command(args.Context, sys.CommandOpts{
		Name: "robocopy",
		Args: []string{args.WorkDir, destination, source, "/E", "/DCOPY:DAT", "/R:0", "/W:0"},
		Dir:  args.WorkDir,
	})

	err = cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			// Exit codes 0-7 are considered successful for robocopy
			if exitCode < 8 {
				return nil
			}
		}
	}

	return err
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
