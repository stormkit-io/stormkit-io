package runner

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

type APIBuilderOpts struct {
	WorkDir        string
	APIDir         string
	OutputDir      string
	PackageManager string // "npm", "yarn", "pnpm"
	EnvVarsMap map[string]string
	Reporter       *ReporterModel
}

// APIBuilder handles the bundling of API files
type APIBuilder struct {
	options APIBuilderOpts
	ctx     context.Context
}

type installPath struct {
	path       string
	installArg string
}

// NewAPIBuilder creates a new APIBuilder instance
func NewAPIBuilder(ctx context.Context, options APIBuilderOpts) *APIBuilder {
	return &APIBuilder{
		options: options,
		ctx:     ctx,
	}
}

func (b *APIBuilder) InstallDependencies() error {
	// Install package.json in case we have some
	installPaths := []installPath{}
	apiDir := path.Join(b.options.WorkDir, b.options.APIDir)

	_ = filepath.WalkDir(apiDir, func(pathToFile string, info fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if path.Base(pathToFile) == "package.json" {
			ip := installPath{
				path:       path.Dir(pathToFile),
				installArg: "install",
			}

			if b.options.PackageManager == "npm" {
				if _, err := os.Stat(path.Join(ip.path, "package-lock.json")); err == nil {
					ip.installArg = "ci --include=dev"
				}
			}

			installPaths = append(installPaths, ip)
		}

		return nil
	})

	rep := b.options.Reporter

	// Install nested dependencies
	for _, ip := range installPaths {
		cmd := sys.Command(b.ctx, sys.CommandOpts{
			Env:    PrepareEnvVars(b.options.EnvVarsMap),
			Name:   b.options.PackageManager,
			Args:   strings.Split(ip.installArg, " "),
			Dir:    ip.path,
			Stdout: rep.File(),
			Stderr: rep.File(),
		})

		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}

// findAPIFiles discovers all API files in the specified directory
func (b *APIBuilder) findAPIFiles() ([]string, error) {
	var apiFiles []string

	apiDir := path.Join(b.options.WorkDir, b.options.APIDir)

	return apiFiles, filepath.WalkDir(apiDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Check for supported file extensions
		ext := strings.ToLower(filepath.Ext(path))
		allowed := []string{".js", ".cjs", ".ts", ".mjs", ".tsx", ".jsx"}

		if !utils.InSliceString(allowed, ext) {
			return nil
		}

		// Skip private files (starting with underscore)
		filename := filepath.Base(path)

		if strings.HasPrefix(filename, "_") {
			return nil
		}

		// Skip test and spec files
		if strings.Contains(filename, ".test.") || strings.Contains(filename, ".spec.") {
			return nil
		}

		apiFiles = append(apiFiles, path)

		return nil
	})
}

// generateEntryPoints creates entry points for esbuild
func (b *APIBuilder) generateEntryPoints(apiFiles []string) map[string]string {
	entryPoints := make(map[string]string)
	apiDir := path.Join(b.options.WorkDir, b.options.APIDir)

	for _, file := range apiFiles {
		// Get relative path from API directory
		relPath, err := filepath.Rel(apiDir, file)
		if err != nil {
			continue
		}

		// Remove extension and normalize path
		name := strings.TrimSuffix(relPath, filepath.Ext(relPath))
		name = strings.ReplaceAll(name, "\\", "/")

		entryPoints[name] = file
	}

	return entryPoints
}

// bundleFile bundles a single API file
func (b *APIBuilder) bundleFile(entryName string, entryPath string) error {
	// Ensure output directory exists
	outputDir := path.Join(b.options.WorkDir, b.options.OutputDir)
	outputFile := path.Join(outputDir, entryName+".mjs")

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	define := map[string]string{}

	for key, value := range b.options.EnvVarsMap {
		define["process.env."+key] = fmt.Sprintf("%q", value)
	}

	// Configure esbuild options
	buildOptions := api.BuildOptions{
		EntryPoints:       []string{entryPath},
		Outfile:           outputFile,
		Bundle:            true,
		Write:             true,
		MinifyWhitespace:  true,
		MinifyIdentifiers: true,
		MinifySyntax:      true,
		External:          []string{},
		Platform:          api.PlatformNode,
		Format:            api.FormatESModule,
		Target:            api.ES2022,
		Sourcemap:         api.SourceMapNone,
		LogLevel:          api.LogLevelSilent,
		AbsWorkingDir:     path.Join(b.options.WorkDir, b.options.APIDir),
		Define:            define,
		Banner: map[string]string{
			// Shim require if needed
			"js": `import module from 'module'; if (typeof globalThis.require === "undefined") { globalThis.require = module.createRequire(import.meta.url); }`,
		},
	}

	// Build the file
	result := api.Build(buildOptions)

	// Handle build errors
	if len(result.Errors) > 0 {
		var errorMessages []string

		for _, err := range result.Errors {
			errorMessages = append(errorMessages, err.Text)
		}

		return fmt.Errorf("build errors for %s: %s", entryName, strings.Join(errorMessages, "; "))
	}

	// Log warnings if any
	if len(result.Warnings) > 0 {
		for _, warning := range result.Warnings {
			log.Printf("Warning for %s: %s\n", entryName, warning.Text)
		}
	}

	return nil
}

// BuildAll builds all API files found in the API directory
func (b *APIBuilder) BuildAll() error {
	// Find all API files
	apiFiles, err := b.findAPIFiles()

	// No API files found, nothing to build
	if err != nil && os.IsNotExist(err) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to find API files: %w", err)
	}

	if len(apiFiles) == 0 {
		return nil
	}

	// Generate entry points
	entryPoints := b.generateEntryPoints(apiFiles)

	if len(entryPoints) == 0 {
		return nil
	}

	if err := b.InstallDependencies(); err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	for entryName, entryPath := range entryPoints {
		if err := b.bundleFile(entryName, entryPath); err != nil {
			return fmt.Errorf("failed to bundle %s: %w", entryName, err)
		}
	}

	return nil
}
