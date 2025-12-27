package app

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	randomdata "github.com/Pallinder/go-randomdata"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/bitbucket"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/github"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/gitlab"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/model"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	null "gopkg.in/guregu/null.v3"
)

// SampleProjectRepo represents the git repo name of the sample
// project, set foreach application as onboarding.
var SampleProjectRepo = "github/stormkit-io/sample-project"

// App is the app which the user is going to work with.
type App struct {
	model.Model `json:"-"`

	// ID is the application id.
	ID types.ID `json:"id,omitempty,string"`

	// UserID is the ID of the user who created the application.
	UserID types.ID `json:"userId,omitempty,string"`

	// TeamID is the ID of the team that the app belongs to.
	TeamID types.ID `json:"teamId,omitempty,string"`

	// DefaultEnvID is the ID of the default environment.
	DefaultEnvID types.ID `json:"defaultEnvId,omitempty,string"`

	// Repo represents the application repository. It is in the following format:
	// :provider/:owner/:slug where provider can be either github, bitbucket or gitlab.
	Repo string `json:"repo"`

	// DisplayName is the app's nickname. It will be used as a subdomain stormkit.dev
	// domains. It needs to be unique across the whole application.
	DisplayName string `json:"displayName,omitempty"`

	// ClientID is generated at application create time. It will be used for authentication
	// thru an API. It can be regenerated.
	ClientID string `json:"clientId,omitempty"`

	// ClientSecret is the secret of the application, that is created upon create time.
	// It is used for authentication thru an API.
	ClientSecret []byte `json:"-"`

	// AutoDeploy is the setting for auto deployments. It can be either commit
	// or pull_request.
	AutoDeploy null.String `json:"autoDeploy,omitempty"`

	// ArtifactsDeleted is a boolean value that indicates whether the artifacts
	// of the application are deleted or not. When an app is deleted, it takes a
	// bit time to remove its artifacts from the AWS systems. Once the operation
	// is done, this field will be set to true.
	ArtifactsDeleted null.Bool `json:"-"`

	// DeletedAt represents the timestamp that the application was deleted.
	DeletedAt null.Time `json:"-"`

	// CreatedAt represents the timestamp that the application was created.
	CreatedAt utils.Unix `json:"createdAt,omitempty"`

	// DefaultEnv is the name of the default environment which is used to
	// deploy feature branches when auto deploy is enabled.
	DefaultEnv string `json:"defaultEnv"`

	// IsDefault specifies whether the app is a default project or not.
	IsDefault bool `json:"-"`

	// Runtime specifies the application runtime environment (e.g. nodejs10.x)
	Runtime string `json:"-"`

	// DeployTrigger represents the hash to trigger to a deployment.
	DeployTrigger string `json:"-" db:"deploy_trigger"`

	// defaultBranch represents the default branch of the repository.
	defaultBranch string

	// privateKey is the private key associated with the app, that is
	// used for bitbucket authentication.
	privateKey *utils.PrivateKey

	// User contains the user object.
	user *user.User `json:"-"`
}

// MyApp is used as a way to marshal the app responses.
// TODO: Remove this and use a proper serializer.
type MyApp struct {
	*App
}

// MyAppJSON is a wrapper around myapp to avoid recursion.
type MyAppJSON MyApp

// MarshalJSON implements the marshaler interface.
func (ma MyApp) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		MyAppJSON
		Endpoint   string `json:"endpoint"`
		AutoDeploy string `json:"autoDeploy,omitempty"`
	}{
		MyAppJSON:  MyAppJSON(ma),
		AutoDeploy: ma.AutoDeploy.ValueOrZero(),
	})
}

// New creates a new app instance.
func New(uid types.ID) *App {
	secret, _ := utils.Encrypt([]byte(utils.RandomToken(32)))

	return &App{
		UserID:       uid,
		ClientID:     utils.RandomToken(16),
		ClientSecret: secret,
		DisplayName:  GenerateDisplayName(),
		DefaultEnv:   config.AppDefaultEnvironmentName,
		Runtime:      config.DefaultNodeRuntime,
		privateKey:   utils.NewPrivateKey(),
	}
}

// GenerateDisplayName generates a new display name.
func GenerateDisplayName() string {
	if config.IsTest() {
		return "testdn-501bx"
	}

	name := strings.ToLower(randomdata.SillyName())
	token := strings.ToLower(utils.RandomToken(6))

	return fmt.Sprintf("%s-%s", name, token)
}

// PrivateKey returns the app private key. If the private key
// is not fetched yet, it will fetch it from the db and cache it.
func (a *App) PrivateKey(ctx context.Context) *utils.PrivateKey {
	if a.privateKey == nil {
		store := NewStore()

		if err := store.privateKey(ctx, a); err != nil {
			a.privateKey = utils.NewPrivateKey()
			_ = store.savePrivateKey(ctx, a)
		}
	}

	return a.privateKey
}

// JSON returns a map that is ready to be sent to the frontend.
func (a *App) JSON() map[string]any {
	return map[string]any{
		"id":           a.ID.String(),
		"userId":       a.UserID.String(),
		"teamId":       a.TeamID.String(),
		"repo":         a.Repo,
		"isBare":       a.Repo == "",
		"displayName":  a.DisplayName,
		"defaultEnv":   a.DefaultEnv,
		"defaultEnvId": a.DefaultEnvID.String(),
		"createdAt":    a.CreatedAt.UnixStr(),
	}
}

// IsGithub returns true if the application repotisory is hosted on GitHub.
func (a *App) IsGithub() bool {
	return strings.HasPrefix(a.Repo, "github")
}

// DefaultBranch returns the default branch of the repository.
func (a *App) DefaultBranch() (db string) {
	if a.defaultBranch != "" {
		return a.defaultBranch
	}

	var err error

	if strings.HasPrefix(a.Repo, "github/") {
		db, err = github.DefaultBranch(a.Repo)
	}

	if strings.HasPrefix(a.Repo, "gitlab/") {
		if client, _ := gitlab.NewClient(a.UserID); client != nil {
			db, err = client.DefaultBranch(a.Repo)
		}
	}

	if strings.HasPrefix(a.Repo, "bitbucket/") {
		if client, _ := bitbucket.NewClient(a.UserID); client != nil {
			db, err = client.DefaultBranch(a.Repo)
		}
	}

	if err != nil {
		if !strings.Contains(err.Error(), "Repository is not accessible.") {
			slog.Errorf("default-branch: %v", err)
		}
	}

	return db
}

// GitCreds returns the credentials for the git repo.
func (a *App) GitCreds(ctx context.Context) (string, error) {
	// TODO: mock these
	if config.IsTest() {
		return "some-token", nil
	}

	// For bitbucket we're gonna use the access key
	if strings.HasPrefix(a.Repo, "bitbucket/") {
		cnf := admin.MustConfig()

		if cnf.IsBitbucketEnabled() && cnf.AuthConfig.Bitbucket.DeployKey != "" {
			return cnf.AuthConfig.Bitbucket.DeployKey, nil
		}

		client, err := bitbucket.NewClient(a.UserID)

		if err != nil || client == nil {
			return "", err
		}

		app := &bitbucket.App{
			ID:         a.ID,
			Repo:       a.Repo,
			Secret:     a.Secret(),
			PrivateKey: a.PrivateKey(ctx),
		}

		// Ensure that webhooks, and deploy keys are installed
		if err := client.InstallWebhooks(app); err != nil {
			return "", err
		}

		pkey := a.PrivateKey(ctx)
		creds := fmt.Sprintf("%s|%s|%s|", "git", pkey.SSHPubKey(), pkey.SSHPrivKey())
		return utils.EncodeToString([]byte(creds)), nil
	}

	// For github, we use the github app authentication.
	// We have several private keys associated with our account, we need
	// to sign these requests with the required payload.
	if strings.HasPrefix(a.Repo, "github/") {
		client, err := github.NewApp(a.Repo)

		if err != nil || client == nil {
			// Check if the repository is public, in that case we can still deploy.
			if is, _ := github.IsPublicRepo(a.Repo); is {
				return "", nil
			}

			return "", err
		}

		return client.Token(context.Background())
	}

	// In this case we need to get the oauth2 token.
	if strings.HasPrefix(a.Repo, "gitlab/") {
		client, err := gitlab.NewClient(a.UserID)

		if err != nil || client == nil {
			return "", err
		}

		// Ensure that webhooks are installed
		if _, err := client.InstallWebhooks(a.Repo); err != nil {
			return "", err
		}

		return client.Token(), nil
	}

	return "", ErrRepoInvalidProvider
}

// Validate impleents model.Validate interface.
func (a *App) Validate() *shttperr.ValidationError {
	err := &shttperr.ValidationError{}

	// Validate the repository
	if a.Repo != "" {
		isBitbucket := strings.HasPrefix(a.Repo, "bitbucket/")
		isGithub := strings.HasPrefix(a.Repo, "github/")
		isGitlab := strings.HasPrefix(a.Repo, "gitlab/")

		if !isBitbucket && !isGithub && !isGitlab {
			err.SetError("repo", ErrRepoInvalidProvider.Error())
		}
	}

	matched, _ := regexp.MatchString(`^[\w0-9-]+$`, a.DisplayName)

	if !matched {
		err.SetError("displayName", ErrInvalidDisplayName.Error())
	}

	if match, _ := regexp.MatchString("--", a.DisplayName); match {
		err.SetError("displayName", ErrDoubleHypenDisplayName.Error())
	}

	if a.DisplayName == "api" || a.DisplayName == "stormkit" {
		err.SetError("displayName", "Display name is not allowed")
	}

	autoDeploy := strings.ToLower(a.AutoDeploy.ValueOrZero())

	switch autoDeploy {
	case "", "disabled", "commit", "pull_request":
		break
	default:
		err.SetError("autoDeploy", ErrInvalidAutoDeployValue.Error())
	}

	return err.ToError()
}

// Secret generates a new secret from the app id. When
// decrypted, resolves to the app id.
func (a *App) Secret() string {
	return utils.EncryptID(a.ID)
}

// User fetches the user object associated with this app. This is
// the owner of the application.
func (a *App) User() *user.User {
	if a.user == nil {
		user, err := user.NewStore().TeamOwner(a.TeamID)

		if err != nil {
			slog.Errorf("error while fetching user by team: %v, teamId=%s", err, a.TeamID)
			return user
		}

		a.user = user
	}

	return a.user
}
