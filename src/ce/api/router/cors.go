package router

import (
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
)

var hostsMux sync.Mutex

var AllowedHosts = []string{}

var AllowedHeaders = []string{
	"Authorization",
	"Connection",
	"Content-Type",
	"Cache-Control",
	"Access-Control-Allow-Methods",
	"Access-Control-Allow-Origin",
	"Access-Control-Allow-Headers",
	"Access-Control-Max-Age",
	"Access-Control-Request-Headers",
	"Access-Control-Request-Method",
	"X-File-Id",
	"X-Total-Chunks",
	"X-Chunk-Index",
	"X-Chunked-Upload",
}

var AllowedMethods = []string{
	"POST",
	"GET",
	"OPTIONS",
	"DELETE",
	"PUT",
	"PATCH",
}

func Cors() []string {
	hostsMux.Lock()
	defer hostsMux.Unlock()

	ep := admin.MustConfig().DomainConfig

	if config.IsSelfHosted() {
		AllowedHosts = append(AllowedHosts,
			"^"+strings.ReplaceAll(ep.App, ".", "\\.")+"$",
		)
	}

	if config.IsStormkitCloud() {
		AllowedHosts = append(AllowedHosts,
			"^https?://app(-*[a-zA-Z0-9]+)?.stormkit.io$",
			"^https?://app(-*[a-zA-Z0-9]+)?.stormkit.dev$",
			"^https?://app(-*[a-zA-Z0-9]+)?.stormkit.app$",
		)
	}

	if config.IsDevelopment() {
		AllowedHosts = append(AllowedHosts,
			"^https?://localhost:[0-9]+$",
		)
	}

	return AllowedHosts
}

// ResetCors clears the allowed hosts and re-applies the cors settings.
func ResetCors() {
	hostsMux.Lock()
	AllowedHosts = []string{}
	hostsMux.Unlock()

	Cors()
}

func WithTimeout(h http.Handler) http.Handler {
	timeout := config.Get().HTTPTimeouts.ReadTimeout
	regular := http.TimeoutHandler(h, timeout, "timeout")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// http.TimeoutHandler buffers the entire response, which is incompatible
		// with SSE streams (long-lived connections). Skip the timeout wrapper only
		// for the known SSE endpoint (GET /v1/mcp) to prevent Accept-header
		// spoofing from bypassing timeouts on arbitrary routes.
		if r.Method == http.MethodGet && r.URL.Path == "/v1/mcp" {
			h.ServeHTTP(w, r)
			return
		}

		regular.ServeHTTP(w, r)
	})
}

// withCors enables cors headers for the api.
func WithCors(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, host := range AllowedHosts {
			origin := r.Header.Get("Origin")

			if match, _ := regexp.MatchString(host, origin); match {
				w.Header().Add("Connection", "keep-alive")
				w.Header().Add("Access-Control-Allow-Origin", origin)
				w.Header().Add("Access-Control-Allow-Headers", strings.Join(AllowedHeaders, ","))
				w.Header().Add("Access-Control-Allow-Methods", strings.Join(AllowedMethods, ","))
				w.Header().Add("Access-Control-Max-Age", "86400")

				if r.Method == http.MethodOptions {
					w.WriteHeader(http.StatusOK)
					return
				}
			}
		}

		// Otherwise continue
		h.ServeHTTP(w, r)
	})
}
