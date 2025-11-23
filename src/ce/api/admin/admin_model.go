package admin

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/aws/smithy-go/middleware"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/mise"
	"go.uber.org/zap"
)

const (
	SIGNUP_MODE_ON       = "on"
	SIGNUP_MODE_OFF      = "off"
	SIGNUP_MODE_WAITLIST = "waitlist"
)

type mdwrs = []func(stack *middleware.Stack) error

type VolumesConfig struct {
	MountType   string `json:"mountType"`            // one of Volume* constants above
	AccessKey   string `json:"accessKey,omitempty"`  // optional: access key for s3 and oss like mount types
	SecretKey   string `json:"secretKey,omitempty"`  // optional: secret key for s3 and oss like mount types
	BucketName  string `json:"bucketName,omitempty"` // optional: bucket name for s3 and oss like mount types
	RootPath    string `json:"rootPath,omitempty"`   // optional: the mount path for filesys mount type
	Region      string `json:"region,omitempty"`     // optional: the bucket region
	Middlewares mdwrs  `json:"-"`                    // testing: used for testing purposes - ignored in production
}

type WorkerserverConfig struct {
	DomainPingInterval    int `json:"domainPingInterval"`    // The interval in minutes
	DomainPingConcurrency int `json:"domainPingConcurrency"` // The number of workers we want to spawn in parallel
}

type AdminUserConfig struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SystemConfig struct {
	AutoInstall bool     `json:"autoInstall"` // Whether to install runtimes automatically or not. Default is true.
	Runtimes    []string `json:"runtimes"`    // The list of runtimes to install in format <name>@<version>
}

type ProxyRule struct {
	Target  string            `json:"target"`            // The host to match (might include port)
	Headers map[string]string `json:"headers,omitempty"` // Optional headers to add to the request
}

type ProxyConfig struct {
	Rules map[string]*ProxyRule `json:"rules"`
}

type LicenseConfig struct {
	Key string `json:"key"`
}

type GithubConfig struct {
	Account      string
	ClientID     string
	ClientSecret string
	PrivateKey   string
	RunnerRepo   string
	RunnerToken  string
	AppID        int
}

type GitlabConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type BitbucketConfig struct {
	ClientID     string
	ClientSecret string
	DeployKey    string
}

type UserManagement struct {
	Whitelist  []string `json:"whitelist"`  // Whitelist of pre-approved email domains (e.g. "example.com")
	SignUpMode string   `json:"signUpMode"` // "on" | "off" | "waitlist" (default: on)
}

type AuthConfig struct {
	Github         GithubConfig    `json:"github,omitempty"`
	Gitlab         GitlabConfig    `json:"gitlab,omitempty"`
	Bitbucket      BitbucketConfig `json:"bitbucket,omitempty"`
	UserManagement UserManagement  `json:"userManagement,omitempty"`
}

type DomainConfig struct {
	API      string `json:"api,omitempty"`      // e.g. api.stormkit.io
	App      string `json:"app,omitempty"`      // e.g. app.stormkit.io
	Health   string `json:"health,omitempty"`   // e.g. health.stormkit.io
	Dev      string `json:"dev,omitempty"`      // e.g. stormkit.dev (uses wildcard certificate)
	Webhooks string `json:"webhooks,omitempty"` // e.g. webhooks.stormkit.io
}

type InstanceConfig struct {
	AdminUserConfig    *AdminUserConfig    `json:"adminUser"`
	VolumesConfig      *VolumesConfig      `json:"volumes"`
	WorkerserverConfig *WorkerserverConfig `json:"workerserver"`
	SystemConfig       *SystemConfig       `json:"systemDependencies"`
	ProxyConfig        *ProxyConfig        `json:"proxy,omitempty"`
	LicenseConfig      *LicenseConfig      `json:"license,omitempty"`
	AuthConfig         *AuthConfig         `json:"auth,omitempty"`
	DomainConfig       *DomainConfig       `json:"domains,omitempty"`
}

// Scan implements the sql.Scanner interface
func (c *InstanceConfig) Scan(val any) error {
	if val == nil {
		// Handle NULL value from database
		*c = InstanceConfig{}
		return nil
	}

	switch v := val.(type) {
	case []byte:
		// Parse JSON data from database
		json.Unmarshal(v, c)

	case string:
		// Parse JSON string from database
		json.Unmarshal([]byte(v), c)

	default:
		return fmt.Errorf("unsupported Scan type for InstanceConfig: %T", val)
	}

	if c.VolumesConfig != nil && c.VolumesConfig.AccessKey != "" {
		c.VolumesConfig.AccessKey = utils.DecryptToString(c.VolumesConfig.AccessKey)
	}

	if c.VolumesConfig != nil && c.VolumesConfig.SecretKey != "" {
		c.VolumesConfig.SecretKey = utils.DecryptToString(c.VolumesConfig.SecretKey)
	}

	if c.LicenseConfig != nil && c.LicenseConfig.Key != "" {
		c.LicenseConfig.Key = utils.DecryptToString(c.LicenseConfig.Key)
	}

	if c.AuthConfig != nil {
		c.AuthConfig.Github.ClientSecret = utils.DecryptToString(c.AuthConfig.Github.ClientSecret)
		c.AuthConfig.Github.PrivateKey = utils.DecryptToString(c.AuthConfig.Github.PrivateKey)
		c.AuthConfig.Github.RunnerToken = utils.DecryptToString(c.AuthConfig.Github.RunnerToken)
		c.AuthConfig.Gitlab.ClientSecret = utils.DecryptToString(c.AuthConfig.Gitlab.ClientSecret)
		c.AuthConfig.Bitbucket.ClientSecret = utils.DecryptToString(c.AuthConfig.Bitbucket.ClientSecret)
		c.AuthConfig.Bitbucket.DeployKey = utils.DecryptToString(c.AuthConfig.Bitbucket.DeployKey)
	}

	return nil
}

// Value implements the Sql Driver interface.
func (c InstanceConfig) Value() (driver.Value, error) {
	if c.VolumesConfig != nil && c.VolumesConfig.AccessKey != "" {
		c.VolumesConfig.AccessKey = utils.EncryptToString(c.VolumesConfig.AccessKey)
	}

	if c.VolumesConfig != nil && c.VolumesConfig.SecretKey != "" {
		c.VolumesConfig.SecretKey = utils.EncryptToString(c.VolumesConfig.SecretKey)
	}

	if c.LicenseConfig != nil && c.LicenseConfig.Key != "" {
		c.LicenseConfig.Key = utils.EncryptToString(c.LicenseConfig.Key)
	}

	if c.AuthConfig != nil {
		if c.AuthConfig.Github.ClientSecret != "" {
			c.AuthConfig.Github.ClientSecret = utils.EncryptToString(c.AuthConfig.Github.ClientSecret)
		}

		if c.AuthConfig.Github.PrivateKey != "" {
			c.AuthConfig.Github.PrivateKey = utils.EncryptToString(c.AuthConfig.Github.PrivateKey)
		}

		if c.AuthConfig.Github.RunnerToken != "" {
			c.AuthConfig.Github.RunnerToken = utils.EncryptToString(c.AuthConfig.Github.RunnerToken)
		}

		if c.AuthConfig.Gitlab.ClientSecret != "" {
			c.AuthConfig.Gitlab.ClientSecret = utils.EncryptToString(c.AuthConfig.Gitlab.ClientSecret)
		}

		if c.AuthConfig.Bitbucket.ClientSecret != "" {
			c.AuthConfig.Bitbucket.ClientSecret = utils.EncryptToString(c.AuthConfig.Bitbucket.ClientSecret)
		}

		if c.AuthConfig.Bitbucket.DeployKey != "" {
			c.AuthConfig.Bitbucket.DeployKey = utils.EncryptToString(c.AuthConfig.Bitbucket.DeployKey)
		}
	}

	return json.Marshal(c)
}

func (vc InstanceConfig) IsEmpty() bool {
	return vc.VolumesConfig == nil
}

// IsAuthEnabled returns whether the auth config is enabled or not.
func (vc InstanceConfig) IsAuthEnabled() bool {
	return vc.IsGithubEnabled() || vc.IsGitlabEnabled() || vc.IsBitbucketEnabled()
}

// IsBitbucketEnabled returns whether the Bitbucket auth config is enabled or not.
func (vc InstanceConfig) IsBitbucketEnabled() bool {
	if vc.AuthConfig == nil {
		return false
	}

	return vc.AuthConfig.Bitbucket.ClientID != "" &&
		vc.AuthConfig.Bitbucket.ClientSecret != ""
}

// IsGitlabEnabled returns whether the GitLab auth config is enabled or not.
func (vc InstanceConfig) IsGitlabEnabled() bool {
	if vc.AuthConfig == nil {
		return false
	}

	return vc.AuthConfig.Gitlab.ClientID != "" &&
		vc.AuthConfig.Gitlab.ClientSecret != ""
}

// IsGithubEnabled returns whether the GitHub auth config is enabled or not.
func (vc InstanceConfig) IsGithubEnabled() bool {
	if vc.AuthConfig == nil {
		return false
	}

	return vc.AuthConfig.Github.ClientID != "" &&
		vc.AuthConfig.Github.ClientSecret != "" &&
		vc.AuthConfig.Github.PrivateKey != "" &&
		vc.AuthConfig.Github.Account != "" &&
		vc.AuthConfig.Github.AppID > 0
}

// SignUpMode returns the configured sign up mode.
// If the AuthConfig or UserManagement configurations are not defined,
// the default is `on`
func (vc InstanceConfig) SignUpMode() string {
	if vc.AuthConfig == nil || vc.AuthConfig.UserManagement.SignUpMode == "" {
		return SIGNUP_MODE_ON
	}

	// Just to make sure we don't return something random
	switch vc.AuthConfig.UserManagement.SignUpMode {
	case SIGNUP_MODE_OFF:
		return SIGNUP_MODE_OFF
	case SIGNUP_MODE_WAITLIST:
		return SIGNUP_MODE_WAITLIST
	default:
		return SIGNUP_MODE_ON
	}
}

// IsUserWhitelisted returns whether the given email is whitelisted.
// If the sign up mode is `on`, all users are whitelisted. If the sign up mode
// is `off`, no users are whitelisted. If the sign up mode is `waitlist`, only
// users in the whitelist are allowed.
//
// The whitelist supports two modes:
// 1. Allow mode: ["example.com", "test.com"] - only these domains are allowed
// 2. Deny mode: ["!spam.com", "!blocked.com"] - all domains except these are allowed
func (vc InstanceConfig) IsUserWhitelisted(email string) bool {
	signUpMode := vc.SignUpMode()

	if signUpMode == SIGNUP_MODE_ON {
		return true
	}

	if signUpMode == SIGNUP_MODE_OFF {
		return false
	}

	whitelist := vc.AuthConfig.UserManagement.Whitelist

	if len(whitelist) == 0 {
		return false
	}

	pieces := strings.SplitN(email, "@", 2)

	if len(pieces) != 2 {
		return false
	}

	domain := pieces[1]

	// Check if we're in deny mode (first entry starts with !)
	if strings.HasPrefix(whitelist[0], "!") {
		// Deny mode: allow all domains except those prefixed with !
		for _, entry := range whitelist {
			if len(entry) > 1 && entry[0] == '!' {
				deniedDomain := entry[1:] // Remove the ! prefix

				if strings.EqualFold(domain, deniedDomain) {
					return false // Domain is explicitly denied
				}
			}
		}
		return true // Domain is not in the deny list, so allow it
	}

	// Allow mode: only allow domains in the whitelist
	if utils.InSliceString(vc.AuthConfig.UserManagement.Whitelist, domain) {
		return true
	}

	return false
}

// SetURL sets the domain config based on the given domain.
func (vc InstanceConfig) SetURL(domain string) {
	// Only set the URL in test mode.
	if !config.IsTest() {
		return
	}

	parsed := GetParsedURL(domain)

	if parsed == nil {
		return
	}

	if vc.DomainConfig == nil {
		vc.DomainConfig = &DomainConfig{}
	}

	vc.DomainConfig.App = fmt.Sprintf("%s://stormkit.%s", parsed.Scheme, parsed.Host)
	vc.DomainConfig.API = fmt.Sprintf("%s://api.%s", parsed.Scheme, parsed.Host)
	vc.DomainConfig.Dev = fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
	vc.DomainConfig.Health = fmt.Sprintf("%s://health.%s", parsed.Scheme, parsed.Host)

	cachedConfigMux.Lock()

	if cachedConfig == nil {
		cachedConfig = &vc
	}

	cachedConfig.DomainConfig = vc.DomainConfig
	cachedConfigMux.Unlock()
}

// PreviewURL returns the fully qualified api for the dev endpoint.
func (vc InstanceConfig) PreviewURL(displayName string, deploymentIDOrEnv ...string) string {
	if vc.DomainConfig == nil {
		return ""
	}

	if len(deploymentIDOrEnv) > 0 && deploymentIDOrEnv[0] != config.AppDefaultEnvironmentName {
		return strings.Replace(vc.DomainConfig.Dev, "//", fmt.Sprintf("//%s--%s.", displayName, deploymentIDOrEnv[0]), 1)
	}

	return strings.Replace(vc.DomainConfig.Dev, "//", fmt.Sprintf("//%s.", displayName), 1)
}

// ApiURL returns the fully qualified api url for the api.
func (vc InstanceConfig) ApiURL(path string) string {
	if vc.DomainConfig == nil {
		return ""
	}

	if path == "" {
		return vc.DomainConfig.API
	}

	return vc.DomainConfig.API + "/" + strings.TrimLeft(path, "/")
}

// WebhooksURL returns the URL for webhooks from GitLab, GitHub and Bitbucket.
func (vc InstanceConfig) WebhooksURL(path string) string {
	if vc.DomainConfig == nil {
		return ""
	}

	if vc.DomainConfig.Webhooks != "" {
		if path == "" {
			return vc.DomainConfig.Webhooks
		}

		return vc.DomainConfig.Webhooks + "/" + strings.TrimLeft(path, "/")
	}

	return vc.ApiURL(path)
}

// AppURL returns the fully qualified app url for the app (https://app.stormkit.io/path).
func (vc InstanceConfig) AppURL(path string) string {
	if vc.DomainConfig == nil {
		return ""
	}

	if path == "" {
		return vc.DomainConfig.App
	}

	return vc.DomainConfig.App + "/" + strings.TrimLeft(path, "/")
}

// DeploymentLogsURL returns the URL to preview deployment logs on the App.
func (vc InstanceConfig) DeploymentLogsURL(appID, deploymentID types.ID) string {
	return vc.AppURL(path.Join("app", appID.String(), "deployments", deploymentID.String()))
}

// RuntimeLogsURL returns the URL to preview runtime logs on the App.
func (vc InstanceConfig) RuntimeLogsURL(appID, envID, deploymentID types.ID) string {
	return vc.AppURL(path.Join("apps", appID.String(), "environments", envID.String(), "deployments", deploymentID.String(), "runtime-logs"))
}

// AddRuntimes adds the given runtimes to the list of dependencies.
func AddRuntimes(ctx context.Context, runtimes []string) error {
	vc, err := Store().Config(ctx)

	if err != nil {
		return err
	}

	if vc.SystemConfig == nil {
		vc.SystemConfig = defaultSystemConfig()
	}

	existingRuntimes := map[string]bool{}

	for _, runtime := range vc.SystemConfig.Runtimes {
		existingRuntimes[runtime] = true
	}

	addedNew := false

	for _, runtime := range runtimes {
		if existingRuntimes[runtime] {
			continue
		}

		vc.SystemConfig.Runtimes = append(vc.SystemConfig.Runtimes, runtime)
		addedNew = true
	}

	if !addedNew {
		return nil
	}

	if err := Store().UpsertConfig(ctx, vc); err != nil {
		return err
	}

	return rediscache.Broadcast(rediscache.EventRuntimesInstall)
}

func defaultSystemConfig() *SystemConfig {
	cfg := &SystemConfig{
		AutoInstall: true,
		Runtimes:    []string{},
	}

	// Backwards compatibility: if the NODE_VERSION environment variable is set, we assume
	// that the user wants to use the old method of installing runtimes.
	if nodeVersion := os.Getenv("NODE_VERSION"); nodeVersion != "" {
		cfg.Runtimes = append(cfg.Runtimes, fmt.Sprintf("node@%s", nodeVersion))
		cfg.Runtimes = append(cfg.Runtimes, "yarn@1.22")
		cfg.Runtimes = append(cfg.Runtimes, "pnpm@latest")
		cfg.Runtimes = append(cfg.Runtimes, "npm@latest")
	}

	return cfg
}

func installDependencies(ctx context.Context) error {
	var err error
	var vc InstanceConfig
	vc, err = Store().Config(ctx)

	if err != nil {
		slog.Errorf("error getting admin config: %v", err)
		return err
	}

	if vc.SystemConfig == nil {
		vc.SystemConfig = defaultSystemConfig()

		// Save the default config so we can see installed packages
		if err := Store().UpsertConfig(ctx, vc); err != nil {
			slog.Errorf("error saving default system config: %v", err)
		}
	}

	m := mise.Client()

	if err = m.InstallMise(ctx); err != nil {
		slog.Errorf("error installing mise: %v", err)
		return err
	}

	if err := m.Prune(ctx); err != nil {
		slog.Errorf("error pruning mise: %v", err)
	}

	var output string

	for _, runtime := range vc.SystemConfig.Runtimes {
		output, err = m.InstallGlobal(ctx, runtime)

		if err != nil {
			slog.Errorf("error installing runtime %s: err=%v, output=%s", runtime, err, output)
			return err
		}

		slog.Debug(slog.LogOpts{
			Msg:   "runtime installed",
			Level: slog.DL1,
			Payload: []zap.Field{
				zap.String("runtime", runtime),
			},
		})
	}

	slog.Debug(slog.LogOpts{
		Msg:   "finished installing runtimes",
		Level: slog.DL1,
		Payload: []zap.Field{
			zap.String("runtimes", strings.Join(vc.SystemConfig.Runtimes, ", ")),
		},
	})

	return nil
}

// InstallDependencies is a job that installs the required runtimes for the instance.
func InstallDependencies(ctx context.Context, payload ...string) {
	client := rediscache.Client()
	keyName := rediscache.Service().Key(rediscache.KEY_RUNTIMES_STATUS)
	client.Set(context.Background(), keyName, rediscache.StatusProcessing, time.Hour)

	retryCount := 0
	maxRetries := 5
	maxDuration := 2 * time.Minute
	startTime := time.Now()

	for retryCount < maxRetries && time.Since(startTime) < maxDuration {
		slog.Debug(slog.LogOpts{
			Msg:   "installing runtime dependencies",
			Level: slog.DL1,
			Payload: []zap.Field{
				zap.Int("attempt", retryCount+1),
			},
		})

		if retryCount > 0 {
			backoff := time.Duration(1<<uint(retryCount-1)) * time.Second

			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}

			time.Sleep(backoff)
		}

		retryCount++

		if err := installDependencies(ctx); err == nil {
			client.Set(ctx, keyName, rediscache.StatusOK, time.Minute)
			return
		}
	}

	client.Set(ctx, keyName, rediscache.StatusErr, time.Minute)
}
