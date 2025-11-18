//go:build windows

package file

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

func envVars() []string {
	return []string{
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		fmt.Sprintf("USERPROFILE=%s", os.Getenv("USERPROFILE")),
	}
}

type ZipArgs struct {
	Source        []string
	ZipName       string
	WorkingDir    string
	IncludeParent bool
}

// ZipV2 the source folder/file to the target zip file.
// If the zip file already exists, this function will open and
// re-use that.
func ZipV2(args ZipArgs) error {
	for _, dirOrFile := range args.Source {
		absolutePath := filepath.Join(args.WorkingDir, dirOrFile)

		if _, err := os.Stat(absolutePath); os.IsNotExist(err) {
			continue
		}

		zipPath := filepath.Join(args.WorkingDir, args.ZipName)

		var command string
		var cmdArgs []string

		if !args.IncludeParent {
			absolutePath = filepath.Join(absolutePath, "*")
		}

		fmt.Println("ZIP_PATH", zipPath)
		fmt.Println("ABSOLUTE_PATH", absolutePath)
		fmt.Println("WORKING_DIR", args.WorkingDir)
		fmt.Println("ZIP_NAME", args.ZipName)

		// For single files
		command = "powershell.exe"
		cmdArgs = []string{
			"-NoProfile",
			"-Command",
			fmt.Sprintf(
				"Compress-Archive -Path '%s' -DestinationPath '%s' -Update -CompressionLevel Optimal",
				absolutePath,
				args.ZipName,
			),
		}

		cmd := sys.Command(context.Background(), sys.CommandOpts{
			Name:   command,
			Args:   cmdArgs,
			Dir:    args.WorkingDir,
			Stdout: io.Discard,
			Stderr: os.Stderr,
			Env:    envVars(),
		})

		fmt.Println("Running command:", cmd.String())

		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
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
// This function will force files and folders to be lowercase if requested.
func Unzip(opts UnzipOpts) error {
	// PowerShell's Expand-Archive doesn't support lowercase conversion
	// If lowercase is needed, we'll do it after extraction
	command := "powershell.exe"
	cmdArgs := []string{
		"-NoProfile",
		"-Command",
		fmt.Sprintf(
			"Expand-Archive -Path '%s' -DestinationPath '%s' -Force",
			opts.ZipFile,
			opts.ExtractDir,
		),
	}

	cmd := sys.Command(context.Background(), sys.CommandOpts{
		Name:   command,
		Args:   cmdArgs,
		Stdout: io.Discard,
		Stderr: os.Stderr,
	})

	if err := cmd.Run(); err != nil {
		return err
	}

	// If lowercase conversion is requested, rename all files and directories
	if opts.LowerCase {
		return lowercaseFiles(opts.ExtractDir)
	}

	return nil
}

// lowercaseFiles renames all files and directories to lowercase recursively
func lowercaseFiles(root string) error {
	// Walk in reverse order (deepest first) to handle nested structures
	var paths []string

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path != root {
			paths = append(paths, path)
		}
		return nil
	})

	if err != nil {
		return err
	}

	// Process paths in reverse order (deepest paths first)
	for i := len(paths) - 1; i >= 0; i-- {
		oldPath := paths[i]
		dir := filepath.Dir(oldPath)
		base := filepath.Base(oldPath)
		newBase := strings.ToLower(base)

		if base != newBase {
			newPath := filepath.Join(dir, newBase)
			if err := os.Rename(oldPath, newPath); err != nil {
				return fmt.Errorf("failed to rename %s to %s: %w", oldPath, newPath, err)
			}
		}
	}

	return nil
}
