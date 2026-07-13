package hosting

import (
	"context"
	"regexp"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/router"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/mise"
	"go.uber.org/zap"
)

// RegisterListeners registers listeners for the hosting machine.
func RegisterListeners() {
	service := rediscache.Service()

	handlers := map[string]rediscache.Handler{
		rediscache.EventInvalidateHostingCache: InvalidateCache,
		rediscache.EventInvalidateAdminCache:   invalidateAdminCache,
		rediscache.EventRuntimesInstall:        admin.InstallDependencies,
		rediscache.EventMiseUpdate:             mise.AutoUpdate,
	}

	for event, handler := range handlers {
		if err := service.SubscribeAsync(event, handler); err != nil {
			slog.Errorf("failed to register event %s: %v", event, err)
		}
	}
}

func invalidateAdminCache(ctx context.Context, payload ...string) {
	admin.ResetCache(ctx, payload...)

	// Make sure to reset internal endpoints
	mux.Lock()
	internalEndpoints = nil
	mux.Unlock()

	// Make sure to reset cors settings as well
	router.ResetCors()

	appCacheMu.Lock()
	appCache = map[string]*CachedConfig{}
	appCacheMu.Unlock()
}

// InvalidateCache is a function that invalidates the domain configuration cache.
func InvalidateCache(ctx context.Context, payload ...string) {
	slog.Debug(slog.LogOpts{
		Msg:   "received cache invalidating message",
		Level: slog.DL2,
		Payload: []zap.Field{
			zap.Strings("payload", payload),
		},
	})

	re, err := regexp.Compile(payload[0])

	if err != nil {
		slog.Errorf("error while creating regexp pattern: %v", err)
		return
	}

	appCacheMu.Lock()

	// Note: We do not remove the custom certificate stored in cache because
	// certmagic uses file content to compute the hash. Therefore calling
	// CacheUnmanagedCertificatePEMBytes will not create duplicate records.
	// Also, whenever there is a new certificate, the old one is overwritten
	// because certmagic uses Domain name to store associate the active certificate.
	for k := range appCache {
		if re.Match([]byte(k)) {
			delete(appCache, k)
		}
	}

	appCacheMu.Unlock()
}
