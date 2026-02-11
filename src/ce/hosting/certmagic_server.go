package hosting

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/caddyserver/certmagic"
	"github.com/libdns/route53"
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

	if err := certmagic.HTTPS(nil, opts.Handler); err != nil {
		fmt.Printf("encountered following error while launching https server: %s", err.Error())
	}
}
