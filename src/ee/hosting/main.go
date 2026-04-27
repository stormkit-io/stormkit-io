package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"

	proxyproto "github.com/pires/go-proxyproto"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/tracking"
	"github.com/stormkit-io/stormkit-io/src/migrations"
)

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

	srv := &http.Server{Handler: handler}

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

		if config.IsStormkitCloud() {
			migrations.DecryptLicenseKeys(conn)
		}

		admin.InstallDependencies(context.Background())
	}

	// Register redis listeners
	hosting.RegisterListeners()

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
