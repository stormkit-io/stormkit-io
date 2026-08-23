package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"strings"

	proxyproto "github.com/pires/go-proxyproto"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/bootstrap"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/tracking"
	"github.com/stormkit-io/stormkit-io/src/migrations"
)

// maybePProfListener exposes net/http/pprof on its own mux when
// STORMKIT_PPROF_ADDR is set. The mux is dedicated (not DefaultServeMux)
// so the pprof endpoints are reachable only on the configured address —
// typically a loopback or admin-only interface (e.g. 127.0.0.1:6060).
// Leave the env var unset in production unless actively debugging.
func maybePProfListener() {
	addr := os.Getenv("STORMKIT_PPROF_ADDR")

	if addr == "" {
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	go func() {
		slog.Infof("pprof listening on %s", addr)

		if err := http.ListenAndServe(addr, mux); err != nil {
			slog.Errorf("pprof server error: %v", err)
		}
	}()
}

// nonMagic starts an http server.
func nonMagic(handler http.Handler, port string) {
	addr := fmt.Sprintf(":%s", port)
	slog.Info(fmt.Sprintf("external server listening on %s", addr))

	ln, err := net.Listen("tcp", addr)

	if err != nil {
		log.Fatal(err)
	}

	if config.Get().ProxyProtocol {
		ln = &proxyproto.Listener{Listener: ln}
	}

	srv := &http.Server{
		Handler:        handler,
		MaxHeaderBytes: hosting.MaxRequestHeaderBytes,
	}

	log.Fatal(srv.Serve(ln))
}

func handler() http.Handler {
	slog.Debug(slog.LogOpts{
		Msg:   "registering middlewares and services",
		Level: slog.DL3,
	})

	r := shttp.NewRouter()
	r.RegisterMiddleware(hosting.WithTimeout)
	r.RegisterMiddleware(hosting.InternalHandlers(hosting.InternalHandlerOpts{
		App:    !config.IsStormkitCloud(),
		API:    os.Getenv("STORMKIT_API") != "off",
		Health: true,
	}))
	r.RegisterService(hosting.Services)

	return r.WithGzip().Handler()
}

func main() {
	os.Setenv("STORMKIT_SERVICE_NAME", rediscache.ServiceHosting)

	c := config.Get()

	if conn := database.Connection(); conn != nil {
		migrations.Up(conn, database.Config)
		admin.InstallDependencies(context.Background())
		bootstrap.FromEnv(context.Background())
	}

	// Register redis listeners
	hosting.RegisterListeners()

	maybePProfListener()

	if c.Tracking != nil && c.Tracking.Prometheus {
		tracking.Prometheus(tracking.PrometheusOpts{
			Apdex: true,
		})
	}

	h := handler()

	https := strings.ToLower(os.Getenv("STORMKIT_HTTPS"))
	skipTLS := https == "false" || https == "0" || https == "off"

	if (config.IsDevelopment() && !config.IsStormkitCloud()) || skipTLS {
		nonMagic(h, config.LocalhostPort)
		return
	}

	hosting.Magic(hosting.MagicOpts{
		Handler:      h,
		FetchAppConf: hosting.FetchAppConf,
	})
}
