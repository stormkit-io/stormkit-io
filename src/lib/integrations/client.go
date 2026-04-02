package integrations

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/aws/smithy-go/middleware"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type GetFileArgs struct {
	Location     string
	FileName     string
	DeploymentID types.ID
}

type GetFileResult struct {
	Size        int64
	ContentType string
	Content     []byte
}

type InvokeArgs struct {
	ARN          string            // Function ARN
	Body         io.ReadCloser     // Request body
	Method       string            // Request method
	HostName     string            // Host Name
	URL          *url.URL          // Request url
	Headers      http.Header       // Request headers
	Command      string            // If provided, this will run as a process manager. This only works on local environments.
	CaptureLogs  bool              // Whether to tell the handlers to capture logs
	EnvVariables map[string]string // This is required for server actions
	IsPublished  bool              // Whether the deployment is published or not
	AppID        types.ID
	EnvID        types.ID
	DeploymentID types.ID
	Context      map[string]any // Additional context to pass to the function
	QueueLog     func(*Log)     // Queue logs for later processing
}

type Log struct {
	Timestamp int64  `json:"ts"`
	Message   string `json:"msg"`
	Level     string `json:"level"`
}

type InvokeResult struct {
	Logs         []Log
	Body         []byte
	StatusCode   int
	Headers      http.Header
	ErrorMessage string
	ErrorStack   string
}

type FunctionRequest struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"` // URL is the relative path + query string
	Path        string            `json:"path"`
	Body        string            `json:"body,omitempty"`
	Query       url.Values        `json:"query,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	RawHeaders  []string          `json:"rawHeaders,omitempty"`
	CaptureLogs bool              `json:"captureLogs,omitempty"`
	Context     map[string]any    `json:"context,omitempty"`
}

type FunctionResponse struct {
	Headers      map[string]any `json:"headers"` // map[string]string | map[string][]string
	Body         string         `json:"body"`
	Buffer       string         `json:"buffer"` // base64 encoded responses are returned this way
	Status       int            `json:"status"` // alias for statusCode
	StatusCode   int            `json:"statusCode"`
	ErrorMessage string         `json:"errorMessage"`
	ErrorStack   string         `json:"errorStack"`
	Logs         []Log          `json:"logs"`
}

type DeleteArtifactsArgs struct {
	APILocation      string
	FunctionLocation string
	StorageLocation  string
}

type ClientInterface interface {
	Name() string
	Upload(UploadArgs) (*UploadResult, error)
	Invoke(InvokeArgs) (*InvokeResult, error)
	GetFile(GetFileArgs) (*GetFileResult, error)
	DeleteArtifacts(context.Context, DeleteArtifactsArgs) error
}

var CachedClient ClientInterface
var clientOnce sync.Once

// SetDefaultClient sets the client that will be returned by Client
// method. This method is mainly used for testing purposes. For environments
// other than tests, this method does nothing.
func SetDefaultClient(client ClientInterface) {
	if !config.IsTest() {
		return
	}

	CachedClient = client
	clientOnce = sync.Once{}
}

type ClientArgs struct {
	Provider     string
	AccessKey    string
	SecretKey    string
	SessionToken string
	Profile      string
	Region       string
	Middlewares  []func(stack *middleware.Stack) error
}

// Client returns the ClientInterface based on the environment configuration.
// For instance, if AWS_* environment variables are declared, this function will
// return the AWS client. After the first call, the client returns the
// cached ClientInterface. clientOnce guarantees that the initialisation runs
// exactly once even under concurrent callers.
func Client(args ...ClientArgs) ClientInterface {
	if CachedClient != nil {
		return CachedClient
	}

	cfg := config.Get()
	opts := ClientArgs{}

	if len(args) > 0 {
		opts = args[0]
	}

	var initErr error

	clientOnce.Do(func() {
		if cfg.AWS != nil || opts.Provider == config.ProviderAWS {
			CachedClient, initErr = AWS(opts, nil)
		} else if cfg.Alibaba != nil || opts.Provider == config.ProviderAlibaba {
			CachedClient, initErr = Alibaba(opts)
		} else if cfg.Deployer.IsLocal() || opts.Provider == config.ProviderFilesys {
			CachedClient = Filesys()
		}

		if CachedClient != nil {
			slog.Infof("integrations client: %s", CachedClient.Name())
		}
	})

	if initErr != nil {
		panic(initErr)
	}

	return CachedClient
}
