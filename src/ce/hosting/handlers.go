package hosting

import (
	"errors"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/router"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"go.uber.org/zap"
)

var mux sync.Mutex
var cacheMux sync.Mutex
var internalEndpoints map[string]http.Handler
var cachedHandlers map[string]http.Handler

type InternalHandlerOpts struct {
	Health bool
	API    bool
	App    bool
}

func woScheme(s, def string) string {
	if !strings.Contains(s, "//") {
		return def
	}

	return strings.Split(s, "//")[1]
}

func woPort(s string) string {
	return strings.Split(s, ":")[0]
}

func getHandlerFromHost(host string, opts InternalHandlerOpts) http.Handler {
	mux.Lock()
	defer mux.Unlock()

	if internalEndpoints == nil {
		cnf := admin.MustConfig().DomainConfig

		internalEndpoints = map[string]http.Handler{}

		if opts.API {
			// This makes sure both with and without port work
			ep := woScheme(cnf.API, "api")
			internalEndpoints[ep] = apiHandler()
			internalEndpoints[woPort(ep)] = internalEndpoints[ep]
		}

		if opts.Health {
			ep := woScheme(cnf.Health, "health")
			internalEndpoints[ep] = healthHandler()
			internalEndpoints[woPort(ep)] = internalEndpoints[ep]
		}

		if opts.App {
			ep := woScheme(cnf.App, "stormkit")
			internalEndpoints[ep] = uiHandler()
			internalEndpoints[woPort(ep)] = internalEndpoints[ep]
		}

		keys := []string{}

		for k := range internalEndpoints {
			keys = append(keys, k)
		}

		slog.Debug(slog.LogOpts{
			Msg: "handlers registered",
			Payload: []zap.Field{
				zap.String("keys", strings.Join(keys, ", ")),
			},
			Level: slog.DL2,
		})
	}

	return internalEndpoints[host]
}

func InternalHandlers(opts InternalHandlerOpts) func(h http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handler := getHandlerFromHost(r.Host, opts)

			slog.Debug(slog.LogOpts{
				Msg:   "incoming request",
				Level: slog.DL3,
				Payload: []zap.Field{
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.String("query", r.URL.RawQuery),
					zap.String("host", r.Host),
					zap.String("remote_addr", r.RemoteAddr),
					zap.String("user_agent", r.Header.Get("User-Agent")),
					zap.String("origin", r.Header.Get("Origin")),
				},
			})

			if handler != nil {
				handler.ServeHTTP(w, r)
				return
			}

			h.ServeHTTP(w, r)
		})
	}

}

func apiHandler() http.Handler {
	cacheMux.Lock()
	defer cacheMux.Unlock()

	if cachedHandlers == nil {
		cachedHandlers = map[string]http.Handler{}
	}

	if cachedHandlers["api"] == nil {
		cachedHandlers["api"] = router.Get().Handler()
	}

	return cachedHandlers["api"]
}

func healthHandler() http.Handler {
	cacheMux.Lock()
	defer cacheMux.Unlock()

	if cachedHandlers == nil {
		cachedHandlers = map[string]http.Handler{}
	}

	if cachedHandlers["health"] == nil {
		healthRouter := shttp.NewRouter()
		healthService := healthRouter.NewService()
		healthService.NewEndpoint("/").CatchAll(func(rc *shttp.RequestContext) *shttp.Response {
			return &shttp.Response{
				Status:  200,
				Data:    "OK",
				Headers: shttp.HeadersFromMap(map[string]string{"Content-Type": "text/html; charset=utf-8"}),
			}
		}, "")

		cachedHandlers["health"] = healthRouter.Handler()
	}

	return cachedHandlers["health"]
}

func uiHandler() http.Handler {
	cacheMux.Lock()
	defer cacheMux.Unlock()

	if cachedHandlers == nil {
		cachedHandlers = map[string]http.Handler{}
	}

	if cachedHandlers["ui"] != nil {
		return cachedHandlers["ui"]
	}

	uiRouter := shttp.NewRouter()
	uiService := uiRouter.NewService()
	uiService.NewEndpoint("/").CatchAll(func(rc *shttp.RequestContext) *shttp.Response {
		relFileName := rc.URL().Path

		if relFileName == "/" {
			relFileName = "index.html"
		}

		absFileName := path.Join("/home/stormkit/ui", relFileName)
		headers := make(http.Header)

		file, err := os.Open(absFileName)

		// Fallback to index.html
		if err != nil && errors.Is(err, os.ErrNotExist) {
			absFileName = "/home/stormkit/ui/index.html"
			file, err = os.Open(absFileName)

			if err != nil && errors.Is(err, os.ErrNotExist) {
				return shttp.NotFound()
			}
		}

		if absFileName == "/home/stormkit/ui/index.html" {
			headers.Set("X-SK-API", admin.MustConfig().ApiURL(""))
		}

		fileInfo, err := os.Stat(absFileName)

		if err != nil {
			return shttp.NotFound()
		}

		return &shttp.Response{
			Status:  http.StatusOK,
			Headers: headers,
			BeforeClose: func() {
				if err := file.Close(); err != nil {
					slog.Errorf("error while closing file: %s", err.Error())
				}
			},
			ServeContent: &shttp.ServeContent{
				Content: file,
				Name:    path.Base(absFileName),
				ModTime: fileInfo.ModTime(),
			},
		}
	}, "")

	cachedHandlers["ui"] = uiRouter.Handler()
	return cachedHandlers["ui"]
}
