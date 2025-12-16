package deployservice

import (
	"encoding/json"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var _mockService DeployerService

type SendPayloadArgs struct {
	EncryptedMsg string
	DeploymentID types.ID
}

type DeployerService interface {
	SendPayload(args SendPayloadArgs) error
	StopDeployment(runID int64) error
}

type DownloadResult struct {
	TempDir   string
	ClientDir string
	ServerZip string
	APIZip    string
}

// ClientConfig holds the configuration for the application.
type ClientConfig struct {
	// Repo is the repository to clone.
	Repo string `json:"repo"`

	// Slug is the repository name (owner / repository name)
	Slug string `json:"slug"`

	// AccessToken is the token to clone the repository.
	AccessToken string `json:"accessToken"`
}

// BuildConfig represents the build configuration for this deployment.
type BuildConfig struct {
	// Env is the environment to deploy the application. By default it is production.
	Env string `json:"env"`

	// The branch to deploy.
	Branch string `json:"branch"`

	// ShouldPublish specifies whether the deployment should be published to external storages
	// when they are enabled. If they are not enabled this has no effect.
	ShouldPublish bool `json:"shouldPublish"`

	// BuildCmd is the build command to execute.
	BuildCmd string `json:"buildCmd"`

	// ServerCmd is the command to run the Node.js application.
	ServerCmd string `json:"serverCmd,omitempty"`

	// ServerFolder is the output folder for the server side application.
	ServerFolder string `json:"serverFolder,omitempty"`

	// DistFolder is the folder which the client is built.
	DistFolder string `json:"distFolder"`

	// Vars is the environment variables that will be passed to the node builders.
	Vars map[string]string `json:"vars"`

	// The obfuscated id of the current deployment.
	DeploymentID string `json:"deploymentId"`

	// The obfuscated id of the current application.
	AppID string `json:"appId"`

	// The obfuscated id of the current environment.
	EnvID string `json:"envId"`

	// The current working directory relative to the repository root.
	WorkDir string `json:"workDir"`

	// The install command. Defaults to `npm install | yarn | pnpm install | bun install`
	InstallCmd string `json:"installCmd"`

	// Relative path (from working directory) to the headers file that will be parsed.
	HeadersFile string `json:"headersFile"`

	// Relative path (from working directory) to the redirects file that will be parsed.
	RedirectsFile string `json:"redirectsFile"`

	// Relative path (from repository root) to the api folder. Default is `/api`.
	APIFolder string `json:"apiFolder"`

	// List of status check commands to execute after the deployment is complete.
	StatusChecks []buildconf.StatusCheck `json:"statusChecks"`

	// MigrationsFolder is the path to the migrations folder, if any.
	MigrationsFolder string `json:"migrationsFolder"`
}

// DeploymentMessage represents a deployment payload.
type DeploymentMessage struct {
	// Client is the configuration regarding the application
	// and exchange settings.
	Client ClientConfig `json:"client"`

	// Build is the configuration regarding the deployment.
	Build BuildConfig `json:"build"`

	Config *config.RunnerConfig `json:"config"`

	Canary *bool `json:"canary,omitempty"`
}

// NewDeploy returns a new Deployment object.
func NewDeploy() *DeploymentMessage {
	return &DeploymentMessage{}
}

// FromEncrypted returns a deployment message from the encrypted message.
func FromEncrypted(msg string) (*DeploymentMessage, error) {
	decoded, err := utils.DecodeString(msg)

	if err != nil {
		return nil, err
	}

	decrypted, err := utils.Decrypt(decoded)

	if err != nil {
		return nil, err
	}

	dm := &DeploymentMessage{}

	if err = json.Unmarshal(decrypted, dm); err != nil {
		return nil, err
	}

	return dm, nil
}

// Encrypt returns an encrypted string representation of the deployment message.
func (dm DeploymentMessage) Encrypt() (string, error) {
	marshaled, err := json.Marshal(dm)

	if err != nil {
		return "", err
	}

	encrypted, err := utils.Encrypt(marshaled)

	if err != nil {
		return "", err
	}

	encoded := utils.EncodeToString(encrypted)

	return encoded, nil
}

func SetMockService(service DeployerService) {
	_mockService = service
}

// Service returns the DeployerService associated with this Stormkit Instance.
// The `Service` is configured through an environment variable.
// Github is the default service.
func Service() DeployerService {
	if config.IsTest() && _mockService != nil {
		return _mockService
	}

	switch config.Get().Deployer.Service {
	case "local":
		return Local()
	default:
		return Github()
	}
}
