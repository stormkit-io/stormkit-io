package file

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

func envVars() []string {
	return []string{
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
	}
}

type ZipArgs struct {
	Source        []string // List of folders/files to zip (relative to WorkingDir)
	ZipName       string   // The target zip file name
	WorkingDir    string   // The absolute path to the working directory
	IncludeParent bool     // Whether to include the parent folder when zipping directories
	GlobPattern   string   // Optional: only include files matching this pattern (e.g., "*.sql")
}

// ZipV2 the source folder/file to the target zip file.
// If the zip file already exists, this function will open and
// re-use that.
func ZipV2(args ZipArgs) error {
	for _, dirOrFile := range args.Source {
		absolutePath := path.Join(args.WorkingDir, dirOrFile)
		info, err := os.Stat(absolutePath)

		if os.IsNotExist(err) {
			continue
		}

		if err != nil {
			slog.Errorf("error while zipping %s: %v", dirOrFile, err)
			continue
		}

		workingDir := args.WorkingDir
		isDir := info.IsDir()

		// When not including parent, cd into the directory
		if isDir && !args.IncludeParent {
			workingDir = absolutePath
		}

		cmd := exec.Command("sh", "-c", buildZipCommand(isDir, args, dirOrFile))
		cmd.Dir = workingDir
		cmd.Stdout = io.Discard
		cmd.Stderr = os.Stderr
		cmd.Env = envVars()

		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

func buildZipCommand(isDir bool, args ZipArgs, dirOrFile string) string {
	// -r recursive
	// -y preserve symlinks
	// -9 max compression

	// Single file - simple zip
	if !isDir {
		return fmt.Sprintf("zip -9 %s %s", args.ZipName, path.Base(dirOrFile))
	}

	// Directory with parent included
	if args.IncludeParent {
		baseCmd := fmt.Sprintf("zip -r -y -9 %s %s", args.ZipName, dirOrFile)
		if args.GlobPattern != "" {
			return fmt.Sprintf("%s -i '%s'", baseCmd, args.GlobPattern)
		}
		return baseCmd
	}

	// Directory without parent - use find
	findFilter := `\( -type f -o -type l \)`

	if args.GlobPattern != "" {
		findFilter += fmt.Sprintf(` -name '%s'`, args.GlobPattern)
	}

	return fmt.Sprintf(
		`files=$(find . %s -print) && [ -n "$files" ] && echo "$files" | zip -r -y -9 -@ %s || exit 0`,
		findFilter,
		args.ZipName,
	)
}

func IsZipEmpty(src string) bool {
	r, err := zip.OpenReader(src)

	if err != nil {
		return true
	}

	defer r.Close()
	return len(r.File) == 0
}

type UnzipOpts struct {
	ZipFile    string
	ExtractDir string
	LowerCase  bool
}

// Unzip the given `zip` file to the given `dest` destination.
// This function will force files and folders to be lowercase.
func Unzip(opts UnzipOpts) error {
	args := []string{}

	if opts.LowerCase {
		args = append(args, "-LL") // Force lowercase names
	}

	args = append(args, "-o", opts.ZipFile, "-d", opts.ExtractDir)

	cmd := sys.Command(context.Background(), sys.CommandOpts{
		Name:   "unzip",
		Args:   args,
		Stdout: io.Discard,
		Stderr: os.Stderr,
	})

	return cmd.Run()
}
