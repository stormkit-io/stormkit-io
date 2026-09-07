package hosting

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shutdown"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// WarmupHeader marks a request this node made to boot a deployment, so an
// application can tell one from a visit.
const WarmupHeader = "X-Stormkit-Warmup"

// warmupInterval is how often the deployment is asked again while it boots.
const warmupInterval = time.Second

// warmupRequest and warmupDiscard are swapped out by this package's tests,
// which have no deployment to serve and no process to stop.
var (
	warmupRequest = doWarmupRequest
	warmupDiscard = doWarmupDiscard
	warmupResolve = warmupConfig
)

// HandlePublishWarmup answers a warm-up broadcast for a deployment that is
// about to be published.
//
// It returns immediately: SubscribeAsync dispatches messages sequentially in a
// single goroutine per event, so probing inline would stall every later
// warm-up on this node.
func HandlePublishWarmup(ctx context.Context, payload ...string) {
	req := deploy.WarmupRequest{}

	if len(payload) == 0 || json.Unmarshal([]byte(payload[0]), &req) != nil {
		slog.Errorf("cannot decode publish warm-up request")
		return
	}

	go warmer{}.run(warmupContext(), req)
}

// warmer boots a deployment on this node and reports whether it serves.
type warmer struct{}

// run probes the deployment and records this node's verdict.
func (w warmer) run(ctx context.Context, req deploy.WarmupRequest) {
	serviceID := rediscache.Service().ServiceID()

	// A node that came up while Redis was down has no identity, so anything it
	// wrote would land on a key the publish does not read. Staying silent is
	// the honest outcome: it counts as pending and the publish fails on its
	// deadline rather than moving traffic to something nobody verified.
	if serviceID == "" {
		slog.Errorf("cannot report publish warm-up %s: this service has no id", req.WarmupID)
		return
	}

	result := w.probe(ctx, req)
	result.ServiceID = serviceID
	result.ServiceName = rediscache.ServiceHosting
	result.DeploymentID = req.DeploymentID

	err := deploy.SaveNodeResult(ctx, deploy.SaveNodeResultParams{
		WarmupID: req.WarmupID,
		Deadline: req.Deadline,
		Result:   result,
	})

	if err != nil {
		slog.Errorf("cannot record publish warm-up result for %s: %s", req.WarmupID, err.Error())
	}
}

// probe asks this node for the deployment until it stops answering "not yet".
//
// The request goes through the same pipeline a visitor's would, so whatever
// makes a deployment serve — static files, a server process, a managed function
// — is exercised without this code having to know which it is. A deployment
// with nothing to boot answers on the first attempt.
//
// The host is the deployment's own endpoint, which is the only name that
// resolves to a deployment that is not published yet.
func (w warmer) probe(ctx context.Context, req deploy.WarmupRequest) deploy.WarmupResult {
	host := warmupHost(req)

	// An instance with no preview domain has no name that addresses an
	// unpublished deployment. Nothing can be warmed, so let the publish through
	// rather than blocking every release on it.
	if host == "" {
		return deploy.WarmupResult{
			Status: rediscache.StatusOK,
			Reason: "this instance has no endpoint that addresses an unpublished deployment",
		}
	}

	// Resolving the config up front is what makes a 404 meaningful: without it
	// a node with a stale cache would answer "not found" to its own request and
	// report the deployment ready without ever booting it.
	cnf, err := warmupResolve(host, req.DeploymentID)

	if err != nil {
		return deploy.WarmupResult{Status: rediscache.StatusErr, Reason: err.Error()}
	}

	deadline := req.DeadlineAt()
	started := time.Now()

	for {
		status := warmupRequest(host)

		// A deployment that answers for itself has come up. A 5xx has not: a
		// server that fails to spawn surfaces as a 500 from the edge and one
		// that is still booting as a 503, so neither may pass the gate.
		if status > 0 && status < http.StatusInternalServerError {
			return deploy.WarmupResult{Status: rediscache.StatusOK, StatusCode: status}
		}

		if ctx.Err() != nil || !time.Now().Before(deadline) {
			warmupDiscard(cnf)

			return deploy.WarmupResult{
				Status:     rediscache.StatusErr,
				StatusCode: status,
				Reason: fmt.Sprintf(
					"the deployment did not serve a request within %s",
					time.Since(started).Round(time.Second),
				),
			}
		}

		select {
		case <-time.After(warmupInterval):
		case <-ctx.Done():
		}
	}
}

// warmupHost is the deployment's own endpoint, without the scheme.
func warmupHost(req deploy.WarmupRequest) string {
	endpoint := admin.MustConfig().PreviewURL(req.DisplayName, req.DeploymentID.String())

	return strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://")
}

// doWarmupRequest runs one HEAD through this node's own request pipeline and
// returns the status it produced.
func doWarmupRequest(host string) int {
	r := &http.Request{
		Method:     http.MethodHead,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Host:       host,
		URL:        &url.URL{Scheme: "https", Host: host, Path: "/"},
		Header:     http.Header{WarmupHeader: []string{"1"}, "User-Agent": []string{"Stormkit-Warmup/1"}},
		Body:       http.NoBody,
		RemoteAddr: "127.0.0.1:0",
	}

	if res := WithHost(HandlerForward)(shttp.NewRequestContext(r)); res != nil {
		return res.Status
	}

	return 0
}

// warmupConfig returns the config this node will serve the deployment from.
func warmupConfig(host string, deploymentID types.ID) (*appconf.Config, error) {
	confs, err := FetchAppConf(host)

	if err != nil {
		return nil, fmt.Errorf("cannot read the configuration of deployment %s: %s", deploymentID.String(), err.Error())
	}

	for _, cnf := range confs {
		if cnf.DeploymentID == deploymentID {
			return cnf, nil
		}
	}

	return nil, fmt.Errorf("no configuration found for deployment %s", deploymentID.String())
}

// doWarmupDiscard stops the server this node started for a deployment that
// never came up, so repeated pushes of an application that cannot boot do not
// stack up one process per attempt until the ten-minute idle timer.
//
// A deployment that is already serving is left alone: re-publishing the live
// deployment would otherwise kill the process answering production.
func doWarmupDiscard(cnf *appconf.Config) {
	if cnf == nil || cnf.FunctionLocation == "" || cnf.Percentage > 0 {
		return
	}

	if service := integrations.Filesys().ProcessManager().GetService(cnf.FunctionLocation); service != nil {
		service.Kill()
	}
}

// warmupContext detaches a probe from the broadcast that triggered it, while
// still ending it when the node is shutting down. The broadcast's own context
// is done as soon as the message is dispatched, and a probe outlives that by
// design.
func warmupContext() context.Context {
	warmupCtxOnce.Do(func() {
		var cancel context.CancelFunc

		warmupCtx, cancel = context.WithCancel(context.Background())

		shutdown.Subscribe(func() error {
			cancel()
			return nil
		})
	})

	return warmupCtx
}

var (
	warmupCtxOnce sync.Once
	warmupCtx     context.Context
)
