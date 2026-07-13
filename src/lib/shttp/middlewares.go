package shttp

import (
	"context"
	"encoding/binary"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/limiter"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// contextHandler creates a context per request and passes it down to the flow.
func contextHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		h.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GzipHandler performs a gzip compression whenever the client can handle it.
func gzipHandler(h http.Handler) http.Handler {
	return gziphandler.GzipHandler(h)
}

// WithRateLimit limits a given endpoint.
// Limit is the number of events that this endpoint can handle for a given user.
// Duration is the time period which the limit is limited to.
// Burst is the maximum number of tokens the algorithm can accumulate during non-requests.
// The algorithm adds a token to the basket every 1 / rate seconds, with a maximum of burst number
// of tokens. The user will consume from this basket on each request. Once the user is out
// of tokens, a 429 error will be displayed.
func WithRateLimit(handler RequestFunc, options ...*limiter.Options) RequestFunc {
	var opts *limiter.Options

	if len(options) > 0 {
		opts = options[0]
	}

	store := limiter.NewStore(opts)

	return func(req *RequestContext) *Response {
		if req.writer == nil {
			return handler(req)
		}

		hash := []string{}

		for _, k := range store.Hash {
			if k == "ip" {
				hash = append(hash, limiter.IP(req.Request))
			} else if k == "path" {
				hash = append(hash, req.URL().Path)
			}
		}

		// Call the getVisitor function to retreive the rate limiter for
		// the current user.
		visit := store.Get(strings.Join(hash, "-"))
		limit := fmt.Sprintf("%d/%s", store.Limit, store.Duration.String())
		reset := strconv.FormatInt(visit.LastSeen.Add(store.Duration).Unix(), 10)
		if !visit.Limiter.Allow() {
			headers := http.Header{}
			headers.Add("X-RateLimit-Limit", limit)
			headers.Add("X-RateLimit-Reset", reset)
			headers.Add("X-RateLimit-Remaining", "0")

			return &Response{
				Status:  http.StatusTooManyRequests,
				Data:    "Too many requests",
				Headers: headers,
			}
		}

		remaining := store.Limit - visit.Count

		if remaining <= 0 {
			remaining = remaining + int64(store.Burst)
		}

		req.writer.Header().Add("X-RateLimit-Limit", limit)
		req.writer.Header().Add("X-RateLimit-Reset", reset)
		req.writer.Header().Add("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))

		return handler(req)
	}
}

// TimeAuthValid checks whether the time based authentication is valid or not.
func TimeAuthValid(req *RequestContext) (bool, error) {
	header := req.Headers().Get("Authorization")

	if header == "" {
		return false, nil
	}

	var err error
	auth := strings.Replace(header, "Stormkit ", "", 1)
	timestamp, _ := utils.DecodeString(auth)
	timestamp, err = utils.Decrypt(timestamp)

	if err != nil {
		return false, err
	}

	ts := int64(binary.LittleEndian.Uint64(timestamp))

	minutesDiff := time.Since(time.Unix(ts, 0)).Minutes()
	return (minutesDiff <= 5 && minutesDiff >= 0), nil
}
