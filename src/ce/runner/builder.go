package runner

import (
	"context"
	_ "embed"
	"fmt"
	"path/filepath"

	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

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
	envVars     map[string]string
	envVarsRaw  []string
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
		envVarsRaw:  opts.Build.EnvVarsRaw,
		envVars:     opts.Build.EnvVars,
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

	cmd := sys.Command(ctx, sys.CommandOpts{
		Name:   "sh",
		Args:   []string{"-c", bm.cmd},
		Env:    bm.envVarsRaw,
		Dir:    bm.workDir,
		Stdout: bm.reporter.File(),
		Stderr: bm.reporter.File(),
	})

	return cmd.Run()
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
		EnvVarsMap:     bm.envVars,
		EnvVarsSlice:   bm.envVarsRaw,
		Reporter:       bm.reporter,
	})

	return true, bundler.BuildAll()
}
