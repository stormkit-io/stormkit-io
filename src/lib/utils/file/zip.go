package file

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

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

// ZipInMemory creates a zip archive in memory from the given files.
// The `files` map should have file names as keys and file contents as values.
func ZipInMemory(files map[string][]byte) ([]byte, error) {
	// Create a buffer to hold the zip content
	var buf bytes.Buffer

	// Create a new zip writer
	zipWriter := zip.NewWriter(&buf)

	for fileName, fileContent := range files {
		zf, err := zipWriter.Create(fileName)

		if err != nil {
			return nil, err
		}

		if _, err = zf.Write(fileContent); err != nil {
			return nil, err
		}
	}

	// Close the zip writer to finalize the zip content
	if err := zipWriter.Close(); err != nil {
		return nil, err
	}

	// Return the zip content as a byte slice
	return buf.Bytes(), nil

}

// ZipIterator allows iterating over the files in a zip content.
// The iterator function should return true to continue iteration, or false to stop.
// Files are processed in sorted order by name.
func ZipIterator(zipContent []byte, iterator func(string, []byte) error) error {
	r, err := zip.NewReader(bytes.NewReader(zipContent), int64(len(zipContent)))

	if err != nil {
		return err
	}

	// Sort files by name
	files := make([]*zip.File, len(r.File))

	copy(files, r.File)

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name < files[j].Name
	})

	for _, f := range files {
		rc, err := f.Open()

		if err != nil {
			return err
		}

		content, err := io.ReadAll(rc)
		closeErr := rc.Close()

		if err != nil {
			return err
		}

		if closeErr != nil {
			return closeErr
		}

		if err := iterator(f.Name, content); err != nil {
			return err
		}
	}

	return nil
}

// sanitizeZipName ensures the zip file name is safe for shell commands
func sanitizeZipName(zipName string) (string, error) {
	// Get the base name to prevent directory traversal
	base := filepath.Base(zipName)

	// Only allow alphanumeric, dash, underscore, and dot
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_.-]+$`, base)
	if err != nil {
		return "", fmt.Errorf("failed to validate zip name: %w", err)
	}

	if !matched || base == "" || base == "." || base == ".." {
		return "", fmt.Errorf("invalid zip file name: %s", zipName)
	}

	return base, nil
}

// sanitizeGlobPattern ensures the glob pattern is safe for shell commands
func sanitizeGlobPattern(pattern string) (string, error) {
	if pattern == "" {
		return "", nil
	}

	// Trim whitespace
	pattern = strings.TrimSpace(pattern)

	// Only allow alphanumeric, *, ?, dash, underscore, and dot
	// No path separators or shell metacharacters
	matched, err := regexp.MatchString(`^[a-zA-Z0-9*?._-]+$`, pattern)

	if err != nil {
		return "", fmt.Errorf("failed to validate glob pattern: %w", err)
	}

	if !matched {
		return "", fmt.Errorf("invalid characters in glob pattern: %s", pattern)
	}

	// Prevent patterns that could escape
	if strings.Contains(pattern, "..") {
		return "", errors.New("glob pattern cannot contain '..'")
	}

	return pattern, nil
}

// ZipV2 the source folder/file to the target zip file.
// If the zip file already exists, this function will open and
// re-use that.
func ZipV2(args ZipArgs) error {
	// Sanitize inputs to prevent command injection
	sanitizedZipName, err := sanitizeZipName(args.ZipName)

	if err != nil {
		return fmt.Errorf("invalid zip name: %w", err)
	}

	sanitizedPattern, err := sanitizeGlobPattern(args.GlobPattern)

	if err != nil {
		return fmt.Errorf("invalid glob pattern: %w", err)
	}

	// Create sanitized args copy
	sanitizedArgs := args
	sanitizedArgs.ZipName = sanitizedZipName
	sanitizedArgs.GlobPattern = sanitizedPattern

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

		cmd := exec.Command("sh", "-c", buildZipCommand(isDir, sanitizedArgs, dirOrFile))
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
