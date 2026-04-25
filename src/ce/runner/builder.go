package runner

import (
	"context"
	_ "embed"
	"fmt"
	"path/filepath"

	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

// nixFlakeDir returns the directory that contains flake.nix, or an empty string
// if no flake.nix is found. It checks workDir first (covers SK_CWD subdirectory
// builds), then repoDir (repo root).
func nixFlakeDir(workDir, repoDir string) string {
	for _, dir := range []string{workDir, repoDir} {
		if dir != "" && file.Exists(filepath.Join(dir, "flake.nix")) {
			return dir
		}
	}

	return ""
}

type BuilderInterface interface {
	ExecCommands(context.Context) error
	BuildApiIfNecessary(context.Context) (bool, error)
}

type Builder struct {
	cmd         string
	repoDir     string
	workDir     string
	distDir     string
	apiDir      string // Path to api dir
	packageMngr string
	envVars map[string]string
	reporter    *ReporterModel
}

// For testing purposes
var DefaultBuilder BuilderInterface

func NewBuilder(opts RunnerOpts) BuilderInterface {
	if DefaultBuilder != nil {
		return DefaultBuilder
	}

	bm := Builder{
		repoDir:     opts.Repo.Dir,
		workDir:     opts.WorkDir,
		packageMngr: opts.PackageManager,
		cmd:         opts.Build.BuildCmd,
		distDir:     opts.Build.DistFolder,
		apiDir:      opts.Build.APIFolder,
		envVars: opts.Build.EnvVars,
		reporter:    opts.Reporter,
	}

	if bm.cmd == "" && opts.Repo.PackageJson != nil && opts.Repo.PackageJson.Scripts["build"] != "" {
		if opts.Repo.IsBun {
			bm.cmd = "bun run build"
		} else if opts.Repo.IsYarn {
			bm.cmd = "yarn build"
		} else if opts.Repo.IsPnpm {
			bm.cmd = "pnpm build"
		} else {
			bm.cmd = "npm run build"
		}
	}

	return bm
}

func (bm Builder) ExecCommands(ctx context.Context) error {
	if bm.cmd == "" {
		return nil
	}

	bm.reporter.AddStep(bm.cmd)

	opts := sys.CommandOpts{
		Env:    PrepareEnvVars(bm.envVars),
		Dir:    bm.workDir,
		Stdout: bm.reporter.File(),
		Stderr: bm.reporter.File(),
	}

	if flakeDir := nixFlakeDir(bm.workDir, bm.repoDir); flakeDir != "" {
		opts.Name = "nix"
		opts.Dir = flakeDir
		opts.Args = []string{"--extra-experimental-features", "nix-command flakes", "develop", "--command", "sh", "-c", bm.cmd}
	} else {
		opts.Name = "sh"
		opts.Args = []string{"-c", bm.cmd}
	}

	return sys.Command(ctx, opts).Run()
}

func (bm Builder) BuildApiIfNecessary(ctx context.Context) (bool, error) {
	if bm.envVars["SK_BUILD_API"] == "off" || bm.apiDir == "" {
		return false, nil
	}

	if !file.Exists(filepath.Join(bm.workDir, bm.apiDir)) {
		return false, nil
	}

	bm.reporter.AddStep("build api")
	bm.reporter.AddLine(fmt.Sprintf("We found `%s` dir. We'll try to build it automatically.", bm.apiDir))
	bm.reporter.AddLine("You can turn off automatic api builds by specifying the following environment variable: `SK_BUILD_API=off`\n")

	bundler := NewAPIBuilder(ctx, APIBuilderOpts{
		WorkDir:        bm.workDir,
		APIDir:         bm.apiDir,
		OutputDir:      filepath.Join(".stormkit", "api"),
		PackageManager: bm.packageMngr,
		EnvVarsMap: bm.envVars,
		Reporter:       bm.reporter,
	})

	return true, bundler.BuildAll()
}
