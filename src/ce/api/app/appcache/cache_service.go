package appcache

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

var DefaultCacheService CacheInterface

type CacheInterface interface {
	Reset(envID types.ID, keys ...string) error
}

type CacheService struct {
}

func Service() CacheInterface {
	if DefaultCacheService != nil {
		return DefaultCacheService
	}

	return CacheService{}
}

type CacheKeyMap struct {
	DomainName                    string
	DisplayName                   string
	DisplayNameAndEnvironmentName string
	DisplayNameAndDeploymentID    string
}

// DevDomainCacheKey returns the pattern to reset cache for development endpoints.
func DevDomainCacheKey(displayName string) string {
	return fmt.Sprintf(`^%s(?:--\d+)?`, displayName)
}

// Reset sends the signal to subscribers to delete a host.
// Use keys to reset only the given keys.
func (CacheService) Reset(envID types.ID, keys ...string) error {
	ctx := context.Background()
	resetKeys := keys

	// if keys are already pre-provided, no need to fetch them from the db.
	if len(resetKeys) == 0 {
		if envID == 0 {
			return errors.New("invalid environment ID")
		}

		args, err := NewStore().ResetCacheArgs(ctx, envID)

		if err != nil {
			return err
		}

		displayName := ""

		for _, arg := range args {
			// www.necksly.com
			if arg.DomainName != "" {
				resetKeys = append(resetKeys, arg.DomainName)
			}

			// necksly-9iyxzt
			if displayName == "" {
				displayName = arg.DisplayName
				resetKeys = append(resetKeys, DevDomainCacheKey(displayName))
			}
		}
	}

	service := rediscache.Service()

	for _, key := range resetKeys {
		if key == "" {
			continue
		}

		slog.Debug(slog.LogOpts{
			Msg:   fmt.Sprintf("invalidating cache: %s", key),
			Level: slog.DL2,
		})

		if err := service.Broadcast(rediscache.EventInvalidateHostingCache, strings.ToLower(key)); err != nil {
			slog.Errorf("error while broadcasting cache invalidation for key %s: %v", key, err)
		}
	}

	// This might happen when an app is being updated and
	// we reset the display name only
	if envID == 0 {
		return nil
	}

	env, err := buildconf.NewStore().EnvironmentByID(ctx, envID)

	if err != nil {
		slog.Errorf("failed to fetch environment for cache purge webhooks: %v", err)
		return err
	}

	if env == nil {
		slog.Errorf("failed to fetch environment for cache purge webhooks: environment %s not found", envID)
		return errors.New("environment not found")
	}

	whs := app.NewStore().OutboundWebhooks(ctx, env.AppID)

	for _, wh := range whs {
		if wh.TriggerOnCachePurge() {
			wh.Dispatch(app.OutboundWebhookSettings{
				AppID:           env.AppID,
				EnvironmentName: env.Name,
			})
		}
	}

	return nil
}
