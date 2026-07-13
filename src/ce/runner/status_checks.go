package runner

import (
	"context"
	"os"

	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

type StatusChecks struct {
	workDir     string
	repoDir     string
	envVars     map[string]string
	autoInstall bool
	reporter    *ReporterModel
}

func NewStatusChecks(opts RunnerOpts) *StatusChecks {
	return &StatusChecks{
		workDir:     opts.WorkDir,
		repoDir:     opts.Repo.Dir,
		envVars:     opts.Build.EnvVars,
		autoInstall: opts.AutoInstall,
		reporter:    opts.Reporter,
	}
}

// buildCommand returns the name and args to run the given command, wrapping it
// with nix develop when a flake.nix is present so that nix-provided tools and
// env vars (e.g. CHROMIUM_EXECUTABLE_PATH) are available. The current PATH is
// forwarded via env(1) so that mise-managed runtimes remain accessible.
func (s *StatusChecks) buildCommand(command string) (string, []string) {
	if s.autoInstall && nixFlakeDir(s.workDir, s.repoDir) != "" {
		return "nix", []string{
			"--extra-experimental-features", "nix-command flakes",
			"develop",
			"--command",
			"env", "PATH=" + os.Getenv("PATH"),
			"sh", "-c", command,
		}
	}

	return "sh", []string{"-c", command}
}

func (s *StatusChecks) Run(ctx context.Context, command string) error {
	rep := s.reporter
	rep.AddStep(command)

	name, args := s.buildCommand(command)

	cmd := sys.Command(ctx, sys.CommandOpts{
		Name:   name,
		Args:   args,
		Env:    PrepareEnvVars(s.envVars),
		Dir:    s.workDir,
		Stdout: rep.File(),
		Stderr: rep.File(),
	})

	err := cmd.Run()

	rep.CloseStep(err == nil)

	return err
}
