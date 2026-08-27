package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deployservice"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
)

const StatusChecksPending = "pending"
const StatusChecksSuccess = "complete"

const RuntimeNode = "node"
const RuntimeBun = "bun"

func normalizeEnvVars(vars map[string]string) map[string]string {
	envVars := map[string]string{}

	for key, value := range vars {
		value = strings.TrimSpace(value)

		if value == "" {
			continue
		}

		// Trim beginning and end quotes
		value = strings.Trim(value, "'")
		value = strings.Trim(value, `"`)

		// Replace escapted quotes
		value = strings.ReplaceAll(value, "\\\"", `"`)
		envVars[key] = value
	}

	return envVars
}

// Normalize applies post-decode adjustments to a DeploymentMessage before
// the runner consumes it. Exported so tests in runner_test can exercise the
// SK_CWD → WorkDir fallback without sitting in this package.
func Normalize(msg *deployservice.DeploymentMessage) *deployservice.DeploymentMessage {
	// Backwards compatibility: SK_CWD was the legacy way to configure the
	// build working directory before the dedicated `workDir` field existed.
	// Honor it only when workDir is unset so the typed field always wins.
	if msg.Build.WorkDir == "" {
		msg.Build.WorkDir = msg.Build.Vars["SK_CWD"]
	}

	msg.Build.Vars = normalizeEnvVars(msg.Build.Vars)
	delete(msg.Build.Vars, "SK_CWD")
	return msg
}

// PrepareEnvVars builds a []string env slice from the map, always using the
// current process PATH and HOME so mise path updates are picked up automatically.
func PrepareEnvVars(envVars map[string]string) []string {
	result := []string{
		"CI=true",
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
	}

	keys := make([]string, 0, len(envVars))

	for k := range envVars {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	for _, k := range keys {
		result = append(result, k+"="+envVars[k])
	}

	return result
}

// unmaskedEnvVars is the explicit allowlist of Stormkit system variables that
// keep their values in the build log. These are injected by Stormkit and are
// not secrets, so they are safe to show and useful for debugging. We use an
// explicit set rather than the SK_ prefix so a user-defined SK_-prefixed
// variable is still masked.
var unmaskedEnvVars = map[string]bool{
	"SK_APP_ID":         true,
	"SK_BRANCH_NAME":    true,
	"SK_COMMIT_SHA":     true,
	"SK_DEPLOYMENT_ID":  true,
	"SK_DEPLOYMENT_URL": true,
	"SK_ENV":            true,
	"SK_ENV_ID":         true,
	"SK_ENV_URL":        true,
	"STORMKIT":          true,
}

func printEnvVariables(opts RunnerOpts) error {
	vars := []string{}

	// Values are masked in the build log: we list the variable names so the log
	// still shows what is configured, but never the values — those are only
	// retrievable through the audited reveal endpoint. The Stormkit system
	// variables in unmaskedEnvVars are the only exception.
	for k, v := range opts.Build.EnvVars {
		if unmaskedEnvVars[k] {
			vars = append(vars, fmt.Sprintf("%s=%s", k, v))
		} else {
			vars = append(vars, fmt.Sprintf("%s=***************", k))
		}
	}

	sort.Strings(vars)

	opts.Reporter.AddStep("environment variables")

	for _, v := range vars {
		opts.Reporter.AddLine(v)
	}

	return nil
}

type Payload struct {
	BaseURL       string `json:"baseUrl"` // https://api.stormkit.io
	RootDir       string `json:"rootDir"`
	DeploymentID  string `json:"deploymentId"`
	DeploymentMsg string `json:"deploymentMsg"` // Encrypted
}

type RepoOpts struct {
	Dir             string
	Address         string
	Branch          string
	AccessToken     string
	PackageJson     *PackageJson
	PackageLockFile bool
	Runtime         string // node
	IsYarn          bool
	IsNpm           bool
	IsPnpm          bool
	IsBun           bool
}

func parsePackageJson(packageJsonPath string) *PackageJson {
	if !file.Exists(packageJsonPath) {
		return nil
	}

	data, err := os.ReadFile(packageJsonPath)

	if err != nil {
		slog.Errorf("could not read package.json: %v", err)
		return nil
	}

	p := &PackageJson{}

	if err := json.Unmarshal(data, p); err != nil {
		slog.Errorf("could not parse package.json: %v", err)
		return nil
	}

	// We are going to use BundleDependencies
	if len(p.BundledDependencies) > 0 {
		p.BundleDependencies = p.BundledDependencies
	}

	return p
}

type BuildOpts struct {
	BuildCmd         string
	InstallCmd       string
	ServerCmd        string
	HeadersFile      string
	RedirectsFile    string
	APIFolder        string            // Relative path to the API folder (trimmed)
	DistFolder       string            // Relative path to the distribution folder (trimmed)
	EnvVars          map[string]string // Normalized environment variables
	DeploymentID     string
	AppID            string
	EnvID            string
	MigrationsFolder string
	StatusChecks     []buildconf.StatusCheck
	CacheDirs        []string // Directories (relative to WorkDir) restored before install and snapshotted after a successful build; empty disables caching
}

type RunnerOpts struct {
	RootDir        string // Absolute path to the root directory
	KeysDir        string // Absolute path to access tokens that we'll store dynamically
	WorkDir        string // Absolute path to the working directory
	PackageManager string // Package manager to use (npm, yarn, pnpm)
	Repo           RepoOpts
	Build          BuildOpts
	Uploader       *config.RunnerConfig
	AutoInstall    bool
	Reporter       *ReporterModel
}

func (o RunnerOpts) MkdirAll() error {
	for _, dir := range []string{o.RootDir, o.KeysDir, o.Repo.Dir} {
		if dir == "" {
			continue
		}

		if err := os.MkdirAll(dir, 0776); err != nil {
			return err
		}
	}

	return nil
}

func (o RunnerOpts) RemoveAll() error {
	if o.RootDir != "" {
		return os.RemoveAll(o.RootDir)
	}

	return nil
}

func Start(payload, rootDir string) error {
	p := Payload{}

	if err := json.Unmarshal([]byte(payload), &p); err != nil {
		return err
	}

	// If the --root-dir flag is provided, override the payload value
	if rootDir != "" {
		p.RootDir = rootDir
	}

	msg, err := deployservice.FromEncrypted(p.DeploymentMsg)

	if err != nil {
		return err
	}

	DeploymentIDEnc = utils.EncryptID(utils.StringToID(p.DeploymentID))

	msg = Normalize(msg)

	if msg.Config == nil {
		msg.Config = &deployservice.RunnerSettings{AutoInstall: true}
	}

	repoDir := path.Join(p.RootDir, "repo")
	workDir := path.Join(repoDir, msg.Build.WorkDir)

	opts := RunnerOpts{
		RootDir:        p.RootDir,
		WorkDir:        workDir,
		KeysDir:        path.Join(p.RootDir, "keys"),
		PackageManager: "", // will be determined later
		Reporter:       NewReporter(strings.Split(strings.TrimSuffix(p.BaseURL, "/"), "?")[0]),
		Repo: RepoOpts{
			Dir:         repoDir,
			Address:     msg.Client.Repo,
			Branch:      msg.Build.Branch,
			AccessToken: msg.Client.AccessToken,
			PackageJson: nil, // will be determined later
		},
		Build: BuildOpts{
			DeploymentID:     p.DeploymentID,
			AppID:            msg.Build.AppID,
			EnvID:            msg.Build.EnvID,
			BuildCmd:         msg.Build.BuildCmd,
			InstallCmd:       msg.Build.InstallCmd,
			ServerCmd:        msg.Build.ServerCmd,
			HeadersFile:      trim(msg.Build.HeadersFile),
			RedirectsFile:    trim(msg.Build.RedirectsFile),
			APIFolder:        trim(msg.Build.APIFolder),
			DistFolder:       trim(msg.Build.DistFolder),
			MigrationsFolder: trim(msg.Build.MigrationsFolder),
			StatusChecks:     msg.Build.StatusChecks,
			EnvVars:          msg.Build.Vars,
			CacheDirs:        msg.Build.CacheDirs,
		},
		Uploader:    &msg.Config.RunnerConfig,
		AutoInstall: msg.Config.AutoInstall,
	}

	if opts.Build.EnvVars == nil {
		opts.Build.EnvVars = make(map[string]string)
	}

	if err := opts.MkdirAll(); err != nil {
		return err
	}

	defer func(opts RunnerOpts) {
		if err := opts.RemoveAll(); err != nil {
			slog.Errorf("could not remove root dir: %v", err)
		}
	}(opts)

	result := Run(opts)

	if result != nil && result.err != nil {
		opts.Reporter.AddLine(result.err.Error())
	}

	if result != nil {
		PostRun(context.Background(), result)
	}

	return nil
}

// Run runs the runner for the the given deployment message
func Run(opts RunnerOpts) *RunResult {
	var err error
	var manifest *deploy.BuildManifest
	var result *integrations.UploadResult

	ctx := context.Background()

	slog.Infof("reporting back to %s", opts.Reporter.CallbackURL)

	repo := NewRepo(opts)

	var artifacts *Artifacts
	var miseOutput []string
	var uploadErr error

	if err := repo.Checkout(ctx); err != nil {
		return &RunResult{opts: opts, err: err}
	}

	// Now that we checked out, parse package.json if it exists
	opts.Repo.PackageJson = parsePackageJson(path.Join(opts.WorkDir, "package.json"))

	if file.Exists(path.Join(opts.WorkDir, "bun.lockb")) ||
		file.Exists(path.Join(opts.WorkDir, "bun.lock")) {
		opts.Repo.IsBun = true
		opts.PackageManager = "bun"
		opts.Repo.Runtime = RuntimeBun
	} else if file.Exists(path.Join(opts.WorkDir, "package-lock.json")) {
		opts.Repo.PackageLockFile = true
		opts.Repo.IsNpm = true
		opts.Repo.Runtime = RuntimeNode
		opts.PackageManager = "npm"
	} else if file.Exists(path.Join(opts.WorkDir, "yarn.lock")) {
		opts.Repo.IsYarn = true
		opts.PackageManager = "yarn"
		opts.Repo.Runtime = RuntimeNode
	} else if file.Exists(path.Join(opts.WorkDir, "pnpm-lock.yaml")) {
		opts.Repo.IsPnpm = true
		opts.PackageManager = "pnpm"
		opts.Repo.Runtime = RuntimeNode
	} else if opts.Repo.PackageJson != nil {
		opts.PackageManager = "npm"
		opts.Repo.Runtime = RuntimeNode
	}

	if err := opts.Reporter.SendCommitInfo(repo.CommitInfo()); err != nil {
		return &RunResult{opts: opts, err: err}
	}

	// Start sending the logs now (we first need to wait for commit info)
	opts.Reporter.SendLogs()

	// Now that the repo is checked out, create the package manager
	installer := NewInstaller(opts)

	if miseOutput, err = installer.InstallRuntimeDependencies(ctx); err != nil {
		return &RunResult{opts: opts, err: err}
	}

	if err := installer.RuntimeVersion(ctx); err != nil {
		return &RunResult{opts: opts, err: err}
	}

	if err := printEnvVariables(opts); err != nil {
		return &RunResult{opts: opts, err: err}
	}

	cache := newCacheManager(opts)
	cache.Restore(ctx)

	if err := installer.Install(ctx); err != nil {
		return &RunResult{opts: opts, err: err}
	}

	builder := NewBuilder(opts)

	if err := builder.ExecCommands(ctx); err != nil {
		return &RunResult{opts: opts, err: err}
	}

	isAPIAutoBuilt, err := builder.BuildApiIfNecessary(ctx)

	if err != nil {
		return &RunResult{opts: opts, err: err}
	}

	bundler := NewBundler(opts)

	if artifacts, err = bundler.Bundle(ctx); err != nil {
		return &RunResult{opts: opts, err: err}
	}

	artifacts.isAPIAutoBuilt = isAPIAutoBuilt

	if err := bundler.ParseRedirects(artifacts); err != nil {
		return &RunResult{opts: opts, err: err}
	}

	if err := bundler.ParseHeaders(artifacts); err != nil {
		return &RunResult{opts: opts, err: err}
	}

	if err := bundler.Zip(artifacts); err != nil {
		return &RunResult{opts: opts, err: err}
	}

	cache.Snapshot(ctx)

	opts.Reporter.AddStep("[system] building finished")

	manifest = &deploy.BuildManifest{
		Success:  err == nil,
		Runtimes: miseOutput,
	}

	if artifacts != nil {
		manifest.Redirects = artifacts.Redirects
		manifest.FunctionHandler = artifacts.FunctionHandler
		manifest.APIHandler = artifacts.ApiHandler
		manifest.CDNFiles = artifacts.CDNFiles()
		manifest.APIFiles = artifacts.APIFiles()

		opts.Reporter.AddStep("deploy")

		result, err = NewUploader(opts.Uploader).Upload(UploadArgs{
			MigrationsZip: artifacts.migrationsZip,
			ClientZip:     artifacts.clientZip,
			ServerZip:     artifacts.serverZip,
			ApiZip:        artifacts.apiZip,
			ServerHandler: artifacts.FunctionHandler,
			ApiHandler:    artifacts.ApiHandler,
			EnvVars:       opts.Build.EnvVars,
			Runtime:       GetRuntimeStringForLambdas(opts.Repo.Runtime, miseOutput),
			DeploymentID:  utils.StringToID(opts.Build.DeploymentID),
			AppID:         utils.StringToID(opts.Build.AppID),
			EnvID:         utils.StringToID(opts.Build.EnvID),
		})

		if err != nil {
			slog.Errorf("upload failed: %v", err)
			manifest.Success = false

			// The error has to travel back to the reporter, otherwise the
			// deploy step is rendered as failed with an empty message and
			// the user has no way of telling what went wrong.
			uploadErr = err
		}
	}

	return &RunResult{opts: opts, result: result, manifest: manifest, err: uploadErr}
}

// GetRuntimeStringForLambdas returns the runtime string for the uploader based on
// the given runtime and mise output.
func GetRuntimeStringForLambdas(runtime string, miseOutput []string) string {
	// No runtime detected
	if len(miseOutput) == 0 {
		return ""
	}

	// node@24.10
	for _, runtimeWithVersion := range miseOutput {
		pieces := strings.SplitN(runtimeWithVersion, "@", 2)

		if len(pieces) != 2 {
			continue
		}

		name, version := pieces[0], pieces[1]

		if name == runtime {
			major, _, _ := utils.ParseSemver(version)

			switch runtime {
			case RuntimeNode:
				return fmt.Sprintf("nodejs%s.x", major)
			}
		}
	}

	return ""
}

type RunResult struct {
	opts     RunnerOpts
	result   *integrations.UploadResult
	manifest *deploy.BuildManifest
	err      error
}

// PostRun executes commands right after the deployment is complete such as:
// - Run status Checks
// - Send last logs
// - Send exit code
// - Close reporter
func PostRun(ctx context.Context, args *RunResult) {
	statusChecksLen := len(args.opts.Build.StatusChecks)
	hasStatusChecks := statusChecksLen > 0

	// If we can't send the exit code after the exponential backoff,
	// we should kill the deployment. Also, we need to send the exit
	// code before the status checks because maybe they need to use
	// the deployment URL to run E2E tests.
	if err := args.opts.Reporter.SendExit(args.manifest, args.result, hasStatusChecks, args.err); err != nil {
		log.Fatalf("cannot update deployment status: %s", err.Error())
	}

	if args.result != nil && hasStatusChecks {
		checks := NewStatusChecks(args.opts)
		success := true

		for _, check := range args.opts.Build.StatusChecks {
			if err := checks.Run(ctx, check.Cmd); err != nil {
				success = false
				break
			}
		}

		// Locking deployment is only necessary when there are status checks
		// because if a deployment has no status checks, the exit callback will
		// lock it automatically.
		if err := args.opts.Reporter.LockDeployment(success); err != nil {
			log.Fatalf("cannot lock deployment: %s", err.Error())
		}
	}

	args.opts.Reporter.AddStep("[system] deployment finished")
	args.opts.Reporter.Close(args.manifest, args.result, args.err)
}
