package hosting

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/libdns/route53"
	proxyproto "github.com/pires/go-proxyproto"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func storage(logger *zap.Logger) certmagic.Storage {
	slog.Infof("using redis storage for certificates")

	storage := NewRedisStorage(logger)
	storage.SetClient(rediscache.Client().Client)

	return storage
}

type MagicOpts struct {
	Handler      http.Handler
	FetchAppConf func(hostName string) ([]*appconf.Config, error)
}

// magic starts a caddy server to serve requests on HTTPS port.
// All HTTP requests will be redirected to HTTPS.
func Magic(opts MagicOpts) {
	// The log package is being used primarily by net package and creates
	// a lot of noise. So disable it to keep the logs clean.
	log.SetOutput(io.Discard)

	zcnf := zap.NewProductionConfig()

	if os.Getenv("STORMKIT_ENABLE_ACME_LOGS") != "true" {
		zcnf.Level = zap.NewAtomicLevelAt(zapcore.FatalLevel + 1) // Disable all logs
	}

	logger, _ := zcnf.Build()
	defaultCA := utils.GetString(os.Getenv("STORMKIT_ACME_CA"), certmagic.LetsEncryptProductionCA)

	certmagic.HTTPPort = utils.StringToInt(utils.GetString(os.Getenv("STORMKIT_HTTP_PORT"), "80"))
	certmagic.HTTPSPort = utils.StringToInt(utils.GetString(os.Getenv("STORMKIT_HTTPS_PORT"), "443"))
	certmagic.Default.Storage = storage(logger)
	certmagic.Default.Logger = logger
	certmagic.DefaultACME.Agreed = true
	certmagic.DefaultACME.CA = defaultCA
	certmagic.DefaultACME.Email = os.Getenv("STORMKIT_ACME_EMAIL")
	certmagic.DefaultACME.Logger = logger

	managedDomain := strings.Split(admin.MustConfig().DomainConfig.Dev, "//")[1]
	certmagic.Default.DefaultServerName = managedDomain

	// See https://letsencrypt.org/2024/12/05/ending-ocsp
	certmagic.Default.OCSP = certmagic.OCSPConfig{
		DisableStapling: true,
	}

	server := certmagic.NewDefault()
	server.Logger = logger

	// This part is needed only for Stormkit Cloud
	if config.IsStormkitCloud() {
		server.Issuers = []certmagic.Issuer{
			certmagic.NewACMEIssuer(server, certmagic.ACMEIssuer{
				CA:                      defaultCA,
				Email:                   "admin@stormkit.io",
				Agreed:                  true,
				DisableTLSALPNChallenge: true,
				Logger:                  logger,
				DNS01Solver: &certmagic.DNS01Solver{
					DNSManager: certmagic.DNSManager{
						DNSProvider: &route53.Provider{
							AccessKeyId:     os.Getenv("STORMKIT_ACME_ACCESS_KEY"),
							SecretAccessKey: os.Getenv("STORMKIT_ACME_SECRET_KEY"),
							HostedZoneID:    os.Getenv("STORMKIT_DEV_ZONE_ID"),
						},
					},
				},
			}),
		}

		managed := []string{fmt.Sprintf("*.%s", certmagic.Default.DefaultServerName)}

		if err := server.ManageAsync(context.Background(), managed); err != nil {
			slog.Errorf("error while managing async certificates: %v", err)
		} else {
			slog.Infof("managing certificates for domains: %s", strings.Join(managed, ", "))
		}
	}

	// This has to be enabled now, after ManageAsync is called
	// and has to be registered at the default level because
	// it will be used in certmagic.NewDefault() call once certmagic.HTTPS is called.
	certmagic.Default.OnDemand = &certmagic.OnDemandConfig{
		DecisionFunc: DecisionFunc(DecisionFuncOpts{
			Server:       server,
			FetchAppConf: opts.FetchAppConf,
		}),
	}

	slog.Infof("external server listening on :%d (https) and :%d (http)", certmagic.HTTPSPort, certmagic.HTTPPort)

	if err := httpsServe(certmagic.NewDefault(), opts.Handler); err != nil {
		slog.Errorf("encountered following error while launching https server: %s", err.Error())
	}
}

// httpsServe is a drop-in replacement for certmagic.HTTPS that removes the
// hardcoded ReadTimeout / WriteTimeout from the HTTPS server. Timeouts for
// normal requests are enforced at the application layer via WithTimeout; the
// server-level timeout is left at zero so that large file uploads are never
// cut short by the underlying http.Server.
func httpsServe(cfg *certmagic.Config, handler http.Handler) error {
	ctx := context.Background()

	httpTCPLn, err := net.Listen("tcp", fmt.Sprintf(":%d", certmagic.HTTPPort))

	if err != nil {
		return err
	}

	httpLn := maybeProxyProto(httpTCPLn)

	tlsConfig := cfg.TLSConfig()
	tlsConfig.NextProtos = append([]string{"h2", "http/1.1"}, tlsConfig.NextProtos...)

	httpsTCPLn, err := net.Listen("tcp", fmt.Sprintf(":%d", certmagic.HTTPSPort))

	if err != nil {
		httpLn.Close()
		return err
	}

	httpsLn := tls.NewListener(maybeProxyProto(httpsTCPLn), tlsConfig)

	// Ensure both listeners are closed when the HTTPS server stops, which
	// also causes the HTTP goroutine below to exit cleanly.
	defer func() {
		httpLn.Close()
		httpsLn.Close()
	}()

	// The HTTP server only handles ACME challenges and redirects; tight
	// timeouts are fine here.
	httpServer := &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       5 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	if len(cfg.Issuers) > 0 {
		if am, ok := cfg.Issuers[0].(*certmagic.ACMEIssuer); ok {
			httpServer.Handler = am.HTTPChallengeHandler(http.HandlerFunc(httpRedirectHandler))
		}
	}

	// No ReadTimeout / WriteTimeout — the application-level WithTimeout
	// middleware handles ordinary request timeouts, and large uploads must
	// not be interrupted by a server-level deadline.
	httpsServer := &http.Server{
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       config.Get().HTTPTimeouts.IdleTimeout,
		Handler:           handler,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	go httpServer.Serve(httpLn)

	return httpsServer.Serve(httpsLn)
}

// maybeProxyProto wraps ln with a PROXY protocol listener when
// STORMKIT_PROXY_PROTOCOL is enabled, otherwise returns ln unchanged.
func maybeProxyProto(ln net.Listener) net.Listener {
	if !config.Get().ProxyProtocol {
		return ln
	}

	return &proxyproto.Listener{Listener: ln}
}

// httpRedirectHandler redirects plain HTTP requests to HTTPS.
func httpRedirectHandler(w http.ResponseWriter, r *http.Request) {
	host := r.Host

	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}

	w.Header().Set("Connection", "close")
	http.Redirect(w, r, "https://"+host+r.URL.RequestURI(), http.StatusMovedPermanently)
}
