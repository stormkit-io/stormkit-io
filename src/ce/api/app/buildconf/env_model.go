package buildconf

import (
	"database/sql/driver"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/mailer"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/model"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
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

type AuthConf struct {
	Secret string
	TTL    int // in minutes
}

// Value implements the Sql Driver interface.
func (ac *AuthConf) Value() (driver.Value, error) {
	return utils.ByteaValue(ac)
}

// Env represents an application's environment.
type Env struct {
	model.Model `json:"-"`

	// ID represents the environment id.
	ID types.ID `json:"id,omitempty,string" db:"env_id"`

	// AppID is the application id.
	AppID types.ID `json:"appId,omitempty,string" db:"app_id"`

	// Name is the name of the environment.
	// TODO: Return this one instead of Env field above.
	Name string `json:"-" db:"env_name"`

	// Data is the build configuration data.
	Data *BuildConf `json:"build" db:"build_conf"`

	// SchemaConf holds the database schema configuration.
	SchemaConf *SchemaConf `json:"-"`

	// AuthConf holds the configuration for authentication.
	AuthConf *AuthConf `json:"authConf"`

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

	// Mailer configuration for sending transactional emails.
	Mailer *mailer.Config `json:"mailer,omitempty"`

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

// EnvJSON is a wrapper around myapp to avoid recursion.
type EnvJSON Env

// MarshalJSON implements the marshaler interface.
func (env Env) MarshalJSON() ([]byte, error) {
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

	var ld *lastDeploy

	if env.LastDeployID.ValueOrZero() != 0 {
		ld = &lastDeploy{
			DeploymentID: types.ID(env.LastDeployID.ValueOrZero()),
			CreatedAt:    env.DeployedAt,
			ExitCode:     env.LastDeployExitCode.Ptr(),
		}
	}

	// TODO: Remove this snippet when we remove the Env field.
	if env.Env == "" && env.Name != "" {
		env.Env = env.Name
	}

	ret := struct {
		EnvJSON
		LastDeploy *lastDeploy `json:"lastDeploy,omitempty"`
		Domain     domain      `json:"domain"`
	}{
		EnvJSON:    EnvJSON(env),
		LastDeploy: ld,
	}

	return json.Marshal(ret)
}

// Validate implements the model.Validate interface.
func (env *Env) Validate() *shttperr.ValidationError {
	err := &shttperr.ValidationError{}
	env.Env = strings.ToLower(strings.TrimSpace(env.Env))

	// TODO: Backward compability. env.Env will be deprecated.
	if env.Env == "" {
		env.Env = strings.ToLower(strings.TrimSpace(env.Name))
	}

	if match, _ := regexp.MatchString("^[a-zA-Z-0-9]+$", env.Env); !match {
		err.SetError("env", ErrInvalidEnv.Error())
	}

	if match, _ := regexp.MatchString("--", env.Env); match {
		err.SetError("env", ErrInvalidEnvDoubleHypens.Error())
	}

	if env.Env == "" {
		err.SetError("env", ErrMissingEnv.Error())
	}

	if match, _ := regexp.MatchString(`^[a-zA-Z0-9-/+=\.]+$`, env.Branch); !match {
		err.SetError("branch", ErrInvalidBranch.Error())
	}

	if env.Env == "" {
		err.SetError("env", ErrMissingEnv.Error())
	}

	if env.AutoDeployBranches.ValueOrZero() != "" {
		if _, rerr := regexp2.Compile(env.AutoDeployBranches.ValueOrZero(), regexp2.IgnoreCase); rerr != nil {
			err.SetError("autoDeployBranches", rerr.Error())
		}
	}

	return err.ToError()
}

type StatusCheck struct {
	Name        string `json:"name"`
	Cmd         string `json:"cmd"`
	Description string `json:"description"`
}

// BuildConf is the struct that represents the JSON data
type BuildConf struct {
	PreviewLinks  null.Bool            `json:"previewLinks,omitempty"`  // Whether preview links are enabled or not.
	Redirects     []redirects.Redirect `json:"redirects,omitempty"`     // The redirects defined from UI. When defined, this one will take precedence over redirects file.
	APIFolder     string               `json:"apiFolder,omitempty"`     // Path to api folder (from repository root).
	APIPathPrefix string               `json:"apiPathPrefix,omitempty"` // Path prefix in the URL that will be used to call api functions, default: /api
	RedirectsFile string               `json:"redirectsFile,omitempty"` // Path to the redirects file.
	ErrorFile     string               `json:"errorFile,omitempty"`     // When specified, we'll load this file instead of the default 404.html or error.html
	Headers       string               `json:"headers,omitempty"`       // Custom headers set from the UI.
	HeadersFile   string               `json:"headersFile,omitempty"`   // Path to the headers file. The path is relative to working dir.
	DistFolder    string               `json:"distFolder,omitempty"`    // DistFolder is the client dist folder.
	ServerFolder  string               `json:"serverFolder,omitempty"`  // The server folder to upload to the server side
	Cmd           string               `json:"cmd,omitempty"`           // Deprecated. Declared only for retro compability.
	InstallCmd    string               `json:"installCmd,omitempty"`    // The install command to install the dependencies.
	BuildCmd      string               `json:"buildCmd,omitempty"`      // The build command to build the application.
	ServerCmd     string               `json:"serverCmd,omitempty"`     // The command to spawn the server. This is a self-hosted only feature.
	Vars          map[string]string    `json:"vars,omitempty"`          // The environment variables that will be injected to the application.
	StatusChecks  []StatusCheck        `json:"statusChecks,omitempty"`  // StatusChecks is an array of commands that will be executed after the deployment is complete.
}

type InterpolatedVarsOpts struct {
	AppID        string
	DisplayName  string
	DeploymentID string
	Env          string
	EnvID        string
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
