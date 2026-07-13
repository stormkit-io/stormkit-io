package jobs

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var CurrentMinute = func() int { return time.Now().Minute() }

// PingDomains pings domains based on the configuration settings and updates the last ping timestamp.
// It retrieves the configuration settings from the admin store and determines the ping interval and concurrency.
// Then, it fetches the domains from the domain store based on the modulo of the current minute and the interval.
// For each domain, it sends a HEAD request with exponential backoff and records the ping result.
// Finally, it updates the last ping timestamp for the domains in the domain store.
// Returns an error if any occurred during the execution of the function, otherwise nil.
func PingDomains(ctx context.Context) error {
	cfg, err := admin.Store().Config(ctx)

	if err != nil {
		return err
	}

	mod := 5
	workers := 10

	// Minimum is 5 minutes
	if cfg.WorkerserverConfig != nil && cfg.WorkerserverConfig.DomainPingInterval > 5 {
		mod = cfg.WorkerserverConfig.DomainPingInterval
	}

	if cfg.WorkerserverConfig != nil && cfg.WorkerserverConfig.DomainPingConcurrency > 0 {
		workers = cfg.WorkerserverConfig.DomainPingConcurrency
	}

	modID := CurrentMinute() % mod

	domains, err := buildconf.DomainStore().Domains(ctx, buildconf.DomainFilters{
		ModID:       &modID,
		ModInterval: mod,
		Verified:    aws.Bool(true),
	})

	if err != nil {
		return err
	}

	domainsLen := len(domains)

	if domainsLen == 0 {
		return nil
	}

	var wg sync.WaitGroup
	rchan := make(chan buildconf.PingResult, domainsLen)
	dchan := make(chan buildconf.DomainModel, domainsLen)

	headers := make(http.Header)
	headers.Set("User-Agent", "StormkitBot/1.0 (+https://www.stormkit.io)")

	for i := 0; i < workers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for domain := range dchan {
				ping := func() (*shttp.HTTPResponse, error) {
					return shttp.
						NewRequestV2(shttp.MethodHead, fmt.Sprintf("https://%s", domain.Name)).
						Headers(headers).
						Do()
				}

				res, err := ping()

				// If the response is nil, or error is not nil, or response code is not 2xx then retry
				// after 3 seconds.
				if err != nil || !isSuccess(res) {
					time.Sleep(3 * time.Second)
					res, err = ping()
				}

				pr := buildconf.PingResult{
					DomainID:   domain.ID,
					LastPingAt: utils.NewUnix(),
				}

				if err != nil {
					errStr := err.Error()
					pr.Error = errStr

					if strings.Contains(errStr, "no such host") {
						pr.Status = http.StatusNotFound
					} else {
						pr.Status = http.StatusInternalServerError
					}
				}

				if res != nil {
					pr.Status = res.StatusCode
				}

				rchan <- pr
			}
		}()
	}

	// Send work
	for _, domain := range domains {
		dchan <- *domain
	}

	close(dchan)

	// Wait for completion
	wg.Wait()
	close(rchan)

	pingResults := []buildconf.PingResult{}

	for result := range rchan {
		pingResults = append(pingResults, result)
	}

	if err := buildconf.DomainStore().UpdateLastPing(ctx, pingResults); err != nil {
		return err
	}

	return nil
}

func isSuccess(res *shttp.HTTPResponse) bool {
	if res == nil {
		return false
	}

	status := fmt.Sprintf("%d", res.StatusCode)

	if len(status) < 1 {
		return false
	}

	return strings.EqualFold(string(status[0]), "2")
}
