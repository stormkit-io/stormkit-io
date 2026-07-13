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
			Help:      "HTTP response time in milliseconds for production endpoints",
			Buckets:   []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000},
		},
		[]string{"method", "status_code"},
	)
)

// RecordResponseTime records the response time for a request
func RecordResponseTime(r *http.Request, status int, duration time.Duration) {
	method := r.Method
	statusCode := fmt.Sprintf("%d", utils.GetInt(status, 200))

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

	RTHistogramProdEndpoints.WithLabelValues(method, statusCode).Observe(float64(duration.Milliseconds()))
}
