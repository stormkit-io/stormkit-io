package provisioners

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Readiness polls a new instance until it accepts its owner API key.
type Readiness struct {
	Client       *http.Client
	PollInterval time.Duration
	Timeout      time.Duration

	// APIURL returns the base URL of an instance's API for its domain.
	APIURL func(domain string) string
}

// NewReadiness returns a Readiness that gives up after timeout.
func NewReadiness(timeout time.Duration) *Readiness {
	return &Readiness{
		Client:       &http.Client{Timeout: 15 * time.Second},
		PollInterval: 10 * time.Second,
		Timeout:      timeout,
		APIURL: func(domain string) string {
			return "https://api." + domain
		},
	}
}

// Wait polls the instance's MCP endpoint with its API key until it answers
// 2xx. Everything else is retried: right after boot the certificate may not
// be issued yet, the containers may still be pulling, and bootstrap may not
// have stored the key yet. Returns an error wrapping ErrNotReady on timeout,
// or the parent context's error if the caller gave up first.
func (r *Readiness) Wait(parent context.Context, instance *Instance) error {
	ctx, cancel := context.WithTimeout(parent, r.Timeout)
	defer cancel()

	url := strings.TrimRight(r.APIURL(instance.Domain), "/") + "/v1/mcp"
	last := "no response"

	for {
		status, err := r.probe(ctx, probeParams{url: url, apiKey: instance.APIKey})

		if err == nil && status >= 200 && status < 300 {
			return nil
		}

		// A probe cut short by our own deadline says nothing about the
		// instance, so keep the previous answer for the error message.
		if err != nil && ctx.Err() == nil {
			last = err.Error()
		} else if err == nil {
			last = fmt.Sprintf("HTTP %d", status)
		}

		select {
		case <-ctx.Done():
			if err := parent.Err(); err != nil {
				return fmt.Errorf("wait for %s: %w", url, err)
			}

			return fmt.Errorf("%w: %s did not accept the api key after %s (last: %s)", ErrNotReady, url, r.Timeout, last)
		case <-time.After(r.PollInterval):
		}
	}
}

type probeParams struct {
	url    string
	apiKey string
}

func (r *Readiness) probe(ctx context.Context, params probeParams) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, params.url, nil)

	if err != nil {
		return 0, err
	}

	req.Header.Set("Authorization", "Bearer "+params.apiKey)

	res, err := r.Client.Do(req)

	if err != nil {
		return 0, err
	}

	res.Body.Close()

	return res.StatusCode, nil
}
