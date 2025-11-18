package config

import (
	"os"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stripe/stripe-go"
)

var LocalhostPort = utils.GetString(os.Getenv("STORMKIT_HTTP_PORT"), "8888")
var IsWindows = runtime.GOOS == "windows"

const (
	// List of supported runtimes for serverless functions
	NodeRuntime12      = "nodejs12.x"
	NodeRuntime14      = "nodejs14.x"
	NodeRuntime16      = "nodejs16.x"
	NodeRuntime18      = "nodejs18.x"
	NodeRuntime20      = "nodejs20.x"
	NodeRuntime22      = "nodejs22.x"
	BunRuntime1        = "bun1.x"
	DefaultNodeRuntime = NodeRuntime22

	// List of providers
	ProviderAWS     = "aws"
	ProviderAlibaba = "alibaba"
	ProviderFilesys = "filesys"

	// Deployer services
	DeployerServiceLocal = "local"
)

// Allowed environments
const (
	EnvProd  = "prod"
	EnvLocal = "local"
)

// These are set by the build
var (
	appSecret     string
	cleanDatabase string // (true|false) Whether to wipe the db (only local environments)
	projectRoot   string
	edition       string // cloud | self-hosted | development
	hash          string // git hash
	version       string // tag like v1.7.30
)

// Export available package names
var (
	PackageFree     = "free"
	PackagePremium  = "premium"
	PackageUltimate = "ultimate"
)

// AppDefaultEnvironmentName is the name of the default environment
// that comes with every app in Stormkit.
const AppDefaultEnvironmentName = "production"

// AwsConfig has the configs for aws.
type AwsConfig struct {
	Region         string
	AccountID      string
	StorageBucket  string
	LambdaRoleName string
}

type AlibabaConfig struct {
	AccountID     string
	AccessKey     string
	SecretKey     string
	StorageBucket string
	Region        string
}

type RunnerConfig struct {
	AccessKey     string `json:"accessKey,omitempty"`
	SecretKey     string `json:"secretKey,omitempty"`
	Provider      string `json:"provider,omitempty"` // One of AWS | Alibaba
	BucketName    string `json:"bucketName,omitempty"`
	Region        string `json:"region,omitempty"`
	LambdaRole    string `json:"lambdaRole,omitempty"`
	AccountID     string `json:"accountId,omitempty"`
	ErrorsChannel string `json:"errorsChannel,omitempty"`
	Concurrency   int    `json:"-"`
	MaxGoRoutines int    `json:"maxGoRoutines,omitempty"`
}

type HttpTimeoutsConfig struct {
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DbConfigTimeouts struct {
	ConnectTimeout time.Duration
	MaxLifetime    int // maximum amount of time that a connection can be reused before it is closed and replaced with a new connection
	MaxIdleConns   int // maximum number of idle connections that can be kept in the connection pool
	MaxOpenConns   int // maximum number of open connections that can be used at the same time
}

// DatabaseConfig has the configs for the database.
type DatabaseConfig struct {
	WipeOnStart bool
}

// DeployerConfig contains config for the deployer app.
type DeployerConfig struct {
	Service    string // Can be `local` | `github`
	StorageDir string // Directory to store deployments
	Executable string // Directory to our deploy.js
}

type TrackingConfig struct {
	Prometheus     bool
	PrometheusPort string
}

func (dc DeployerConfig) IsLocal() bool {
	return dc.Service == "local"
}

// StripeConfig represents the stripe configuration.
type StripeConfig struct {
	ClientID       string
	ClientSecret   string
	WebhooksSecret string
}

type ReportingConfig struct {
	DiscordErrorChannel              string
	DiscordDeploymentsSuccessChannel string
	DiscordDeploymentsFailedChannel  string
	DiscordSignupsChannel            string
	DiscordProductionChannel         string
}

type VersionConfig struct {
	Hash string
	Tag  string
}

type limits struct {
	BuildMinutes        int64
	BandwidthInBytes    int64
	StorageInBytes      int64
	FunctionInvocations int64
}

const OneGB = 1e+9
const OneTB = 1e+12

// These prices are multipled for each seat
var Limits = map[string]limits{
	PackageFree: {
		BuildMinutes:        300,
		BandwidthInBytes:    100 * OneGB,
		StorageInBytes:      100 * OneGB,
		FunctionInvocations: 500 * 1000, // 500 thousand
	},
	PackagePremium: {
		BuildMinutes:        1000,
		BandwidthInBytes:    OneTB,
		StorageInBytes:      OneTB,
		FunctionInvocations: 1500 * 1000, // 1.5 million
	},
	PackageUltimate: {
		BuildMinutes:        5000,
		BandwidthInBytes:    5 * OneTB,
		StorageInBytes:      5 * OneTB,
		FunctionInvocations: 5000 * 1000, // 5 million
	},
}

// Config is the root object for the configuration.
type Config struct {
	AWS              *AwsConfig
	Alibaba          *AlibabaConfig
	Database         *DatabaseConfig
	Deployer         *DeployerConfig
	Stripe           *StripeConfig
	Reporting        *ReportingConfig
	Runner           *RunnerConfig
	Tracking         *TrackingConfig
	InstanceID       string // The ID of the instance that is randomly assigned during start time
	AppSecret        string
	APIKey           string // The API key used to access several endpoints (for dedicated instances)
	Env              string
	Version          VersionConfig
	Hash             string
	RedisAddr        string
	Secrets          map[string]string
	HTTPTimeouts     *HttpTimeoutsConfig
	DbConfigTimeouts *DbConfigTimeouts
}

var c *Config

func init() {
	if root := os.Getenv("STORMKIT_PROJECT_ROOT"); root != "" {
		projectRoot = root
	}

	slog.SetConfig(&slog.Config{
		Disabled: IsTest(),
		Colorful: IsDevelopment(),
	})

	if edition == "" {
		edition = "development"
	}
}

// New returns a new Config instance.
func New() *Config {
	secrets := Secrets()
	env := Env()

	// Prepare configuration for Deployer Service
	deployer := deployerService()
	storageDir := ""
	executable := ""

	if deployer == "local" {
		defaultStorageDir := "/shared/deployments"
		defaultExecutable := "/bin/runner"

		if IsDevelopment() {
			defaultStorageDir = path.Join(projectRoot, "build")
			defaultExecutable = path.Join(projectRoot, "bin", "runner")
		}

		storageDir = get(os.Getenv("STORMKIT_DEPLOYER_DIR"), defaultStorageDir)
		executable = get(os.Getenv("STORMKIT_DEPLOYER_EXECUTABLE"), defaultExecutable)
	}

	config := &Config{
		Database: &DatabaseConfig{
			WipeOnStart: env == "local" && cleanDatabase == "true",
		},

		Stripe: &StripeConfig{
			ClientID:       os.Getenv("STRIPE_CLIENT_ID"),
			ClientSecret:   secrets["STRIPE_SECRET"],
			WebhooksSecret: secrets["STRIPE_WH_SECRET"],
		},

		Deployer: &DeployerConfig{
			Service:    deployer,
			StorageDir: storageDir,
			Executable: executable,
		},

		Reporting: &ReportingConfig{
			DiscordErrorChannel:              os.Getenv("DISCORD_ERRORS_CHANNEL"),
			DiscordSignupsChannel:            os.Getenv("DISCORD_SIGNUPS_CHANNEL"),
			DiscordDeploymentsFailedChannel:  os.Getenv("DISCORD_DEPLOYMENTS_FAILED_CHANNEL"),
			DiscordDeploymentsSuccessChannel: os.Getenv("DISCORD_DEPLOYMENTS_SUCCESS_CHANNEL"),
			DiscordProductionChannel:         os.Getenv("DISCORD_PRODUCTION_CHANNEL"),
		},

		HTTPTimeouts: &HttpTimeoutsConfig{
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  60 * time.Second,
		},

		DbConfigTimeouts: &DbConfigTimeouts{
			ConnectTimeout: 30 * time.Second,
			MaxLifetime:    0, // unlimited
			MaxIdleConns:   2,
			MaxOpenConns:   50,
		},

		Runner: &RunnerConfig{
			AccessKey:     os.Getenv("STORMKIT_RUNNER_ACCESS_KEY"),
			SecretKey:     os.Getenv("STORMKIT_RUNNER_SECRET_KEY"),
			Concurrency:   getInt(os.Getenv("STORMKIT_RUNNER_CONCURRENCY"), 10),
			MaxGoRoutines: getInt(os.Getenv("STORMKIT_RUNNER_PARALLEL_UPLOADS"), 25),
		},

		Tracking: &TrackingConfig{
			Prometheus:     isTrueString(os.Getenv("PROMETHEUS_METRICS")),
			PrometheusPort: get(os.Getenv("PROMETHEUS_PORT"), "2112"),
		},

		Env:       Env(),
		RedisAddr: os.Getenv("REDIS_ADDR"),
		AppSecret: AppSecret(),
		Secrets:   secrets,
		Version: VersionConfig{
			Hash: hash,
			Tag:  version,
		},
	}

	dbconf := database.DBConf{
		DBName:   secrets["POSTGRES_DB"],
		Host:     secrets["POSTGRES_HOST"],
		User:     secrets["POSTGRES_USER"],
		Password: secrets["POSTGRES_PASSWORD"],
		SSLMode:  secrets["POSTGRES_SSL"],
		Port:     secrets["POSTGRES_PORT"],
		Schema:   "skitapi",

		MaxLifetime:  time.Duration(config.DbConfigTimeouts.MaxLifetime),
		MaxOpenConns: config.DbConfigTimeouts.MaxOpenConns,
		MaxIdleConns: config.DbConfigTimeouts.MaxIdleConns,
	}

	// use test db for test env
	if IsTest() {
		config.RedisAddr = "localhost:6379"
		dbconf.DBName = "sktest"
	}

	database.Configure(dbconf)

	if os.Getenv("AWS_ACCOUNT_ID") != "" {
		config.AWS = &AwsConfig{
			Region:         os.Getenv("AWS_REGION"),
			AccountID:      os.Getenv("AWS_ACCOUNT_ID"),
			LambdaRoleName: os.Getenv("AWS_LAMBDA_ROLE_NAME"),
			StorageBucket:  os.Getenv("AWS_S3_BUCKET_NAME"),
		}
	}

	if bucketName := os.Getenv("ALIBABA_OSS_BUCKET_NAME"); bucketName != "" {
		config.Alibaba = &AlibabaConfig{
			StorageBucket: bucketName,
			Region:        os.Getenv("ALIBABA_REGION"),
			AccountID:     os.Getenv("ALIBABA_ACCOUNT_ID"), // Log on to Function Compute and see the account id
			AccessKey:     os.Getenv("ALIBABA_ACCESS_KEY_ID"),
			SecretKey:     os.Getenv("ALIBABA_ACCESS_KEY_SECRET"),
		}
	}

	if config.Reporting.DiscordErrorChannel != "" {
		config.Runner.ErrorsChannel = config.Reporting.DiscordErrorChannel
	}

	if config.AWS != nil {
		config.Runner.Provider = ProviderAWS
		config.Runner.BucketName = config.AWS.StorageBucket
		config.Runner.Region = config.AWS.Region
		config.Runner.AccountID = config.AWS.AccountID
		config.Runner.LambdaRole = config.AWS.LambdaRoleName
	} else if config.Alibaba != nil {
		config.Runner.Provider = ProviderAlibaba
		config.Runner.BucketName = config.Alibaba.StorageBucket
		config.Runner.Region = config.Alibaba.Region
		config.Runner.AccountID = config.Alibaba.AccountID

		// Note: I'm not sure about this solution. Currently, the same
		// secret/access key is used for the runner. This is not good in cases
		// where we want to use the access keys and machine-level permissions. But
		// as of writing, there is still no use case to use them combined so I'm doing
		// this dirty hack to pass the keys to the runner.
		config.Runner.AccessKey = config.Alibaba.AccessKey
		config.Runner.SecretKey = config.Alibaba.SecretKey
	} else {
		config.Runner.Provider = ProviderFilesys
	}

	if IsSelfHosted() {
		config.Stripe = nil
	}

	if config.Stripe != nil {
		stripe.Key = config.Stripe.ClientSecret
	}

	return config
}

var cnfMutex sync.Mutex

// Get returns the config object.
func Get() *Config {
	cnfMutex.Lock()
	defer cnfMutex.Unlock()

	if c == nil {
		Set(New())

		slog.Infof("stormkit environment: %s", c.Env)
		slog.Infof("stormkit version: %s", c.Version)
		slog.Infof("stormkit edition: %v", edition)
	}

	return c
}

// Set enables manipulating the Config package.
func Set(config *Config) *Config {
	c = validate(config)
	utils.SetAppKey([]byte(AppSecret()))

	return c
}

// Reset resets the config.
func Reset() {
	c = nil
}

var _cachedSecrets map[string]string

// Secrets decrypts environment variables that start with Salted_ key.
func Secrets() map[string]string {
	if _cachedSecrets != nil {
		return _cachedSecrets
	}

	secret := []byte(AppSecret())
	secretMap := map[string]string{
		"POSTGRES_SSL":  "disable",
		"POSTGRES_PORT": "5432",
	}

	for _, keyValue := range os.Environ() {
		envVar := strings.SplitN(keyValue, "=", 2)

		if len(envVar) != 2 {
			continue
		}

		envName, envValue := envVar[0], envVar[1]

		if !IsStormkitCloud() {
			secretMap[envName] = envValue
			continue
		}

		// Deprecated: do not use this method anymore. It's overengineering.
		if !strings.HasPrefix(envValue, "Salted_") {
			secretMap[envName] = envValue
			continue
		}

		// Remove the Salted_ part
		decoded, err := utils.DecodeString(envValue[7:])

		if err != nil {
			slog.Errorf("error while decoding env variable=%s, err=%v", envName, err)
			continue
		}

		decrypted, err := utils.Decrypt(decoded, secret)

		if err == nil && decrypted != nil {
			secretMap[envName] = string(decrypted)
		} else if err != nil {
			slog.Errorf("error while decrypting env variable=%s,  err=%v", envName, err)
		}
	}

	_cachedSecrets = secretMap

	return secretMap
}

func AppSecret() string {
	if appSecret == "" {
		appSecret = os.Getenv("STORMKIT_APP_SECRET")
	}

	if appSecret == "" && !IsTest() {
		panic("Missing STORMKIT_APP_SECRET. Make sure to generate a 32 characters long (alphanumeric) secret.")
	}

	return appSecret
}

// Env returns the environment.
func Env() string {
	if IsStormkitCloud() || IsSelfHosted() {
		return EnvProd
	} else {
		return EnvLocal
	}
}

// IsStormkitCloud returns true for our Cloud Managed environments (stormkit.io).
func IsStormkitCloud() bool {
	return edition == "cloud"
}

func IsProduction() bool {
	return IsStormkitCloud() || IsSelfHosted()
}

// IsTest returns true when the caller file has a `test` in its name.
func IsTest() bool {
	return strings.Contains(os.Args[0], "_test") || strings.Contains(os.Args[0], ".test")
}

// IsDevelopment returns true when it is local environment and not self-hosted.
func IsDevelopment() bool {
	return edition == "development"
}

// IsSelfHosted returns tru if the build-time 'selfHosted'
// variable is set to 'true'.
func IsSelfHosted() bool {
	return edition == "self-hosted"
}

// SetIsStormkitCloud is specifically designed for tests to manipulate
// this value. If the environment is not a test environment, the function call
// has no effect.
func SetIsStormkitCloud(value bool) {
	if !IsTest() {
		return
	}

	if value {
		edition = "cloud"
	} else {
		edition = "development"
	}
}

// SetIsSelfHosted is specifically designed for tests to manipulate
// this value. If the environment is not a test environment, the function call
// has no effect.
func SetIsSelfHosted(value bool) {
	if !IsTest() {
		return
	}

	if value {
		edition = "self-hosted"
	} else {
		edition = "development"
	}
}

func deployerService() string {
	env := os.Getenv("STORMKIT_DEPLOYER_SERVICE")

	if env != "" {
		return env
	}

	if IsStormkitCloud() {
		return "github"
	}

	return DeployerServiceLocal
}

func validate(c *Config) *Config {
	if len(c.AppSecret) != 32 {
		slog.Debug(slog.LogOpts{
			Msg:   "Warning: App secret is not 32 chars long.",
			Level: slog.DL1,
		})
	}

	return c
}

// get is a helper function to get one of the two values.
func get(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}

	return ""
}

func getInt(val string, def int) int {
	if val == "" {
		return def
	}

	valInt := utils.StringToInt(val)

	if valInt == 0 {
		return def
	}

	return valInt
}

func isTrueString(s string) bool {
	return strings.EqualFold(s, "true") || strings.EqualFold(s, "1") || strings.EqualFold(s, "yes")
}

type Runtime struct {
	Name    string
	Version string
}

// ParseRuntime parses the given runtime string (in nodejs12.x) format
// and returns a Runtime object with Name and Version properties.
// The names are in lowercase.
func ParseRuntime(runtime string) Runtime {
	rt := Runtime{}
	runtimeAndVersion := strings.Replace(runtime, ".x", "", 1)

	if strings.HasPrefix(runtime, "bun") {
		rt.Name = "bun"
		rt.Version = strings.Replace(runtimeAndVersion, "bun", "", 1)
	} else {
		rt.Name = "node"
		rt.Version = strings.Replace(runtimeAndVersion, "nodejs", "", 1)
	}

	return rt
}

// IsEnterprise returns true if the current package is enterprise.
func IsEnterprise() bool {
	return true
}
