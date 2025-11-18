package runner_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/runner"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/mise"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type InstallerSuite struct {
	suite.Suite
	mockService *mocks.MicroServiceInterface
	mockMise    *mocks.MiseInterface
	mockCmd     *mocks.CommandInterface
	config      runner.RunnerOpts
	tmpDir      string
}

func (s *InstallerSuite) BeforeTest(_, _ string) {
	tmpDir, err := os.MkdirTemp("", "tmp-test-runner-")

	s.NoError(err)

	s.config = runner.RunnerOpts{
		RootDir:  tmpDir,
		WorkDir:  path.Join(tmpDir, "repo"),
		Reporter: runner.NewReporter("http://example.com"),
		Build: runner.BuildOpts{
			EnvVarsRaw: []string{
				"CI=true",
			},
		},
		Repo: runner.RepoOpts{
			Dir: path.Join(tmpDir, "repo"),
		},
	}

	s.NoError(s.config.MkdirAll())

	s.tmpDir = tmpDir

	s.mockMise = &mocks.MiseInterface{}
	s.mockCmd = &mocks.CommandInterface{}
	s.mockService = &mocks.MicroServiceInterface{}
	mise.DefaultMise = s.mockMise
	sys.DefaultCommand = s.mockCmd
	rediscache.DefaultService = s.mockService
}

func (s *InstallerSuite) AfterTest(_, _ string) {
	if strings.Contains(s.config.RootDir, os.TempDir()) {
		s.config.RemoveAll()
	}

	s.config.Reporter.Close(nil, nil, nil)
	runner.DefaultInstaller = nil
	mise.DefaultMise = nil
	sys.DefaultCommand = nil
	rediscache.DefaultService = nil
}

func (s *InstallerSuite) packageJsonWithWorkspaces() []byte {
	buffer := new(bytes.Buffer)
	packageJson := `{
		"private": true,
		"workspaces": [
			"apps/*"
		],
		"dependencies": {
			"@stormkit/serverless": "2.0.8"
		}
	}`

	s.NoError(json.Compact(buffer, []byte(packageJson)))
	return buffer.Bytes()
}

func (s *InstallerSuite) packageJson() []byte {
	buffer := new(bytes.Buffer)
	packageJson := `{
		"dependencies": {
			"@stormkit/serverless": "2.0.8"
		}
	}`

	s.NoError(json.Compact(buffer, []byte(packageJson)))
	return buffer.Bytes()
}

func (s *InstallerSuite) Test_PackageJson_Unmarshal() {
	packageJson := `{ "bundleDependencies": ["@stormkit/serverless"] }`

	p := runner.PackageJson{}

	s.NoError(json.Unmarshal([]byte(packageJson), &p))
	s.Equal(runner.BundleDependencies{"@stormkit/serverless"}, p.BundleDependencies)

	packageJson = `{ "bundleDependencies": true }`

	p = runner.PackageJson{}

	s.NoError(json.Unmarshal([]byte(packageJson), &p))
	s.Equal(runner.BundleDependencies{"*"}, p.BundleDependencies)

	packageJson = `{ "bundleDependencies": false }`

	p = runner.PackageJson{}

	s.NoError(json.Unmarshal([]byte(packageJson), &p))
	s.Equal(runner.BundleDependencies{}, p.BundleDependencies)

	packageJson = `{ "bundledDependencies": true }`

	p = runner.PackageJson{}

	s.NoError(json.Unmarshal([]byte(packageJson), &p))
	s.Equal(runner.BundleDependencies{"*"}, p.BundledDependencies)
}

func (s *InstallerSuite) Test_Install_Yarn() {
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "package.json"), s.packageJsonWithWorkspaces(), 0776))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "yarn.lock"), []byte{}, 0776))

	s.config.Repo.PackageJson = &runner.PackageJson{
		Workspaces: []string{"apps/*"},
	}

	s.config.Repo.IsYarn = true

	p := runner.NewInstaller(s.config)

	// Fetch version
	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name: "yarn",
		Args: []string{"--version"},
		Dir:  s.config.WorkDir,
		Env:  s.config.Build.EnvVarsRaw,
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Output").Return([]byte("1.22.19\n"), nil).Once()

	// Set workspaces-experimental to true for yarn v1
	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name:   "yarn",
		Args:   []string{"config", "set", "workspaces-experimental", "true"},
		Dir:    s.config.WorkDir,
		Env:    s.config.Build.EnvVarsRaw,
		Stdout: s.config.Reporter.File(),
		Stderr: s.config.Reporter.File(),
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Run").Return(nil, nil).Once()

	// Install
	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name:   "yarn",
		Args:   []string{"--production=false"},
		Dir:    s.config.Repo.Dir,
		Env:    s.config.Build.EnvVarsRaw,
		Stdout: s.config.Reporter.File(),
		Stderr: s.config.Reporter.File(),
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Run").Return(nil, nil).Once()

	s.NoError(p.Install(context.Background()))

	lines := []string{
		"[sk-step] enable yarn workspaces",
		"yarn",
	}

	logs := s.config.Reporter.Logs()

	for _, line := range lines {
		s.Contains(logs, line)
	}
}

func (s *InstallerSuite) Test_Install_Npm() {
	s.config.Repo.PackageJson = &runner.PackageJson{}
	s.config.Repo.PackageLockFile = true
	s.config.Repo.IsNpm = true

	p := runner.NewInstaller(s.config)

	commands := [][]string{
		{"echo", "-n", "registry: "},
		{"npm", "config", "get", "registry"},
		{"npm", "ci", "--no-audit", "--include=dev"},
	}

	for _, cmd := range commands {
		s.mockCmd.On("SetOpts", sys.CommandOpts{
			Name:   cmd[0],
			Args:   cmd[1:],
			Dir:    s.config.Repo.Dir,
			Env:    s.config.Build.EnvVarsRaw,
			Stdout: s.config.Reporter.File(),
			Stderr: s.config.Reporter.File(),
		}).Return(s.mockCmd).Once()

		s.mockCmd.On("Run").Return(nil, nil).Once()
	}

	s.NoError(p.Install(context.Background()))

	lines := []string{
		"[sk-step] npm ci",
	}

	logs := s.config.Reporter.Logs()

	for _, line := range lines {
		s.Contains(logs, line)
	}
}

func (s *InstallerSuite) Test_Install_Npm_Windows() {
	config.IsWindows = true
	defer func() { config.IsWindows = false }()

	s.config.Repo.PackageJson = &runner.PackageJson{}
	s.config.Repo.PackageLockFile = true
	s.config.Repo.IsNpm = true

	p := runner.NewInstaller(s.config)

	commands := [][]string{
		{"powershell.exe", "-NoProfile", "-Command", "Write-Host -NoNewline 'registry: '"},
		{"npm", "config", "get", "registry"},
		{"npm", "ci", "--no-audit", "--include=dev"},
	}

	for _, cmd := range commands {
		s.mockCmd.On("SetOpts", sys.CommandOpts{
			Name:   cmd[0],
			Args:   cmd[1:],
			Dir:    s.config.Repo.Dir,
			Env:    s.config.Build.EnvVarsRaw,
			Stdout: s.config.Reporter.File(),
			Stderr: s.config.Reporter.File(),
		}).Return(s.mockCmd).Once()

		s.mockCmd.On("Run").Return(nil, nil).Once()
	}

	s.NoError(p.Install(context.Background()))

	lines := []string{
		"[sk-step] npm ci",
	}

	logs := s.config.Reporter.Logs()

	for _, line := range lines {
		s.Contains(logs, line)
	}
}

func (s *InstallerSuite) Test_Install_Pnpm() {
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "package.json"), s.packageJson(), 0776))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "pnpm-lock.yaml"), []byte{}, 0776))

	s.config.Repo.PackageJson = &runner.PackageJson{}
	s.config.Repo.IsPnpm = true

	p := runner.NewInstaller(s.config)

	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name:   "pnpm",
		Args:   []string{"install"},
		Dir:    s.config.Repo.Dir,
		Env:    s.config.Build.EnvVarsRaw,
		Stdout: s.config.Reporter.File(),
		Stderr: s.config.Reporter.File(),
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Run").Return(nil, nil).Once()

	s.NoError(p.Install(context.Background()))

	lines := []string{
		"[sk-step] pnpm install",
		"pnpm install",
	}

	logs := s.config.Reporter.Logs()

	for _, line := range lines {
		s.Contains(logs, line)
	}
}

func (s *InstallerSuite) Test_Install_Bun() {
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "package.json"), s.packageJson(), 0776))
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "bun.lockb"), []byte{}, 0776))

	s.config.Repo.PackageJson = &runner.PackageJson{}
	s.config.Repo.IsBun = true

	p := runner.NewInstaller(s.config)

	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name:   "bun",
		Args:   []string{"install"},
		Dir:    s.config.Repo.Dir,
		Env:    s.config.Build.EnvVarsRaw,
		Stdout: s.config.Reporter.File(),
		Stderr: s.config.Reporter.File(),
	}).Return(s.mockCmd).Once()

	s.mockCmd.On("Run").Return(nil, nil).Once()

	s.NoError(p.Install(context.Background()))

	lines := []string{
		"[sk-step] bun install",
		"bun install",
	}

	logs := s.config.Reporter.Logs()

	for _, line := range lines {
		s.Contains(logs, line)
	}
}

func (s *InstallerSuite) TestInstall_Custom() {
	s.NoError(os.WriteFile(path.Join(s.config.Repo.Dir, "package.json"), s.packageJson(), 0776))

	s.config.Build.InstallCmd = "npm install"

	p := runner.NewInstaller(s.config)

	ctx := context.Background()
	s.mockCmd.On("SetOpts", sys.CommandOpts{
		Name:   "sh",
		Args:   []string{"-c", "npm install"},
		Dir:    s.config.Repo.Dir,
		Env:    s.config.Build.EnvVarsRaw,
		Stdout: s.config.Reporter.File(),
		Stderr: s.config.Reporter.File(),
	}).Return(s.mockCmd).Once()
	s.mockCmd.On("Run").Return(nil, nil).Once()

	s.NoError(p.Install(ctx))

	lines := []string{
		"[sk-step] npm install",
		"npm install",
	}

	logs := s.config.Reporter.Logs()

	for _, line := range lines {
		s.Contains(logs, line)
	}
}

func (s *InstallerSuite) Test_Runtimes() {
	type Runtime struct {
		ExpectedMessage       string
		DependencyFile        string
		DependencyFileContent string
	}

	runtimes := map[string]Runtime{
		"nodejs22.x": {
			ExpectedMessage:       "node --version",
			DependencyFile:        "package.json",
			DependencyFileContent: `{"name": "test", "engines": {"node": ">=18.0.0"}}`,
		},
		"go1.20": {
			ExpectedMessage:       "go version",
			DependencyFile:        "go.mod",
			DependencyFileContent: `module test`,
		},
	}

	for _, expected := range runtimes {
		if expected.DependencyFile == "package.json" {
			s.config.Repo.PackageJson = &runner.PackageJson{}
		} else {
			s.config.Repo.PackageJson = nil
		}

		r := runner.NewInstaller(s.config)

		s.mockCmd.On("SetOpts", sys.CommandOpts{
			Name:   strings.Split(expected.ExpectedMessage, " ")[0],
			Args:   strings.Split(expected.ExpectedMessage, " ")[1:],
			Dir:    s.config.Repo.Dir,
			Env:    s.config.Build.EnvVarsRaw,
			Stdout: s.config.Reporter.File(),
			Stderr: s.config.Reporter.File(),
		}).Return(s.mockCmd).Once()

		s.mockCmd.On("Run").Return(nil, nil).Once()

		s.NoError(r.RuntimeVersion(context.Background()))
	}
}

func (s *InstallerSuite) Test_InstallingRuntimeDeps() {
	ctx := context.Background()
	workDir := path.Join(s.config.Repo.Dir, "my-dir")
	stdout := s.config.Reporter.File()

	// Prepare files: these are required to trigger the installation
	s.NoError(os.Mkdir(workDir, 0776))
	s.NoError(os.WriteFile(path.Join(workDir, "mise.toml"), []byte(`[tools]\ngo = "1.24"`), 0776))

	s.mockMise.On("InstallMise", ctx).Return(nil).Once()
	s.mockMise.On("InstallLocal", ctx, mise.LocalOpts{Dir: workDir, Stdout: stdout, Stderr: stdout}).Return(nil).Once()
	s.mockMise.On("ListLocal", ctx, mise.LocalOpts{Dir: workDir}).Return([]string{"go@1.24"}, nil).Once()

	s.config.WorkDir = workDir
	s.config.Repo.Runtime = "go"

	p := runner.NewInstaller(s.config)

	installed, err := p.InstallRuntimeDependencies(ctx)
	s.Equal([]string{"go@1.24"}, installed)
	s.NoError(err)
	s.Contains(s.config.Reporter.Logs(), "[sk-step] mise install")

	// Now let's try returning no runtimes installed
	s.mockMise.On("InstallMise", ctx).Return(nil).Once()
	s.mockMise.On("InstallLocal", ctx, mise.LocalOpts{Dir: workDir, Stdout: stdout, Stderr: stdout}).Return(nil).Once()
	s.mockMise.On("ListLocal", ctx, mise.LocalOpts{Dir: workDir}).Return(nil, nil).Once()

	// Since InstallLocal returns empty runtimes, we expect another InstallLocal with the runtime
	s.mockMise.On("InstallLocal", ctx, mise.LocalOpts{Dir: workDir, Stdout: stdout, Stderr: stdout, Runtime: "go"}).Return(nil).Once()

	p = runner.NewInstaller(s.config)

	installed, err = p.InstallRuntimeDependencies(ctx)
	s.Equal([]string{"go"}, installed)
	s.NoError(err)
	s.Contains(s.config.Reporter.Logs(), "[sk-step] mise install")
}

func TestInstallerSuite(t *testing.T) {
	suite.Run(t, &InstallerSuite{})
}
