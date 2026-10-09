package hosting

import (
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"go.uber.org/zap"
)

func WithRedirect(req *RequestContext) (*shttp.Response, error) {
	conf := req.Host.Config

	if conf == nil || len(conf.Redirects) == 0 {
		return nil, nil
	}

	fields := []zap.Field{
		zap.String("host", req.Host.Name),
		zap.String("request_id", req.RequestID),
		zap.String("api_location", req.Host.Config.APILocation),
		zap.String("api_path_prefix", req.Host.Config.APIPathPrefix),
		zap.String("url", req.URL().String()),
	}

	slog.Debug(slog.LogOpts{
		Msg:     "running redirect middleware",
		Level:   slog.DL4,
		Payload: fields,
	})

	url := req.URL()
	match := redirects.Match(redirects.MatchArgs{
		URL:           url,
		HostName:      req.Host.Name,
		APIPathPrefix: req.Host.Config.APIPathPrefix,
		APILocation:   req.Host.Config.APILocation,
		Redirects:     conf.Redirects,
	})

	if match == nil {
		return nil, nil
	}

	if match.Proxy {
		return shttp.Proxy(req.RequestContext, shttp.ProxyArgs{
			Target:   match.Redirect,
			DialAddr: localDialAddr(match.Redirect),
		}), nil
	}

	if match.Rewrite != "" {
		pieces := strings.Split(match.Rewrite, "?")
		url.Path = pieces[0]

		if len(pieces) > 1 {
			url.RawQuery = pieces[1]
		}

		// The loader is resolved here, against the path the visitor asked for,
		// and read later against the document the rewrite landed on.
		req.Loader = match.Data

		return nil, nil
	}

	return &shttp.Response{
		Redirect: &match.Redirect,
		Status:   match.Status,
	}, nil
}

// localProxyAddr is where this process accepts HTTPS connections, set when the
// server starts. Empty disables local dialling.
var localProxyAddr string

// localDialAddr returns localProxyAddr when target is an HTTPS URL on one of
// this instance's dev domains, so the proxied request is handled on this
// machine. Resolving such a domain through DNS sends the request out to the
// load balancer and back in, which costs public bandwidth and can hang when the
// balancer preserves client IPs and routes the request back to this node.
//
// Custom domains are left alone: they can be pointed somewhere else while still
// registered here, and only DNS knows where they live now.
func localDialAddr(target string) string {
	if localProxyAddr == "" {
		return ""
	}

	u, err := url.Parse(target)

	if err != nil || u.Scheme != "https" || (u.Port() != "" && u.Port() != "443") {
		return ""
	}

	if !appconf.IsStormkitDev(u.Hostname()) {
		return ""
	}

	confs, err := FetchAppConf(u.Hostname())

	// A custom domain can sit under the dev domain; DomainID tells them apart.
	if err != nil || len(confs) == 0 || confs[0].DeploymentID == 0 || confs[0].DomainID != 0 {
		return ""
	}

	return localProxyAddr
}
