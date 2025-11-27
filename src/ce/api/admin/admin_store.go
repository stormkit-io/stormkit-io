package admin

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var stmt = struct {
	selectConfig string
	upsertConfig string
	deleteConfig string
}{
	selectConfig: `
		SELECT config_data FROM stormkit_config LIMIT 1;
	`,

	upsertConfig: `
		INSERT INTO stormkit_config
			(config_id, config_data)
		VALUES
			(1, $1)
		ON CONFLICT
			(config_id)
		DO UPDATE SET
			config_data = EXCLUDED.config_data;
	`,

	deleteConfig: `
		DELETE FROM stormkit_config;
	`,
}

// store represents a store for admin operations.
type store struct {
	*database.Store
}

func Store() *store {
	return &store{
		Store: database.NewStore(),
	}
}

var cachedConfig *InstanceConfig
var cachedConfigMux = &sync.Mutex{}

// MustConfig is like Config but logs an error if it fails.
func MustConfig() InstanceConfig {
	cnf, err := Store().Config(context.Background())

	if err != nil {
		slog.Errorf("error getting instance config: %v", err)
	}

	return cnf
}

var once sync.Once

// Config returns the Volume Configuration for the given Stormkit instance.
func (s *store) Config(ctx context.Context) (InstanceConfig, error) {
	cachedConfigMux.Lock()
	cnf := cachedConfig
	cachedConfigMux.Unlock()

	if cnf != nil {
		return *cnf, nil
	}

	row, err := s.QueryRow(ctx, stmt.selectConfig)

	if err != nil {
		return InstanceConfig{}, err
	}

	cnf = &InstanceConfig{}

	if err := row.Scan(cnf); err != nil {
		if err == sql.ErrNoRows {
			return InstanceConfig{}, nil
		}

		return InstanceConfig{}, err
	}

	// Ensure DomainConfig is always set, this adds backwards compatibility for instances
	// that were created before the DomainConfig was added to the InstanceConfig.
	if cnf.DomainConfig == nil {
		cnf.DomainConfig = &DomainConfig{
			API:      os.Getenv("STORMKIT_API_URL"),
			App:      os.Getenv("STORMKIT_APP_URL"),
			Dev:      os.Getenv("STORMKIT_DEV_URL"),
			Health:   os.Getenv("STORMKIT_HEALTH_URL"),
			Webhooks: os.Getenv("STORMKIT_WEBHOOKS_URL"),
		}

		parsed := GetParsedURL(
			os.Getenv("STORMKIT_DOMAIN"),
			os.Getenv("STORMKIT_URL"),
			cnf.DomainConfig.Dev,
			fmt.Sprintf("http://localhost:%s", config.LocalhostPort),
		)

		if parsed != nil {
			if cnf.DomainConfig.Health == "" {
				cnf.DomainConfig.Health = fmt.Sprintf("%s://health.%s", parsed.Scheme, parsed.Host)
			}

			if cnf.DomainConfig.Dev == "" {
				cnf.DomainConfig.Dev = fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)
			}

			if cnf.DomainConfig.App == "" {
				cnf.DomainConfig.App = fmt.Sprintf("%s://stormkit.%s", parsed.Scheme, parsed.Host)
			}

			if cnf.DomainConfig.API == "" {
				cnf.DomainConfig.API = fmt.Sprintf("%s://api.%s", parsed.Scheme, parsed.Host)
			}
		}

		cnf.DomainConfig.API = utils.NormalizeURL(cnf.DomainConfig.API)
		cnf.DomainConfig.App = utils.NormalizeURL(cnf.DomainConfig.App)
		cnf.DomainConfig.Dev = utils.NormalizeURL(cnf.DomainConfig.Dev)
		cnf.DomainConfig.Health = utils.NormalizeURL(cnf.DomainConfig.Health)
		cnf.DomainConfig.Webhooks = utils.NormalizeURL(cnf.DomainConfig.Webhooks)
	}

	once.Do(func() {
		slog.Infof("api: %s", cnf.DomainConfig.API)
		slog.Infof("ui:  %s", cnf.DomainConfig.App)
		slog.Infof("dev: %s",
			strings.Replace(
				strings.Replace(cnf.DomainConfig.Dev, "http://", "http://*.", 1),
				"https://", "https://*.", 1,
			))
	})

	// Ensure AuthConfig is always set, this adds backwards compatibility for instances
	// that were created before the AuthConfig was added to the InstanceConfig.
	if cnf.AuthConfig == nil {
		secrets := config.Secrets()
		ghPrivKey := ""
		ghAppName := utils.GetString(os.Getenv("GITHUB_APP_NAME"), os.Getenv("GITHUB_ACCOUNT"))
		glClientID := os.Getenv("GITLAB_CLIENT_ID")
		bitbucketID := os.Getenv("BITBUCKET_CLIENT_ID")

		if decoded, err := utils.DecodeString(secrets["GITHUB_PRIV_KEY"]); err == nil && decoded != nil {
			ghPrivKey = string(decoded)
		}

		cnf.AuthConfig = &AuthConfig{}

		if ghAppName != "" {
			cnf.AuthConfig.Github = GithubConfig{
				AppID:        utils.StringToInt(os.Getenv("GITHUB_APP_ID")),
				Account:      utils.GetString(os.Getenv("GITHUB_APP_NAME"), os.Getenv("GITHUB_ACCOUNT")),
				ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
				ClientSecret: secrets["GITHUB_SECRET"],
				PrivateKey:   ghPrivKey,
				RunnerRepo:   utils.GetString(os.Getenv("GITHUB_RUNNER"), "stormkit-io/deployer-service"),
				RunnerToken:  secrets["GITHUB_APP_TOKEN"],
			}
		}

		if glClientID != "" {
			cnf.AuthConfig.Gitlab = GitlabConfig{
				ClientID:     glClientID,
				ClientSecret: secrets["GITLAB_SECRET"],
				RedirectURL: utils.GetString(
					os.Getenv("GITLAB_REDIRECT_URL"),
					cnf.ApiURL("/auth/gitlab/callback"),
				),
			}

		}

		if bitbucketID != "" {
			cnf.AuthConfig.Bitbucket = BitbucketConfig{
				ClientID:     bitbucketID,
				ClientSecret: secrets["BITBUCKET_SECRET"],
				DeployKey:    os.Getenv("BITBUCKET_DEPLOY_KEY"),
			}
		}
	}

	cachedConfigMux.Lock()
	cachedConfig = cnf
	cachedConfigMux.Unlock()

	return *cnf, nil
}

// UpsertConfig creates or updates the volumes config.
// This method has the side effect of resetting the cached config.
func (s *store) UpsertConfig(ctx context.Context, cnf InstanceConfig) error {
	_, err := s.Exec(ctx, stmt.upsertConfig, cnf)

	if err == nil {
		// ResetCache for this instance immediately
		ResetCache(ctx)
		return rediscache.Service().Broadcast(rediscache.EventInvalidateAdminCache)
	}

	return err
}

// DeleteConfig deletes the admin config.
// This method has the side effect of resetting the cached config.
func (s *store) DeleteConfig(ctx context.Context) error {
	_, err := s.Exec(ctx, stmt.deleteConfig)

	if err == nil {
		// ResetCache for this instance immediately
		ResetCache(ctx)
		return rediscache.Service().Broadcast(rediscache.EventInvalidateAdminCache)
	}

	return err
}

// ResetCache is a function that invalidates the admin configuration cache.
// This is used for redis pub/sub.
func ResetCache(ctx context.Context, payload ...string) {
	slog.Debug(slog.LogOpts{
		Msg:   "invalidating admin config cache",
		Level: slog.DL2,
	})

	cachedConfigMux.Lock()
	cachedConfig = nil
	cachedConfigMux.Unlock()

	ResetLicense()
}

// SetConfig sets the cached admin configuration.
// This is used by tests.
func SetConfig(cnf *InstanceConfig) {
	if !config.IsTest() {
		panic("admin.SetConfig can only be used in tests")
	}

	cachedConfigMux.Lock()
	cachedConfig = cnf
	cachedConfigMux.Unlock()
}

func GetParsedURL(candidates ...string) *url.URL {
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}

		if !strings.HasPrefix(candidate, "http") {
			candidate = fmt.Sprintf("https://%s", candidate)
		}

		u, err := url.Parse(candidate)

		if err != nil {
			slog.Errorf("cannot parse STORMKIT_DOMAIN: %s", err.Error())
			os.Exit(1)
		}

		if u.Scheme == "" {
			u.Scheme = "https"
		}

		return u
	}

	return nil
}
