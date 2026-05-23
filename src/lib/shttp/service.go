package shttp

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/tracking"
	"go.uber.org/zap"
)

func init() {
	http.DefaultClient.Timeout = time.Minute * 2
}

// ServiceFunc represents a service function signature.
type ServiceFunc func(r *Router) *Service

// RequestFunc represents a request function signature.
type RequestFunc func(*RequestContext) *Response

// HandlerFunc represents a handler function signature for an endpoint.
type HandlerFunc func(method, path string, fn RequestFunc) *Response

// Service is a service wrapper for the given endpoints.
type Service struct {
	Endpoints []*ServiceEndpoint
	router    *Router
	handlers  map[string]RequestFunc
}

// NewEndpoint returns a new endpoint handler.
// The returned instance can be used to attach handlers to various endpoints.
func (s *Service) NewEndpoint(ep string) *ServiceEndpoint {
	return &ServiceEndpoint{
		service: s,
		prefix:  ep,
	}
}

// Handlers returns the registered handler endpoints.
// @deprecated Use HandlerKeys instead.
func (s *Service) Handlers() []string {
	handlers := []string{}

	for k := range s.handlers {
		handlers = append(handlers, k)
	}

	return handlers
}

// HandlerKeys returns the registered handler endpoints.
func (s *Service) HandlerKeys() []string {
	handlers := []string{}

	for k := range s.handlers {
		handlers = append(handlers, k)
	}

	sort.Strings(handlers)

	return handlers
}

// HandlerFuncs returns the registered handler endpoints, mapped to their functions.
func (s *Service) HandlerFuncs() map[string]RequestFunc {
	return s.handlers
}

// Router returns the associated router.
func (s *Service) Router() *Router {
	return s.router
}

// ServiceEndpoint is a handler for endpoints. It allows attaching
// handlers to various endpoints
type ServiceEndpoint struct {
	service     *Service
	prefix      string
	middlewares []RequestFunc
}

func (se *ServiceEndpoint) Middleware(handler RequestFunc) *ServiceEndpoint {
	if se.middlewares == nil {
		se.middlewares = []RequestFunc{}
	}

	se.middlewares = append(se.middlewares, handler)
	return se
}

// Handler is a middleware for generic routes.
func (se *ServiceEndpoint) Handler(method, path string, handler RequestFunc) *ServiceEndpoint {
	endpoint := se.prefix + path
	wrapper := func(req *RequestContext) *Response {
		for _, mw := range se.middlewares {
			// If a middleware returns a response, we terminate it here.
			if res := mw(req); res != nil {
				return res
			}
		}

		return handler(req)
	}

	se.service.router.mux.HandleFunc(
		endpoint,
		func(w http.ResponseWriter, r *http.Request) {
			req := requestContext(w, r)
			res := wrapper(req)
			se.Send(w, req, res)
		},
	).Methods(method)

	if config.IsTest() {
		if se.service.handlers == nil {
			se.service.handlers = map[string]RequestFunc{}
		}

		se.service.handlers[fmt.Sprintf("%s:%s", method, endpoint)] = wrapper
	}

	return se
}

// CatchAll is a catch all handler. All requests will be
func (se *ServiceEndpoint) CatchAll(handler RequestFunc, devDomain string) *ServiceEndpoint {
	cnf := config.Get()
	trackingEnabled := cnf.Tracking != nil && cnf.Tracking.Prometheus

	slowThresholdMs := 0

	if cnf.Tracking != nil {
		slowThresholdMs = cnf.Tracking.SlowRequestThresholdMs
	}

	wrapper := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		req := requestContext(w, r)
		res := handler(req)
		se.Send(w, req, res)

		elapsed := time.Since(start)
		isDevHost := devDomain != "" && strings.HasSuffix(req.HostName(), devDomain)

		// Record the response time if tracking is enabled
		// and the request is not for the dev domain.
		if trackingEnabled && !isDevHost {
			tracking.RecordResponseTime(r, res.Status, elapsed)
		}

		if slowThresholdMs > 0 && !isDevHost && elapsed.Milliseconds() >= int64(slowThresholdMs) {
			slog.Debug(slog.LogOpts{
				Msg:   "slow request",
				Level: slog.DL2,
				Payload: []zap.Field{
					zap.String("method", r.Method),
					zap.String("host", req.HostName()),
					zap.String("path", r.URL.Path),
					zap.Int("status", res.Status),
					zap.Int64("duration_ms", elapsed.Milliseconds()),
					zap.String("remote", r.RemoteAddr),
					zap.String("ua", r.UserAgent()),
				},
			})
		}
	})

	se.service.router.mux.PathPrefix("").Handler(wrapper)

	return se
}

func (se *ServiceEndpoint) attachHeadersAndCookies(w http.ResponseWriter, res *Response) {
	if res.Headers == nil {
		res.Headers = make(http.Header)
	}

	if res.Cookies != nil {
		for _, c := range res.Cookies {
			http.SetCookie(w, &c)
		}
	}

	for k, v := range res.Headers {
		for _, h := range v {
			w.Header().Add(k, h)
		}
	}
}

// Send sends a response to the client.
func (se *ServiceEndpoint) Send(w http.ResponseWriter, req *RequestContext, res *Response) {
	if res == nil {
		return
	}

	if res.Redirect != nil {
		se.attachHeadersAndCookies(w, res)
		req.Redirect(*res.Redirect, res.Status)
		return
	}

	if res.Headers == nil {
		res.Headers = make(http.Header)
	}

	ce := res.Headers.Get("Content-Encoding")

	// If the response is already compressed, do not re-compress it.
	if ce == "gzip" || ce == "bz" || ce == "br" {
		switch t := w.(type) {
		case *gziphandler.GzipResponseWriter:
			se.Write(t.ResponseWriter, req.Request, res)
			return
		case gziphandler.GzipResponseWriterWithCloseNotify:
			se.Write(t.GzipResponseWriter.ResponseWriter, req.Request, res)
			return
		default:
			break
		}
	}

	se.Write(w, req.Request, res)
}

// Write writes a response to the client.
func (se *ServiceEndpoint) Write(w http.ResponseWriter, req *http.Request, res *Response) {
	if res.BeforeClose != nil {
		defer res.BeforeClose()
	}

	if res.ServeContent == nil {
		if val := res.Headers.Get("Content-Type"); val == "" {
			res.Headers.Set("Content-Type", "application/json")
		}
	}

	se.attachHeadersAndCookies(w, res)

	if res.ServeContent != nil {
		if res.ServeContent.Name == "" {
			res.ServeContent.Name = req.URL.Path
		}

		http.ServeContent(w, req, res.ServeContent.Name, res.ServeContent.ModTime, res.ServeContent.Content)
		return
	}

	if res.Status == 0 {
		res.Status = http.StatusOK
	}

	w.WriteHeader(res.Status)

	switch data := res.Data.(type) {

	// If the response data is typeof a byte array then print it.
	case []byte:
		w.Write(data)
		return

	case string:
		w.Write([]byte(data))
		return

	case io.ReadCloser:
		body, err := io.ReadAll(data)

		if err != nil {
			w.Write([]byte(err.Error()))
		} else {
			w.Write(body)
		}
		return

	// If it is a slice, return an items object.
	case []interface{}:
		encoder := json.NewEncoder(w)
		encoder.Encode(map[string]interface{}{
			"items": data,
		})
		return

	// Otherwise default to encode
	default:
		if res.Data == nil {
			w.Write([]byte(""))
			return
		}

		encoder := json.NewEncoder(w)
		if err := encoder.Encode(res.Data); err != nil {
			slog.Error(err.Error())
			w.Write([]byte(""))
		}

		return
	}
}

func requestContext(w http.ResponseWriter, r *http.Request) *RequestContext {
	return &RequestContext{
		writer:    w,
		Request:   r,
		StartTime: time.Now(),
	}
}
