package tracking

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"go.uber.org/zap"
)

var (
	// RTHistogramProdEndpoints tracks HTTP response times
	RTHistogramProdEndpoints = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "stormkit",
			Subsystem: "lb",
			Name:      "response_time_ms",
			Help:      "Time in milliseconds to produce a response on production endpoints, up to the point it is ready to be written; the transfer to the client is not included",
			Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		},
		[]string{"method", "status_code"},
	)

	// RTTotalHistogramProdEndpoints tracks the whole request, transfer included.
	// Its gap to RTHistogramProdEndpoints is mostly the time spent getting the
	// body to the client, which says more about the visitor's connection and the
	// size of the asset than about the server; encoding the body and compressing
	// it on the way out sit in that gap too, at about a millisecond. The top
	// bucket is higher because a large download on a slow link legitimately
	// takes longer than any server work.
	RTTotalHistogramProdEndpoints = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "stormkit",
			Subsystem: "lb",
			Name:      "request_total_ms",
			Help:      "Time in milliseconds from receiving a request on a production endpoint until its body has been handed to the connection, so the transfer to the client is included",
			Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000, 30000},
		},
		[]string{"method", "status_code"},
	)
)

// RecordResponseTime records how long a request took to produce its response.
func RecordResponseTime(r *http.Request, status int, duration time.Duration) {
	method, statusCode := requestLabels(r, status)

	RTHistogramProdEndpoints.WithLabelValues(method, statusCode).Observe(float64(duration.Milliseconds()))
}

// RecordRequestTotalTime records how long a request took end to end, body
// transfer included.
func RecordRequestTotalTime(r *http.Request, status int, duration time.Duration) {
	method, statusCode := requestLabels(r, status)

	RTTotalHistogramProdEndpoints.WithLabelValues(method, statusCode).Observe(float64(duration.Milliseconds()))
}

// requestLabels folds the method and status into the few label values the
// histograms carry, so cardinality stays flat whatever a handler returns.
func requestLabels(r *http.Request, status int) (method, statusCode string) {
	method = r.Method
	statusCode = fmt.Sprintf("%d", utils.GetInt(status, 200))

	if method == http.MethodPost || method == http.MethodPut || method == http.MethodDelete || method == http.MethodPatch {
		method = http.MethodPost
	} else if method != http.MethodGet {
		method = "OTHER"
	}

	if statusCode == "" {
		statusCode = "200"
	}

	if statusCode != "200" && statusCode != "304" {
		switch statusCode[0] {
		case '2':
			statusCode = "2xx"
		case '3':
			statusCode = "3xx"
		case '4':
			statusCode = "4xx"
		case '5':
			statusCode = "5xx"
		default:
			statusCode = "other"
			slog.Debug(slog.LogOpts{
				Msg:   "metrics unknown status code",
				Level: slog.DL3,
				Payload: []zap.Field{
					zap.String("method", method),
					zap.String("path", r.URL.Path),
					zap.String("host", r.Host),
					zap.String("status_code", statusCode),
				},
			})
		}
	}

	return method, statusCode
}
