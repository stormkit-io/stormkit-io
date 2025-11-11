package mise

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"go.uber.org/zap"
)

var cachedClient *Mise
var mux sync.Mutex

type ListOutput struct {
	RequestedVersion string `json:"requested_version"`
	Installed        bool   `json:"installed"`
	Active           bool   `json:"active"`
	Version          string `json:"version"`
}

type MiseInterface interface {
	InstallMise(context.Context) error
	InstallGlobal(context.Context, string) (string, error)
	InstallLocal(context.Context, LocalOpts) error // Runs mise install in the specified directory
	ListLocal(context.Context, LocalOpts) ([]string, error)
	ListGlobal(context.Context) (map[string][]ListOutput, error)
	Version() (string, error)
	SelfUpdate(ctx context.Context) error
	Prune(ctx context.Context) error
}

var DefaultMise MiseInterface

type Mise struct {
	paths map[string]bool
}

func ResetCache() {
	mux.Lock()
	defer mux.Unlock()

	cachedClient = nil
}

func Client() MiseInterface {
	mux.Lock()
	defer mux.Unlock()

	if DefaultMise != nil {
		return DefaultMise
	}

	if cachedClient == nil {
		cachedClient = &Mise{
			paths: map[string]bool{},
		}
	}

	return cachedClient
}

// InstallMise installs mise if necessary.
func (m *Mise) InstallMise(ctx context.Context) error {
	// Check if mise is already installed
	cmd := sys.Command(ctx, sys.CommandOpts{
		Name: "mise",
		Args: []string{"--version"},
	})

	if err := cmd.Run(); err == nil {
		return nil
	}

	// Install mise using the provided command
	cmd = sys.Command(ctx, sys.CommandOpts{
		Name: "sh",
		Args: []string{
			"-c",
			`curl https://mise.run | sh && echo 'eval "$(~/.local/bin/mise activate bash)"' >> ~/.bashrc`,
		},
		Env: []string{
			"MISE_VERSION=v2025.10.1",
			"PATH=" + os.Getenv("PATH"),
			"HOME=" + os.Getenv("HOME"),
		},
	})

	err := cmd.Run()

	if err != nil {
		return err
	}

	// Update path so that it supports mise
	os.Setenv("PATH", fmt.Sprintf("%s:%s", os.Getenv("HOME")+"/.local/bin", os.Getenv("PATH")))

	for _, runtime := range []string{"go", "node", "ruby", "elixir", "python"} {
		cmd := sys.Command(ctx, sys.CommandOpts{
			Name: "mise",
			Args: []string{
				"settings", "add", "idiomatic_version_file_enable_tools", runtime,
			},
			Env: []string{
				"PATH=" + os.Getenv("PATH"),
				"HOME=" + os.Getenv("HOME"),
			},
		})

		if err := cmd.Run(); err != nil {
			slog.Errorf("error enabling idiomatic version file for %s: %v", runtime, err)
			continue
		}
	}

	return nil
}

// PHPInstaller installs PHP using herd-lite. We cannot use mise
// for PHP installations at the moment because it is very cumbersome
// to manage PHP extensions using mise.
func (m *Mise) PHPInstaller(ctx context.Context, rt string) error {
	pieces := strings.SplitN(rt, "@", 2)
	version := "8.4"

	if len(pieces) == 2 && pieces[1] != "latest" {
		version = pieces[1]
	}

	var osys string

	switch runtime.GOOS {
	case "linux":
		osys = "linux"
	case "darwin":
		osys = "macos"
	default:
		return fmt.Errorf("unsupported OS for PHP installation: %s", runtime.GOOS)
	}

	// PHP installation using mise is not yet supported
	// We're using https://laravel.com/docs/12.x/installation as an alternative installation method
	cmd := sys.Command(ctx, sys.CommandOpts{
		Name: "sh",
		Args: []string{
			"-c",
			fmt.Sprintf(`/bin/bash -c "$(curl -fsSL https://php.new/install/%s/%s)"`, osys, version),
		},
		Env: []string{
			"PATH=" + os.Getenv("PATH"),
			"HOME=" + os.Getenv("HOME"),
		},
	})

	if err := cmd.Run(); err != nil {
		return err
	}

	if err := m.updatePath(ctx, ""); err != nil {
		return err
	}

	return nil
}

// phpDir returns the path to the PHP binary installed via herd-lite.
func (m *Mise) phpDir() string {
	return path.Join(os.Getenv("HOME"), ".config", "herd-lite", "bin")
}

// Install installs the specified runtime using mise.
// It returns information on the installed runtime or an error if the installation fails.
func (m *Mise) InstallGlobal(ctx context.Context, runtime string) (string, error) {
	if strings.HasPrefix(runtime, "php") {
		return "", m.PHPInstaller(ctx, runtime)
	}

	// Install the runtime using mise
	cmd := sys.Command(ctx, sys.CommandOpts{
		Name: "mise",
		Args: []string{"use", "--global", runtime},
	})

	output, err := cmd.CombinedOutput()
	trimmed := strings.TrimSpace(string(output))

	if err != nil {
		return trimmed, fmt.Errorf("error installing runtime %s: %v", runtime, err)
	}

	if err := m.updatePath(ctx, ""); err != nil {
		return trimmed, err
	}

	return trimmed, nil
}

type LocalOpts struct {
	Runtime string // If specified, will install using "mise use <runtime>", otherwise mise install
	Dir     string
	Env     []string
	Stdout  io.Writer
	Stderr  io.Writer
}

// InstallInDir installs mise using the configuration in the specified directory.
func (m *Mise) InstallLocal(ctx context.Context, opts LocalOpts) error {
	var args string

	if opts.Runtime != "" {
		args = fmt.Sprintf("mise use -y %s", opts.Runtime)
	} else {
		args = "mise trust -q && mise install"
	}

	// Install the runtime using mise
	cmd := sys.Command(ctx, sys.CommandOpts{
		Name:   "sh",
		Args:   []string{"-c", args},
		Dir:    opts.Dir,
		Env:    opts.Env,
		Stdout: opts.Stdout,
		Stderr: opts.Stderr,
	})

	if err := cmd.Run(); err != nil {
		return err
	}

	if err := m.updatePath(ctx, opts.Dir); err != nil {
		return err
	}

	return nil
}

// ListLocal lists the installed runtimes in the specified directory.
// It returns a slice of strings representing the installed runtimes or an error if the listing fails.
// The runtimes are formatted as "runtime@version".
func (m *Mise) ListLocal(ctx context.Context, opts LocalOpts) ([]string, error) {
	cmd := sys.Command(ctx, sys.CommandOpts{
		Name:   "mise",
		Args:   []string{"ls", "-c", "--json"},
		Dir:    opts.Dir,
		Env:    opts.Env,
		Stdout: opts.Stdout,
		Stderr: opts.Stderr,
	})

	output, err := cmd.Output()

	if err != nil {
		return nil, err
	}

	data := map[string][]ListOutput{}

	if err := json.Unmarshal(output, &data); err != nil {
		return nil, err
	}

	requiredDependencies := []string{}

	for runtime, installedPackages := range data {
		for _, pckg := range installedPackages {
			requiredDependencies = append(requiredDependencies, fmt.Sprintf("%s@%s", runtime, pckg.RequestedVersion))
		}
	}

	return requiredDependencies, nil
}

// ListGlobal lists the globally installed runtimes using mise.
func (m *Mise) ListGlobal(ctx context.Context) (map[string][]ListOutput, error) {
	cmd := sys.Command(ctx, sys.CommandOpts{
		Name: "mise",
		Args: []string{"ls", "--global", "--json"},
		Dir:  os.Getenv("HOME"),
	})

	output, err := cmd.Output()

	if err != nil {
		return nil, err
	}

	data := map[string][]ListOutput{}

	if err := json.Unmarshal(output, &data); err != nil {
		return nil, err
	}

	// Check if PHP is installed
	phpCmd := sys.Command(ctx, sys.CommandOpts{
		Name: "php",
		Args: []string{"-r", "echo phpversion();"},
	})

	version, _ := phpCmd.Output()

	if version != nil {
		data["php"] = []ListOutput{{
			// We don't have the requested version info here
			// so we set it to the same as Version
			RequestedVersion: strings.TrimSpace(string(version)),
			Version:          strings.TrimSpace(string(version)),
			Installed:        true,
			Active:           true,
		}}
	}

	return data, nil
}

func (m *Mise) updatePath(ctx context.Context, dir string) error {
	cmd := sys.Command(ctx, sys.CommandOpts{
		Name: "mise",
		Args: []string{"bin-paths"},
		Dir:  dir,
	})

	data, err := cmd.Output()

	if err != nil {
		return err
	}

	paths := strings.Split(strings.TrimSpace(string(data)), "\n")
	newPaths := []string{}

	for _, path := range paths {
		if path == "" || m.paths[path] {
			continue
		}

		newPaths = append(newPaths, path)
		m.paths[path] = true
	}

	if len(newPaths) == 0 {
		return nil
	}

	newPath := strings.Join(newPaths, ":")
	oldPath := os.Getenv("PATH")
	phpPath := m.phpDir()

	// We are setting php path here even if it's not installed to make sure
	// that if it gets installed later, it's already in the PATH.
	os.Setenv("PATH", fmt.Sprintf("%s:%s:%s", newPath, oldPath, phpPath))

	return nil
}

// Version returns JSON output in the following format:
//
//	{
//	  "version": "2025.8.7 linux-x64 (2025-08-06)",
//	  "latest": "2025.8.7",
//	  "os": "linux",
//	  "arch": "x64",
//	  "build_time": "2025-08-06 12:08:12 +00:00"
//	}
type VersionOutput struct {
	Version   string `json:"version"`
	Latest    string `json:"latest"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	BuildTime string `json:"build_time"`
}

// Version retrieves the version of mise installed on the system.
func (m *Mise) Version() (string, error) {
	cmd := sys.Command(context.Background(), sys.CommandOpts{
		String: "mise version --json",
	})

	output, err := cmd.Output()

	if err != nil {
		return "", fmt.Errorf("error getting mise version: %v", err)
	}

	vo := VersionOutput{}

	if err := json.Unmarshal(output, &vo); err != nil {
		return "", fmt.Errorf("error parsing mise version output: %v", err)
	}

	return vo.Version, nil
}

// Version retrieves the version of mise installed on the system.
func (m *Mise) SelfUpdate(ctx context.Context) error {
	cmd := sys.Command(ctx, sys.CommandOpts{
		String: "mise self-update --yes",
	})

	output, err := cmd.Output()

	if err != nil {
		slog.Errorf("mise self-update output: %s, err: %s", string(output), err.Error())
		return err
	}

	return nil
}

// Prune removes unused mise installations and cleans up the environment.
func (m *Mise) Prune(ctx context.Context) error {
	cmd := sys.Command(ctx, sys.CommandOpts{
		String: "mise prune --yes",
		Dir:    os.Getenv("HOME"),
	})

	return cmd.Run()
}

// AutoUpdate is a job that triggers the automatic update of mise.
func AutoUpdate(ctx context.Context, payload ...string) {
	var version string
	var err error

	keyName := rediscache.Service().Key("mise_update")
	client := Client()

	slog.Debug(slog.LogOpts{
		Msg:   "running mise auto update job",
		Level: slog.DL1,
	})

	defer func() {
		if err != nil {
			rediscache.Client().Set(ctx, keyName, rediscache.StatusErr, time.Minute)
		} else {
			rediscache.Client().Set(ctx, keyName, rediscache.StatusOK, time.Minute)
		}
	}()

	// Make sure we have mise installed
	if err := client.InstallMise(ctx); err != nil {
		slog.Errorf("error installing mise: %v", err)
		return
	}

	// Trigger the mise update process
	if err = client.SelfUpdate(ctx); err != nil {
		return
	}

	version, err = client.Version()

	if err != nil {
		slog.Errorf("error getting mise version: %v", err)
		return
	}

	slog.Debug(slog.LogOpts{
		Msg:     "mise updated",
		Level:   slog.DL1,
		Payload: []zap.Field{zap.String("version", version)},
	})
}
