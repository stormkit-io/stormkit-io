package sysstats

import (
	"context"
	"encoding/json"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"go.uber.org/zap"
)

const (
	// processStatsKey is the per-service key suffix. rediscache expands it to
	// service:procstats:<service>:<instance-id>.
	processStatsKey = "procstats"

	// processStatsTTL outlives a couple of missed reports, so a brief hiccup
	// does not blank the panel, while a stopped instance drops off quickly.
	processStatsTTL = 3 * time.Minute

	processReportInterval = time.Minute
)

// StartProcessReporter publishes this instance's own resource usage on an
// interval.
//
// Machine stats are scraped by the leader, but process stats are not visible
// from another machine — each instance has to report its own. It stops when the
// context is cancelled.
func StartProcessReporter(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(processReportInterval)
		defer ticker.Stop()

		publishProcessStats(ctx)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				publishProcessStats(ctx)
			}
		}
	}()
}

func publishProcessStats(ctx context.Context) {
	client := rediscache.Client()

	if client == nil {
		return
	}

	// Identity is deliberately left off the payload: the reader gets the key
	// from the registry entry, which already carries the service name and
	// instance ID, so writing them again would only risk disagreement.
	payload, err := json.Marshal(CollectProcess(CollectProcessParams{}))

	if err != nil {
		return
	}

	key := rediscache.Service().Key(processStatsKey)

	if err := client.Set(ctx, key, payload, processStatsTTL).Err(); err != nil {
		slog.Debug(slog.LogOpts{
			Msg:     "error while publishing process stats",
			Level:   slog.DL2,
			Payload: []zap.Field{zap.Error(err)},
		})
	}
}

// ReadProcessStats returns the latest report from every live instance, keyed by
// the host it runs on so the UI can group them with that machine.
func ReadProcessStats() map[string][]ProcessStats {
	out := map[string][]ProcessStats{}
	client := rediscache.Client()

	if client == nil {
		return out
	}

	services, err := rediscache.Service().List(nil)

	if err != nil {
		return out
	}

	for _, service := range services {
		raw, err := client.Get(context.Background(), service.Key(processStatsKey)).Result()

		if err != nil {
			continue
		}

		var stats ProcessStats

		if err := json.Unmarshal([]byte(raw), &stats); err != nil {
			continue
		}

		stats.Service = service.Name
		stats.InstanceID = service.ID

		out[service.Host] = append(out[service.Host], stats)
	}

	return out
}
