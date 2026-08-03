package adminhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/sysstats"
)

// machineResponse is one monitored machine and its latest reading. Sample is
// nil until the scraper has run at least once, which is the normal state in the
// first minute after boot.
type machineResponse struct {
	Host     string           `json:"host"`
	Services []string         `json:"services"`
	Manual   bool             `json:"manual"`
	Sample   *sysstats.Sample `json:"sample"`
}

func handlerMetrics(req *user.RequestContext) *shttp.Response {
	ctx := req.Context()
	store := sysstats.NewStore(sysstats.NewStoreParams{})

	targets, err := monitoringTargets(req)

	if err != nil {
		return shttp.Error(err)
	}

	machines := make([]machineResponse, 0, len(targets))

	for _, target := range targets {
		sample, err := store.Latest(ctx, target.Host)

		if err != nil {
			return shttp.Error(err)
		}

		machines = append(machines, machineResponse{
			Host:     target.Host,
			Services: target.Services,
			Manual:   target.Manual,
			Sample:   sample,
		})
	}

	dependencies, err := store.ReadDependencies(ctx)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"machines":       machines,
			"dependencies":   dependencies,
			"pool":           sysstats.NewPoolStats(database.Connection()),
			"retentionHours": int(sysstats.Retention.Hours()),
		},
	}
}

// monitoringTargets merges the machines that registered themselves with any
// configured by hand, mirroring what the scraper does.
func monitoringTargets(req *user.RequestContext) ([]sysstats.Target, error) {
	var registered []sysstats.RegisteredService

	services, err := rediscache.Service().List(nil)

	if err != nil {
		return nil, err
	}

	for _, service := range services {
		registered = append(registered, sysstats.RegisteredService{
			Name: service.Name,
			Host: service.Host,
		})
	}

	var manual []string

	cfg, err := admin.Store().Config(req.Context())

	if err != nil {
		return nil, err
	}

	if cfg.MonitoringConfig != nil {
		manual = cfg.MonitoringConfig.Targets
	}

	return sysstats.ResolveTargets(sysstats.ResolveTargetsParams{
		Registered: registered,
		Manual:     manual,
	}), nil
}
