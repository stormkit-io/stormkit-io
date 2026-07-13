package hosting

import (
	"strings"

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
		return shttp.Proxy(req.RequestContext, shttp.ProxyArgs{Target: match.Redirect}), nil
	}

	if match.Rewrite != "" {
		pieces := strings.Split(match.Rewrite, "?")
		url.Path = pieces[0]

		if len(pieces) > 1 {
			url.RawQuery = pieces[1]
		}

		return nil, nil
	}

	return &shttp.Response{
		Redirect: &match.Redirect,
		Status:   match.Status,
	}, nil
}
