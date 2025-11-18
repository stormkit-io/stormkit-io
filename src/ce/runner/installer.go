package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/mise"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
)

type BundleDependencies []string

func (w *BundleDependencies) UnmarshalJSON(data []byte) (err error) {
	if string(data) == "true" {
		*w = []string{"*"}
		return nil
	} else if string(data) == "false" {
		*w = []string{}
		return nil
	}

	arr := []string{}

	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	*w = BundleDependencies(arr)

	return nil
}

type PackageJson struct {
	Name                string             `json:"name"`
	Version             string             `json:"version"`
	Workspaces          []string           `json:"workspaces,omitempty"`
	Scripts             map[string]string  `json:"scripts,omitempty"`
	Dependencies        map[string]string  `json:"dependencies"`
	DevDependencies     map[string]string  `json:"devDependencies,omitempty"`
	PeerDependencies    map[string]string  `json:"peerDependencies,omitempty"`
	BundleDependencies  BundleDependencies `json:"bundleDependencies"`            // We are going to use this one internally
	BundledDependencies BundleDependencies `json:"bundledDependencies,omitempty"` // This is an alternative syntax
}

func (pck *PackageJson) Write(fullPath string) error {
	data, err := json.Marshal(pck)

	if err != nil {
		return err
	}

	return os.WriteFile(fullPath, data, 0664)
}

type InstallerInterface interface {
	InstallRuntimeDependencies(context.Context) ([]string, error)
	Install(context.Context) error
	RuntimeVersion(context.Context) error
}

type Installer struct {
	installCmd         string
	packageJson        *PackageJson
	reporter           *ReporterModel
	isBun              bool
	isYarn             bool
	isPnpm             bool
	workDir            string
	buildCmd           string
	hasPackageLockFile bool
	runtime            string // The runtime that is going to be used to build the project
	envVars            []string
}

// For testing purposes
var DefaultInstaller InstallerInterface

func NewInstaller(opts RunnerOpts) InstallerInterface {
	if DefaultInstaller != nil {
		return DefaultInstaller
	}

	p := &Installer{
		workDir:            opts.WorkDir,
		buildCmd:           opts.Build.BuildCmd,
		installCmd:         opts.Build.InstallCmd,
		envVars:            opts.Build.EnvVarsRaw,
		reporter:           opts.Reporter,
		packageJson:        opts.Repo.PackageJson,
		hasPackageLockFile: opts.Repo.PackageLockFile,
		isPnpm:             opts.Repo.IsPnpm,
		isYarn:             opts.Repo.IsYarn,
		isBun:              opts.Repo.IsBun,
		runtime:            opts.Repo.Runtime,
	}

	return p
}

func (r *Installer) hasGoMod() bool {
	_, err := os.Stat(path.Join(r.workDir, "go.mod"))
	return err == nil
}

// RuntimeVersion prints the runtime version to the logs.
func (p Installer) RuntimeVersion(ctx context.Context) error {
	runtimeCmd := ""
	versionArg := "--version"

	if p.packageJson != nil {
		if p.isBun {
			runtimeCmd = "bun"
		} else {
			runtimeCmd = "node"
		}
	} else if p.hasGoMod() {
		runtimeCmd = "go"
		versionArg = "version"
	}

	if runtimeCmd == "" {
		return nil
	}

	p.reporter.AddStep(fmt.Sprintf("%s %s", runtimeCmd, versionArg))

	return sys.Command(ctx, sys.CommandOpts{
		Name:   runtimeCmd,
		Args:   []string{versionArg},
		Dir:    p.workDir,
		Env:    p.envVars,
		Stdout: p.reporter.File(),
		Stderr: p.reporter.File(),
	}).Run()
}

// InstallRuntimeDependencies installs runtime dependencies using mise.
func (p *Installer) InstallRuntimeDependencies(ctx context.Context) ([]string, error) {
	p.reporter.AddStep("mise install")

	m := mise.Client()

	// Make sure runner has mise installed
	if err := m.InstallMise(ctx); err != nil {
		return nil, err
	}

	opts := mise.LocalOpts{
		Dir:    p.workDir,
		Stdout: p.reporter.File(),
		Stderr: p.reporter.File(),
	}

	err := m.InstallLocal(ctx, opts)

	if err != nil {
		return nil, err
	}

	runtimes, err := m.ListLocal(ctx, mise.LocalOpts{
		Dir: p.workDir,
	})

	if err != nil {
		return nil, err
	}

	// If not found, let's install it manually
	found := false

	for _, rt := range runtimes {
		pieces := strings.Split(rt, "@")

		if pieces[0] == p.runtime {
			found = true
			break
		}
	}

	if !found && p.runtime != "" {
		opts.Runtime = p.runtime

		if err := m.InstallLocal(ctx, opts); err != nil {
			return nil, err
		}

		runtimes = append(runtimes, p.runtime)
	}

	// Make sure to update the PATH
	if len(p.envVars) > 1 {
		p.envVars[1] = fmt.Sprintf("PATH=%s", os.Getenv("PATH"))
	}

	return runtimes, nil
}

// Install runs the install command on the repository to install the dependencies.
func (p *Installer) Install(ctx context.Context) error {
	if p.installCmd != "" {
		return p.installCustom(ctx)
	}

	// noop
	if p.packageJson == nil {
		return nil
	}

	if p.isBun {
		return p.installBun(ctx)
	}

	if p.isYarn {
		return p.installYarn(ctx)
	}

	if p.isPnpm {
		return p.installPnpm(ctx)
	}

	// Install NPM
	return p.installNpm(ctx)
}

func (p *Installer) installCustom(ctx context.Context) error {
	p.reporter.AddStep(p.installCmd)

	cmd := sys.Command(ctx, sys.CommandOpts{
		Name:   "sh",
		Args:   []string{"-c", p.installCmd},
		Dir:    p.workDir,
		Env:    p.envVars,
		Stdout: p.reporter.File(),
		Stderr: p.reporter.File(),
	})

	return cmd.Run()
}

func (p *Installer) installBun(ctx context.Context) error {
	p.reporter.AddStep("bun install")

	return sys.Command(ctx, sys.CommandOpts{
		Name:   "bun",
		Args:   []string{"install"},
		Dir:    p.workDir,
		Env:    p.envVars,
		Stdout: p.reporter.File(),
		Stderr: p.reporter.File(),
	}).Run()
}

func (p *Installer) installPnpm(ctx context.Context) error {
	p.reporter.AddStep("pnpm install")

	return sys.Command(ctx, sys.CommandOpts{
		Name:   "pnpm",
		Args:   []string{"install"},
		Dir:    p.workDir,
		Env:    p.envVars,
		Stdout: p.reporter.File(),
		Stderr: p.reporter.File(),
	}).Run()
}

type Version struct {
	Major string
}

func (p *Installer) yarnVersion() Version {
	cmd := sys.Command(context.Background(), sys.CommandOpts{
		Name: "yarn",
		Args: []string{"--version"},
		Env:  p.envVars,
		Dir:  p.workDir,
	})

	out, err := cmd.Output()

	if err != nil {
		return Version{}
	}

	ver := strings.Replace(strings.Replace(string(out), "\n", "", -1), "v", "", 1)
	pieces := strings.Split(ver, ".")

	return Version{
		Major: pieces[0],
	}
}

func (p *Installer) installYarn(ctx context.Context) error {
	version := p.yarnVersion()
	file := p.reporter.File()

	if version.Major == "1" && len(p.packageJson.Workspaces) > 0 {
		p.reporter.AddStep("enable yarn workspaces")

		cmd := sys.Command(ctx, sys.CommandOpts{
			Name:   "yarn",
			Args:   []string{"config", "set", "workspaces-experimental", "true"},
			Env:    p.envVars,
			Dir:    p.workDir,
			Stdout: file,
			Stderr: file,
		})

		if err := cmd.Run(); err != nil {
			return err
		}
	}

	p.reporter.AddStep("yarn")

	opts := sys.CommandOpts{
		Name:   "yarn",
		Env:    p.envVars,
		Dir:    p.workDir,
		Stdout: file,
		Stderr: file,
	}

	if version.Major == "1" {
		opts.Args = []string{"--production=false"}
	}

	return sys.Command(ctx, opts).Run()
}

func (p *Installer) installNpm(ctx context.Context) error {
	installCmd := "install"

	if p.hasPackageLockFile {
		installCmd = "ci"
	}

	p.reporter.AddStep(fmt.Sprintf("npm %s", installCmd))

	printCmd := []string{"echo", "-n", "registry: "}

	if config.IsWindows {
		printCmd = []string{"powershell.exe", "-NoProfile", "-Command", "Write-Host -NoNewline 'registry: '"}
	}

	cmds := [][]string{
		printCmd,
		{"npm", "config", "get", "registry"},
		{"npm", installCmd, "--no-audit", "--include=dev"},
	}

	for _, eval := range cmds {
		cmd := sys.Command(ctx, sys.CommandOpts{
			Name:   eval[0],
			Args:   eval[1:],
			Dir:    p.workDir,
			Env:    p.envVars,
			Stdout: p.reporter.File(),
			Stderr: p.reporter.File(),
		})

		if err := cmd.Run(); err != nil {
			return err
		}
	}

	return nil
}
