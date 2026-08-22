package buildconf

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/mise"
	null "gopkg.in/guregu/null.v3"
)

type PublishedInfo struct {
	DeploymentID  types.ID    `json:"deploymentId,string"`
	Percentage    float64     `json:"percentage"`
	Branch        string      `json:"branch"`
	CommitAuthor  null.String `json:"commitAuthor"`
	CommitSha     null.String `json:"commitSha"`
	CommitMessage null.String `json:"commitMessage"`
}

type SKAuthConf struct {
	Secret     string
	SuccessURL string
	Status     bool // Whether the authentication is enabled or not.
	TTL        int  // in minutes
	// AllowedOrigins is an opt-in allow-list of origins (scheme + host, e.g.
	// "https://app.example.com") permitted to initiate the magic-link flow
	// cross-origin. When non-empty, POSTs to /_stormkit/auth/magic must
	// carry a matching Origin header and the user is redirected back to
	// that origin after clicking the email link.
	AllowedOrigins []string

	// CookieDomain optionally scopes the session cookie to a parent domain
	// (e.g. ".example.com") so it is shared across subdomains — needed when the
	// login origin and the OAuth authorization-server origin are different
	// subdomains. Empty means a host-only cookie.
	CookieDomain string

	// LoginURL is the app-owned login page the OAuth /authorize endpoint
	// redirects an unauthenticated user to. The app authenticates with its own
	// UI, establishes the shared session cookie, and bounces the user back to
	// the return_to URL /authorize appends. A relative path (leading "/") on
	// the authorization-server origin, or an absolute URL on the login origin.
	LoginURL string

	// OAuthServer, when enabled, turns this environment into an OAuth 2.1
	// authorization server for MCP connectors (see /_stormkit/oauth and the
	// .well-known discovery documents). Note this is the opposite role from the
	// Google/X provider logins above: here the app IS the OAuth server that
	// external clients connect to. It builds on the SkAuth identity;
	// redirect_uri targets are validated against AllowedOrigins.
	OAuthServer *OAuthServerConf
}

// SessionCookieName is the name of the SkAuth session cookie.
const SessionCookieName = "skauth_session"

// OAuthServerConf configures the OAuth 2.1 authorization server layered on
// SkAuth. Distinct from the OAuth *provider* logins (Google, X), where the app
// is instead an OAuth client of an external identity provider.
type OAuthServerConf struct {
	// Enabled turns the /_stormkit/oauth/* endpoints and the .well-known
	// discovery documents on for this environment.
	Enabled bool

	// ResourcePath is the path this environment serves its MCP server on (e.g.
	// "/mcp"). When set it is appended to the resource identifier in the
	// protected-resource metadata and the access-token audience, and it selects
	// which /.well-known/oauth-protected-resource/<path> probe is answered. MCP
	// clients (Claude, ChatGPT) require the resource to match the URL the user
	// entered, path included, so this must equal the connector URL's path.
	ResourcePath string

	// AllowLoopback opts in to RFC 8252 loopback redirects for native/CLI
	// clients (e.g. Claude Code), which listen on an ephemeral localhost port.
	// When on, a redirect_uri whose host is a loopback literal is matched on
	// scheme+path only, ignoring the port.
	AllowLoopback bool
}

// OAuthServerEnabled reports whether the OAuth authorization server is active.
// It requires SkAuth itself to be enabled, since the server reuses its signing
// secret and app-user identities.
func (ac *SKAuthConf) OAuthServerEnabled() bool {
	return ac != nil && ac.Status && ac.OAuthServer != nil && ac.OAuthServer.Enabled
}

// ResourcePath returns the configured MCP resource path normalized to a leading
// slash with no trailing slash (e.g. "mcp/" -> "/mcp"). An unset path returns
// "", meaning the resource identifier is the bare issuer.
func (ac *SKAuthConf) ResourcePath() string {
	if !ac.OAuthServerEnabled() || ac.OAuthServer.ResourcePath == "" {
		return ""
	}

	p := utils.TrimPath(ac.OAuthServer.ResourcePath)

	if p == "/" {
		return ""
	}

	return p
}

// AllowLoopbackRedirects reports whether native/CLI clients using RFC 8252
// loopback redirects are permitted for this environment.
func (ac *SKAuthConf) AllowLoopbackRedirects() bool {
	return ac.OAuthServerEnabled() && ac.OAuthServer.AllowLoopback
}

// Value implements the Sql Driver interface.
func (ac *SKAuthConf) Value() (driver.Value, error) {
	return utils.ByteaValue(ac)
}

// IsAllowedOrigin reports whether origin (scheme + host, no trailing slash,
// e.g. "https://app.example.com") is present in the configured AllowedOrigins
// allow-list. An empty origin or an empty list returns false.
func (ac *SKAuthConf) IsAllowedOrigin(origin string) bool {
	if ac == nil || origin == "" || len(ac.AllowedOrigins) == 0 {
		return false
	}

	for _, allowed := range ac.AllowedOrigins {
		if strings.TrimRight(allowed, "/") == origin {
			return true
		}
	}

	return false
}

// Env represents an application's environment.
type Env struct {
	// ID represents the environment id.
	ID types.ID `json:"id,omitempty,string" db:"env_id"`

	// AppID is the application id.
	AppID types.ID `json:"appId,omitempty,string" db:"app_id"`

	// Name is the name of the environment.
	// TODO: Return this one instead of Env field above.
	Name string `json:"name" db:"env_name"`

	// Data is the build configuration data.
	Data *BuildConf `json:"build" db:"build_conf"`

	// SchemaConf holds the database schema configuration.
	SchemaConf *SchemaConf `json:"-"`

	// AuthConf holds the configuration for authentication.
	AuthConf *SKAuthConf `json:"authConf"`

	// MailerConf holds the configuration for sending transactional emails.
	MailerConf *MailerConf `json:"mailer,omitempty"`

	// AutoPublish specifies whether successful deployment should be
	// publish to 100% immediately.
	AutoPublish bool `json:"autoPublish" db:"auto_publish"`

	// Branch is the associated branch name with this environment.
	// If this is specified, pushes/merges to these branches will trigger a deploy.
	Branch string `json:"branch" db:"branch"`

	// AutoDeploy specifies whether the automatic deployments for this environment
	// are turned or not.
	AutoDeploy bool `json:"autoDeploy" db:"auto_deploy"`

	// AutoDeployBranches is a regexp config that specifies which
	// branches to deploy automatically.
	AutoDeployBranches null.String `json:"autoDeployBranches,omitempty" db:"auto_deploy_branches"`

	// AutoDeployCommits is a regexp config that specifies which
	// commits to deploy automatically.
	AutoDeployCommits null.String `json:"autoDeployCommits,omitempty"`

	UpdatedAt utils.Unix `json:"-"`

	DeletedAt utils.Unix `json:"-" db:"deleted_at"`

	// DeployedAt specifies the last deployment time for the environment.
	DeployedAt utils.Unix `json:"-"`

	// LastDeployID is the id of the last deployment, if any.
	LastDeployID null.Int `json:"-"`

	// LastDeployExitCode holds the last exit code.
	LastDeployExitCode null.Int `json:"-"`

	// Env is the name of the environment.
	// @deprecated Use 'Name' instead.
	Env string `json:"env"`

	// if env has published deployment.
	Published []*PublishedInfo `json:"published,omitempty"`

	// Preview is the preview URL for the environment.
	Preview string `json:"preview,omitempty"`
}

func (Env) TableName() string {
	return "apps_build_conf"
}

// AuthReady reports whether authentication is fully provisioned for this
// environment: enabled auth config and a schema to back it. Auth handlers guard
// on this before touching the auth store, so the check lives here once instead
// of being spelled out (in varying orders) at every call site.
func (e *Env) AuthReady() bool {
	return e != nil && e.AuthConf != nil && e.AuthConf.Status && e.SchemaConf != nil
}

// DomainInfo represents a domainInfo struct returned by the database.
// It is used by the store's DomainInfo method.
type DomainInfo struct {
	// DomainName represents the domain name.
	DomainName null.String

	// Token represents the domain token for verification.
	Token null.String

	// Verifies specifies whether the domain is verified or not.
	Verified null.Bool

	// AppDisplayName represents the display name of the application.
	// In case a domain name is missing, then this can be used to construct urls.
	AppDisplayName string

	// EnvName represents the environment name. This is used in conjuction with
	// AppDisplayName to generate a dev domain when the DomainName is missing.
	EnvName string
}

// MaskVars returns a copy of vars with every VALUE blanked (keys kept). It is
// the single masking rule for env-var values — used when serializing an Env and
// a deployment's config snapshot — so values are never exposed in plaintext
// outside the dedicated, access-controlled pull endpoint.
func MaskVars(vars map[string]string) map[string]string {
	masked := make(map[string]string, len(vars))

	for key := range vars {
		masked[key] = ""
	}

	return masked
}

// JSON returns the env as a client-ready map (App.JSON() style) with secret
// env-var values masked. The map is built explicitly — every exposed field is
// listed here rather than derived from struct tags via a marshal/unmarshal
// round-trip. MarshalJSON delegates to this, so an Env can never be serialized
// to a client with plaintext values by accident. Internal consumers that need
// real values read Env.Data.Vars directly and never go through here.
func (env Env) JSON() map[string]any {
	type domain struct {
		Name     string `json:"name,omitempty"`
		Verified bool   `json:"verified"`
		CName    string `json:"cname,omitempty"`
	}

	type lastDeploy struct {
		DeploymentID types.ID   `json:"id"`
		CreatedAt    utils.Unix `json:"createdAt"`
		ExitCode     *int64     `json:"exit,omitempty"`
	}

	// Mask on a copy so the original Env keeps the real values for internal use.
	build := env.Data

	if env.Data != nil && len(env.Data.Vars) > 0 {
		conf := *env.Data
		conf.Vars = MaskVars(env.Data.Vars)
		build = &conf
	}

	// env is deprecated; mirror Name when it is empty.
	envName := env.Env

	if envName == "" {
		envName = env.Name
	}

	m := map[string]any{
		"name":        env.Name,
		"env":         envName,
		"branch":      env.Branch,
		"build":       build,
		"authConf":    env.AuthConf,
		"autoPublish": env.AutoPublish,
		"autoDeploy":  env.AutoDeploy,
		"domain":      domain{},
	}

	if env.AutoDeployBranches.ValueOrZero() != "" {
		m["autoDeployBranches"] = env.AutoDeployBranches
	}

	if env.AutoDeployCommits.ValueOrZero() != "" {
		m["autoDeployCommits"] = env.AutoDeployCommits
	}

	if env.ID != 0 {
		m["id"] = env.ID.String()
	}

	if env.AppID != 0 {
		m["appId"] = env.AppID.String()
	}

	if env.MailerConf != nil {
		m["mailer"] = env.MailerConf.JSON()
	}

	if len(env.Published) > 0 {
		m["published"] = env.Published
	}

	if env.Preview != "" {
		m["preview"] = env.Preview
	}

	if env.LastDeployID.ValueOrZero() != 0 {
		m["lastDeploy"] = lastDeploy{
			DeploymentID: types.ID(env.LastDeployID.ValueOrZero()),
			CreatedAt:    env.DeployedAt,
			ExitCode:     env.LastDeployExitCode.Ptr(),
		}
	}

	return m
}

// MarshalJSON implements json.Marshaler by delegating to JSON, so masking is
// the fail-safe default everywhere an Env is serialized.
func (env Env) MarshalJSON() ([]byte, error) {
	return json.Marshal(env.JSON())
}

type StatusCheck struct {
	Name        string `json:"name"`
	Cmd         string `json:"cmd"`
	Description string `json:"description"`
}

// BuildConf is the struct that represents the JSON data
type BuildConf struct {
	PreviewLinks    null.Bool            `json:"previewLinks,omitempty"`    // Whether preview links are enabled or not.
	Redirects       []redirects.Redirect `json:"redirects,omitempty"`       // The redirects defined from UI. When defined, this one will take precedence over redirects file.
	APIFolder       string               `json:"apiFolder,omitempty"`       // Path to api folder (from repository root).
	APIPathPrefix   string               `json:"apiPathPrefix,omitempty"`   // Path prefix in the URL that will be used to call api functions, default: /api
	RedirectsFile   string               `json:"redirectsFile,omitempty"`   // Path to the redirects file.
	ErrorFile       string               `json:"errorFile,omitempty"`       // When specified, we'll load this file instead of the default 404.html or error.html
	Markdown        null.Bool            `json:"markdown,omitempty"`        // Serve the .md twin of a page to clients that send Accept: text/markdown. Off unless enabled.
	Headers         string               `json:"headers,omitempty"`         // Custom headers set from the UI.
	HeadersFile     string               `json:"headersFile,omitempty"`     // Path to the headers file. The path is relative to working dir.
	DistFolder      string               `json:"distFolder,omitempty"`      // DistFolder is the client dist folder.
	WorkDir         string               `json:"workDir,omitempty"`         // Working directory relative to the repository root where install/build commands run. Falls back to the SK_CWD env var when unset.
	Cmd             string               `json:"cmd,omitempty"`             // Deprecated. Declared only for retro compability.
	InstallCmd      string               `json:"installCmd,omitempty"`      // The install command to install the dependencies.
	BuildCmd        string               `json:"buildCmd,omitempty"`        // The build command to build the application.
	ServerCmd       string               `json:"serverCmd,omitempty"`       // The command to spawn the server. This is a self-hosted only feature.
	Vars            map[string]string    `json:"vars,omitempty"`            // The environment variables that will be injected to the application.
	StatusChecks    []StatusCheck        `json:"statusChecks,omitempty"`    // StatusChecks is an array of commands that will be executed after the deployment is complete.
	PriorityPattern string               `json:"priorityPattern,omitempty"` // PriorityPattern is a regex matched against the commit message to auto-prioritize deployments.
	CacheDirs       []string             `json:"cacheDirs,omitempty"`       // CacheDirs is a list of directories (relative to the working directory) restored before install and snapshotted after a successful build.
}

type InterpolatedVarsOpts struct {
	AppID        string
	DisplayName  string
	DeploymentID string
	Env          string
	EnvID        string
	BinPaths     bool
}

func systmVars() map[string]string {
	if config.IsStormkitCloud() {
		return map[string]string{}
	}

	return config.Get().Secrets
}

func (bc *BuildConf) InterpolatedVars(opts InterpolatedVarsOpts) map[string]string {
	conf := admin.MustConfig()
	vars := bc.Vars

	if vars == nil {
		vars = map[string]string{}
	}

	if opts.BinPaths {
		paths, err := mise.Client().BinPaths(context.Background())

		if err != nil {
			slog.Errorf("Error fetching mise paths: %v\n", err)
		}

		for key, value := range paths {
			vars[key] = value
		}
	}

	vars["SK_APP_ID"] = opts.AppID
	vars["SK_ENV"] = opts.Env
	vars["SK_ENV_ID"] = opts.EnvID
	vars["SK_ENV_URL"] = conf.PreviewURL(opts.DisplayName, opts.Env)
	vars["SK_DEPLOYMENT_ID"] = opts.DeploymentID
	vars["SK_DEPLOYMENT_URL"] = conf.PreviewURL(opts.DisplayName, opts.DeploymentID)
	vars["STORMKIT"] = "true"

	svars := systmVars()

	// Interpolate variables like:
	// NEXT_PUBLIC_SITE_URL = $SK_ENV_URL
	for k, v := range vars {
		// If the variable starts with $,
		// we assume it's a reference to another variable
		// and we replace it with the value of that variable.
		// For example, if v is "$SK_ENV_URL", we replace it with vars["SK_ENV_URL"].
		if strings.HasPrefix(v, "$") && len(v) > 1 {
			// Remove the leading $ sign and make sure we have a valid reference
			if ref := v[1:]; ref != "" {
				// If the reference is the same as the key,
				// then check system variables.
				if ref != k && vars[ref] != "" {
					vars[k] = vars[ref]
				} else if svars[ref] != "" {
					vars[k] = svars[ref]
				}
			}
		}
	}

	return vars
}

// DefaultConfig returns the default configuration.
func DefaultConfig(appID types.ID) *Env {
	env := &Env{
		AppID:       appID,
		Name:        "production",
		Branch:      "main",
		AutoDeploy:  true,
		AutoPublish: true,
		Data: &BuildConf{
			Vars: map[string]string{
				"NODE_ENV": "production",
			},
		},
	}

	return env
}

func Validate(env *Env) []string {
	errors := []string{}

	if env.Branch == "" {
		errors = append(errors, "Branch is a required field")
	} else if match, _ := regexp.MatchString(`^[a-zA-Z0-9-/+=\.]+$`, env.Branch); !match {
		// See https://wincent.com/wiki/Legal_Git_branch_names for more details.
		errors = append(errors, "Branch name can only contain following characters: alphanumeric, -, +, /, ., and =")
	}

	env.Name = utils.GetString(env.Name, env.Env) // Env is deprecated, but we still want to support it for a while

	if env.Name == "" {
		errors = append(errors, "Name is a required field")
	} else if match, _ := regexp.MatchString("^[a-zA-Z-0-9]+$", env.Name); !match {
		errors = append(errors, "Environment name can only contain alphanumeric characters and hyphens")
	}

	if match, _ := regexp.MatchString("--", env.Name); match {
		errors = append(errors, "Double hyphens (--) are not allowed as they are reserved for Stormkit")
	}

	if env.AutoDeployBranches.ValueOrZero() != "" {
		if _, rerr := regexp2.Compile(env.AutoDeployBranches.ValueOrZero(), regexp2.IgnoreCase); rerr != nil {
			errors = append(errors, fmt.Sprintf("Auto deploy branches regex is invalid: %s", rerr.Error()))
		}
	}

	if env.Data != nil {
		errors = append(errors, ValidateCacheDirs(env.Data.CacheDirs)...)
	}

	if len(errors) == 0 {
		return nil
	}

	return errors
}

// NormalizeCacheDirs trims each entry and drops empty ones, so callers can
// accept a raw list (e.g. a textarea split by newlines) before validation.
func NormalizeCacheDirs(dirs []string) []string {
	normalized := make([]string, 0, len(dirs))

	for _, dir := range dirs {
		if dir = strings.TrimSpace(dir); dir != "" {
			normalized = append(normalized, dir)
		}
	}

	return normalized
}

// ValidateCacheDirs ensures every cache directory is a relative path that
// stays inside the build working directory. The runner extracts cache
// archives with these paths, so anything escaping the workdir is rejected.
func ValidateCacheDirs(dirs []string) []string {
	errors := []string{}

	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)

		if dir == "" {
			errors = append(errors, "Cache directories cannot be empty")
			continue
		}

		if strings.HasPrefix(dir, "/") || strings.HasPrefix(dir, "~") {
			errors = append(errors, fmt.Sprintf("Cache directory %q must be relative to the working directory", dir))
			continue
		}

		// A leading dash would be interpreted as a flag when the directory is
		// passed to tar during snapshot/restore.
		if strings.HasPrefix(dir, "-") {
			errors = append(errors, fmt.Sprintf("Cache directory %q cannot start with a dash", dir))
			continue
		}

		cleaned := path.Clean(dir)

		if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
			errors = append(errors, fmt.Sprintf("Cache directory %q must point inside the working directory", dir))
		}
	}

	return errors
}
