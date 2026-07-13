package hosting

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/caddyserver/certmagic"
	"github.com/google/uuid"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// RequestContext is the extended shttp request context.
type RequestContext struct {
	*shttp.RequestContext

	OriginalPath string

	// Host is the associated host configuration for this request.
	Host *Host

	// Whether debug mode is on or not.
	// The debug mode will be turned on by accessing canary environment.
	// Debugging will post serialized information on the request/response
	// to Cloudwatch.
	Debug bool

	Fields []zap.Field

	RequestID string
}

var cachedCertMagicServer *certmagic.Config
var cachedCertMagicMu sync.Mutex

// CertMagic returns the cached certmagic configuration.
func CertMagic() *certmagic.Config {
	if cachedCertMagicServer == nil {
		zcnf := zap.NewProductionConfig()
		zcnf.Level = zap.NewAtomicLevelAt(zapcore.FatalLevel + 1) // Disable all logs
		logger, _ := zcnf.Build()

		slog.Info("certmagic server configuration is empty, generating a new one using NewDefault")

		cachedCertMagicServer = certmagic.NewDefault()
		cachedCertMagicServer.Logger = logger
	}

	return cachedCertMagicServer
}

type DecisionFuncOpts struct {
	Server       *certmagic.Config
	FetchAppConf func(hostName string) ([]*appconf.Config, error)
}

// DecisionFunc decides whether to issue a certificate or not.
func DecisionFunc(opts DecisionFuncOpts) func(context.Context, string) error {
	cachedCertMagicMu.Lock()
	defer cachedCertMagicMu.Unlock()

	cachedCertMagicServer = opts.Server

	return func(ctx context.Context, name string) error {
		cnf := admin.MustConfig()

		// Allow using Stormkit as a proxy server for configured domains.
		if cnf.ProxyConfig != nil && cnf.ProxyConfig.Rules != nil {
			if rule := cnf.ProxyConfig.Rules[name]; rule != nil {
				return nil
			}
		}

		if name == "localhost" {
			return fmt.Errorf("localhost is not a valid host name")
		}

		if getHandlerFromHost(name, InternalHandlerOpts{API: true, Health: true, App: !config.IsStormkitCloud()}) != nil {
			return nil
		}

		if addr := net.ParseIP(name); addr != nil {
			return fmt.Errorf("invalid domain name: %v", name)
		}

		confs, err := opts.FetchAppConf(name)

		if err != nil {
			slog.Errorf("decision func get configs err: %v", err)
			return err
		}

		if len(confs) == 0 {
			return fmt.Errorf("domain %s is not allowed for requesting a certificate", name)
		}

		if confs[0].CertKey != "" && confs[0].CertValue != "" {
			return fmt.Errorf("custom certificate provided - aborting automatic process")
		}

		if len(confs) > 0 && confs[0].DeploymentID == 0 {
			return fmt.Errorf("deployment not found")
		}

		slog.Debug(slog.LogOpts{
			Msg:   fmt.Sprintf("requesting certificate for: %s", name),
			Level: slog.DL3,
		})

		return nil
	}
}

// WithHost adds the host that is currently requested to the context.
func WithHost(handler func(*RequestContext) *shttp.Response) shttp.RequestFunc {
	isCloud := config.IsStormkitCloud()

	return func(req *shttp.RequestContext) *shttp.Response {
		requestID := uuid.New().String()
		absURL := req.URL().String()
		url := req.URL()

		slog.Debug(slog.LogOpts{
			Msg:   "incoming request",
			Level: slog.DL4,
			Payload: []zap.Field{
				zap.String("method", req.Method),
				zap.String("path", url.Path),
				zap.String("query", url.RawQuery),
				zap.String("host", req.Host),
				zap.String("remote_addr", req.RemoteAddr()),
				zap.String("user_agent", req.Header.Get("User-Agent")),
				zap.String("origin", req.Header.Get("Origin")),
				zap.String("request_id", requestID),
			},
		})

		if isCloud && appconf.IsStormkitDevStrict(req.Host) {
			return &shttp.Response{
				Redirect: utils.Ptr("https://www.stormkit.io"),
				Status:   http.StatusTemporaryRedirect,
			}
		}

		cnf, err := admin.Store().Config(req.Context())

		if err != nil {
			slog.Errorf("error getting admin config: %v", err)
		}

		// Handle proxy requests.
		if cnf.ProxyConfig != nil && cnf.ProxyConfig.Rules != nil {
			if rule := cnf.ProxyConfig.Rules[req.Host]; rule != nil {
				proxiedURL := strings.TrimPrefix(strings.TrimPrefix(absURL, "https://"), "http://")
				proxiedURL = strings.Replace(proxiedURL, req.Host, rule.Target, 1)

				if rule.Headers != nil {
					for k, v := range rule.Headers {
						req.Header.Set(k, v)
					}
				}

				return shttp.Proxy(req, shttp.ProxyArgs{Target: proxiedURL})
			}
		}

		host := hostFromContext(req)

		slog.Debug(slog.LogOpts{
			Msg:   "host fetched",
			Level: slog.DL4,
			Payload: []zap.Field{
				zap.Bool("has_host", host != nil),
				zap.String("request_id", requestID),
			},
		})

		if host == nil {
			return shttp.NotFound()
		}

		fields := []zap.Field{
			zap.String("host", host.Name),
			zap.String("request_id", requestID),
		}

		return handler(&RequestContext{
			Host:           host,
			RequestContext: req,
			OriginalPath:   fmt.Sprintf("/%s", strings.TrimLeft(req.URL().Path, "/")),
			Fields:         fields,
			RequestID:      requestID,
		})
	}
}

// hostFromContext gets the host from the context.
func hostFromContext(req *shttp.RequestContext) *Host {
	host := &Host{
		Request: req,
	}

	domain := utils.GetString(req.HostName(), req.Host)

	host.IsStormkitSubdomain = appconf.IsStormkitDev(domain)
	host.Name = domain

	if err := host.RequestConfig(); err != nil {
		return nil
	}

	return host
}
