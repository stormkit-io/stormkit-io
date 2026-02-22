package hosting

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// VersionCookieName represents the name of the cookie that
// will determine version for the user. If this value is empty,
// or does not match any of the deployments, then it will be reset.
const VersionCookieName = "sk_variant"

// SettingsCookieName is the name of the cookie to which stores
// the application's settings.
const SettingsCookieName = "sk_settings"

const inMemoryCacheTTL = 10 * time.Minute

type AppCache = map[string]*CachedConfig

type CachedConfig struct {
	Config         []*appconf.Config
	InMemorySince  time.Time
	CustomCertHash string
}

var appCache AppCache
var appCacheMu sync.Mutex

func init() {
	appCache = AppCache{}

	go func() {
		for {
			for hostName, cache := range appCache {
				if time.Since(cache.InMemorySince) > inMemoryCacheTTL {
					appCacheMu.Lock()
					delete(appCache, hostName)
					appCacheMu.Unlock()
				}
			}

			time.Sleep(time.Minute * 1)
		}
	}()
}

// Host represents a host
type Host struct {
	Config *appconf.Config

	// Name is the host name. It is obtained from the headers.
	Name string

	// Whether the host is a stormkit subdomain or not.
	IsStormkitSubdomain bool

	Request *shttp.RequestContext
}

// FetchAppConf fetches the config for the host name from the database.
// If the config is found in local cache, it's returned from the local cache.
func FetchAppConf(hostName string) ([]*appconf.Config, error) {
	appCacheMu.Lock()
	confFromCache := appCache[hostName]
	appCacheMu.Unlock()

	if confFromCache != nil {
		// Frequently used caches should not be invalidated.
		confFromCache.InMemorySince = time.Now()
		return confFromCache.Config, nil
	}

	configs, err := appconf.FetchConfig(hostName)

	if err != nil {
		slog.Errorf("Error fetching config %v for host %s", err, hostName)
	}

	cached := &CachedConfig{
		Config:        configs,
		InMemorySince: time.Now(),
	}

	appCacheMu.Lock()
	appCache[hostName] = cached
	appCacheMu.Unlock()

	if len(configs) > 0 && configs[0].CertKey != "" && configs[0].CertValue != "" {
		hash, err := CertMagic().
			CacheUnmanagedCertificatePEMBytes(
				context.Background(),
				[]byte(configs[0].CertValue),
				[]byte(configs[0].CertKey),
				nil,
			)

		if err != nil {
			slog.Errorf("cannot configure custom certificate: %s", err.Error())
		} else {
			slog.Infof("custom certificate cache key: %s", hash)
			cached.CustomCertHash = hash
		}
	}

	return configs, err
}

// RequestConfig requests the config from api and assigns it to the
// .Config field. It also chooses the right version if there are multiple version.
func (h *Host) RequestConfig() error {
	confs, err := FetchAppConf(h.Name)

	if err != nil {
		return err
	}

	if len(confs) == 0 {
		return nil
	}

	h.Config = h.ChooseVersion(confs)
	return nil
}

// ChooseVersion chooses one version from possible multiple configs.
// It is used for doing A/B testing.
func (h *Host) ChooseVersion(confs []*appconf.Config) *appconf.Config {
	if len(confs) == 0 {
		return nil
	}

	if len(confs) == 1 {
		return confs[0]
	}

	variant, err := h.Request.Cookie(VersionCookieName)

	if err == nil && variant != nil {
		for _, c := range confs {
			if c.DeploymentID.String() == variant.Value {
				return c
			}
		}
	}

	rand := float64(utils.Random(0, 100))

	for _, c := range confs {
		if rand = rand - c.Percentage; rand <= 0 {
			return c
		}
	}

	return confs[0]
}

// HostNameIdentifier returns either the domain name, or the subdomain
// from the managed domain. For instance, if the host name is a custom
// domain such as example.org, it returns example.org. If it's a managed
// subdomain such as my-app--staging.stormkit.dev it returns my-app--staging.
func HostNameIdentifier(name string) string {
	pieces := strings.Split(admin.MustConfig().DomainConfig.Dev, "//")

	if len(pieces) > 0 && strings.HasSuffix(name, pieces[1]) {
		return strings.Replace(name, fmt.Sprintf(".%s", pieces[1]), "", 1)
	}

	if strings.HasPrefix(name, "www.") {
		name = strings.Replace(name, "www.", "", 1)
	}

	return name
}
