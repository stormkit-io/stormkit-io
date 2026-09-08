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
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

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
var appConfSFGroup singleflight.Group

// fetchConfigFn is the function used to fetch app config from the database.
// It is a variable to allow overriding in tests.
var fetchConfigFn = appconf.FetchConfig

func init() {
	appCache = AppCache{}

	go func() {
		deleteAllExpired := func() {
			appCacheMu.Lock()
			defer appCacheMu.Unlock()

			for hostName, cache := range appCache {
				if time.Since(cache.InMemorySince) > inMemoryCacheTTL {
					delete(appCache, hostName)
				}
			}
		}

		for {
			deleteAllExpired()
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

func getConfFromCache(hostName string) *CachedConfig {
	appCacheMu.Lock()
	defer appCacheMu.Unlock()

	confFromCache := appCache[hostName]

	if confFromCache != nil {
		// Frequently used caches should not be invalidated.
		confFromCache.InMemorySince = time.Now()
	}

	return confFromCache
}

// FetchAppConf fetches the config for the host name from the database.
// If the config is found in local cache, it's returned from the local cache.
// singleflight ensures that concurrent cache misses for the same hostname
// result in exactly one database query, preventing thundering herd under load.
func FetchAppConf(hostName string) ([]*appconf.Config, error) {
	confFromCache := getConfFromCache(hostName)

	if confFromCache != nil {
		return confFromCache.Config, nil
	}

	v, err, _ := appConfSFGroup.Do(hostName, func() (any, error) {
		return fetchConfigFn(hostName)
	})

	if err != nil {
		slog.Errorf("Error fetching config %v for host %s", err, hostName)
		return nil, err
	}

	configs := v.([]*appconf.Config)

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

	return configs, nil
}

// RequestConfig requests the config from api and assigns it to the
// .Config field. It also chooses the right version if there are multiple version.
func (h *Host) RequestConfig() error {
	slog.Debug(slog.LogOpts{
		Msg:   "requesting config for host",
		Level: slog.DL4,
		Payload: []zap.Field{
			zap.String("host", h.Name),
		},
	})

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

// ChooseVersion returns the configuration to serve this request from.
//
// An environment serves exactly one deployment — the database holds it as a
// unique index — so there is nothing to choose between. The lookup can still
// return nothing, which is what the nil is for.
func (h *Host) ChooseVersion(confs []*appconf.Config) *appconf.Config {
	if len(confs) == 0 {
		return nil
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
