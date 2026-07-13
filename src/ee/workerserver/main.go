package main

import (
	"context"
	"log"
	"os"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/tracking"
	"github.com/stormkit-io/stormkit-io/src/migrations"
)

const (
	CRON_HOURLY       = "0 * * * *"
	CRON_EVERY_2_HOUR = "0 */2 * * *"
	CRON_MONTHLY      = "0 0 1 * *"
	CRON_EVERY_MINUTE = "* * * * *"
	CRON_DAILY        = "0 1 * * *"  // Everyday at 01:00
	EVERY_SUNDAY      = "0 18 * * 0" // Every sunday 18:00
)

func main() {
	os.Setenv("STORMKIT_SERVICE_NAME", rediscache.ServiceWorkerserver)

	conf := config.Get()

	if conn := database.Connection(); conn != nil {
		migrations.Up(conn, database.Config)
		go admin.InstallDependencies(context.Background())
	}

	slog.Infof("deployer service: %s", conf.Deployer.Service)

	if config.IsDevelopment() || config.IsSelfHosted() {
		slog.Infof("local storage: %s", conf.Deployer.StorageDir)
		slog.Infof("local deployer: %s", conf.Deployer.Executable)
	}

	if conf.Tracking.Prometheus {
		tracking.Prometheus(tracking.PrometheusOpts{})
	}

	srv, mux := jobs.Server()

	if err := srv.Run(mux); err != nil {
		log.Fatalf("could not run worker server: %v", err)
	}
}
