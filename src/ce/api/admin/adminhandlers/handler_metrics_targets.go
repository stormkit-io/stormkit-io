package adminhandlers

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/sysstats"
)

type metricsTargetsRequest struct {
	Targets []string `json:"targets"`
}

func handlerMetricsTargets(req *user.RequestContext) *shttp.Response {
	cfg, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	targets := []string{}

	if cfg.MonitoringConfig != nil && cfg.MonitoringConfig.Targets != nil {
		targets = cfg.MonitoringConfig.Targets
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"targets": targets,
		},
	}
}

func handlerMetricsTargetsUpdate(req *user.RequestContext) *shttp.Response {
	data := metricsTargetsRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	targets, err := normalizeTargets(data.Targets)

	if err != nil {
		return shttp.BadRequest(map[string]any{
			"error": err.Error(),
		})
	}

	ctx := req.Context()
	cfg, err := admin.Store().Config(ctx)

	if err != nil {
		return shttp.Error(err)
	}

	removed := removedTargets(cfg, targets)

	if cfg.MonitoringConfig == nil {
		cfg.MonitoringConfig = &admin.MonitoringConfig{}
	}

	cfg.MonitoringConfig.Targets = targets

	if err := admin.Store().UpsertConfig(ctx, cfg); err != nil {
		return shttp.Error(err)
	}

	// Drop history for machines that are no longer monitored, so they stop
	// appearing immediately instead of lingering until their key expires.
	store := sysstats.NewStore(sysstats.NewStoreParams{})

	for _, target := range removed {
		if err := store.Drop(ctx, target); err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"targets": targets,
		},
	}
}

func removedTargets(cfg admin.InstanceConfig, next []string) []string {
	if cfg.MonitoringConfig == nil {
		return nil
	}

	var removed []string

	for _, target := range cfg.MonitoringConfig.Targets {
		if !slices.Contains(next, target) {
			removed = append(removed, target)
		}
	}

	return removed
}

// normalizeTargets trims, de-duplicates and validates the submitted hosts.
//
// Targets are fetched by the server, so an unchecked value is a request
// forgery primitive. Only http and https are accepted, and a bare host is
// treated as a host rather than a URL.
func normalizeTargets(targets []string) ([]string, error) {
	out := []string{}

	for _, target := range targets {
		target = strings.TrimSpace(target)

		if target == "" {
			continue
		}

		if err := validateTarget(target); err != nil {
			return nil, err
		}

		if !slices.Contains(out, target) {
			out = append(out, target)
		}
	}

	return out, nil
}

func validateTarget(target string) error {
	if strings.Contains(target, "://") {
		parsed, err := url.Parse(target)

		if err != nil {
			return fmt.Errorf("%q is not a valid target", target)
		}

		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			return fmt.Errorf("%q must use http or https", target)
		}

		if parsed.Hostname() == "" {
			return fmt.Errorf("%q is missing a host", target)
		}

		return nil
	}

	host := target

	if strings.Contains(target, ":") {
		var err error
		host, _, err = net.SplitHostPort(target)

		if err != nil {
			return fmt.Errorf("%q is not a valid host:port", target)
		}
	}

	if host == "" || strings.ContainsAny(host, " /?#") {
		return fmt.Errorf("%q is not a valid host", target)
	}

	return nil
}
