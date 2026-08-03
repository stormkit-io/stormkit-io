package jobs

import (
	"context"
	"sync"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/sysstats"
	"go.uber.org/zap"
)

// scrapeConcurrency bounds how many exporters are contacted at once. Targets
// are machines, so this stays small — the work is waiting on the network, not
// computing.
const scrapeConcurrency = 8

// sharedCollector is package-level because it retains the previous CPU counters
// per target. A collector built per run would never produce a CPU reading, as
// every scrape would look like a first scrape.
var sharedCollector = sysstats.NewCollector(sysstats.NewCollectorParams{})

// CollectSystemStats scrapes every known node_exporter and records the results,
// then snapshots Postgres and Redis health.
//
// It is a master task: the targets are shared, so running it on every replica
// would multiply the scrape load and write the same samples repeatedly.
func CollectSystemStats(ctx context.Context) error {
	store := sysstats.NewStore(sysstats.NewStoreParams{})
	targets := resolveTargets(ctx)

	if len(targets) == 0 {
		slog.Debug(slog.LogOpts{
			Msg:   "no monitoring targets to scrape",
			Level: slog.DL2,
		})
	}

	scrapeTargets(ctx, store, targets)

	health := sysstats.CollectDependencies(ctx, sysstats.CollectDependenciesParams{
		DB:    database.Connection(),
		Cache: rediscache.Client(),
	})

	if err := store.SaveDependencies(ctx, health); err != nil {
		slog.Errorf("error while saving dependency health: %v", err)
	}

	return nil
}

// resolveTargets merges machines that registered themselves with any configured
// manually.
func resolveTargets(ctx context.Context) []sysstats.Target {
	var registered []sysstats.RegisteredService

	services, err := rediscache.Service().List(nil)

	if err != nil {
		slog.Errorf("error while listing services for monitoring: %v", err)
	}

	for _, service := range services {
		registered = append(registered, sysstats.RegisteredService{
			Name: service.Name,
			Host: service.Host,
		})
	}

	var manual []string

	if cfg, err := admin.Store().Config(ctx); err == nil && cfg.MonitoringConfig != nil {
		manual = cfg.MonitoringConfig.Targets
	}

	return sysstats.ResolveTargets(sysstats.ResolveTargetsParams{
		Registered: registered,
		Manual:     manual,
	})
}

func scrapeTargets(ctx context.Context, store *sysstats.Store, targets []sysstats.Target) {
	sem := make(chan struct{}, scrapeConcurrency)
	wg := sync.WaitGroup{}

	for _, target := range targets {
		wg.Add(1)

		go func(host string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			// An unreachable machine still yields a sample. Recording it is what
			// lets the UI say node_exporter is not running there, which is the
			// most likely thing to be wrong.
			sample := sharedCollector.Collect(ctx, host)

			if !sample.Reachable {
				slog.Debug(slog.LogOpts{
					Msg:   "monitoring target unreachable",
					Level: slog.DL2,
					Payload: []zap.Field{
						zap.String("target", host),
						zap.String("error", sample.Error),
					},
				})
			}

			if err := store.Append(ctx, sample); err != nil {
				slog.Errorf("error while storing sample for %s: %v", host, err)
			}
		}(target.Host)
	}

	wg.Wait()
}
