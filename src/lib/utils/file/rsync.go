//go:build !windows

package file

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

type RsyncArgs struct {
	Context     context.Context
	Source      string
	Destination string
	WorkDir     string
}

// Rsync copies files using the rsync command.
func Rsync(args RsyncArgs) error {
	cmd := sys.Command(args.Context, sys.CommandOpts{
		Name: "rsync",
		Args: []string{"-a", "-R", args.Source, args.Destination},
		Dir:  args.WorkDir,
	})

	return cmd.Run()
}
