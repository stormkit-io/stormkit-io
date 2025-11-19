//go:build windows

package file

import (
	"context"
	"path/filepath"

	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

type RsyncArgs struct {
	Context     context.Context
	Source      string
	Destination string
	WorkDir     string
}

// Rsync copies files using robocopy on Windows.
// robocopy flags:
// /E - Copy subdirectories, including empty ones
// /DCOPY:DAT - Copy directory timestamps, attributes, and security
// /R:0 - Retry 0 times on failed copies
// /W:0 - Wait 0 seconds between retries
func Rsync(args RsyncArgs) error {
	source := args.Source
	destination := args.Destination

	// robocopy expects directory paths, so if source is a file,
	// we need to extract the directory and filename
	sourceDir := filepath.Dir(source)
	sourceFile := filepath.Base(source)

	// If source is a directory (no extension or ends with separator),
	// use it directly
	if sourceFile == "." || sourceFile == source {
		sourceDir = source
		sourceFile = "*.*" // Copy all files
	}

	cmd := sys.Command(args.Context, sys.CommandOpts{
		Name: "robocopy",
		Args: []string{sourceDir, destination, sourceFile, "/E", "/DCOPY:DAT", "/R:0", "/W:0"},
		Dir:  args.WorkDir,
	})

	return cmd.Run()
}
