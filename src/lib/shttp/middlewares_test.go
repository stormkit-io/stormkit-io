package shttp_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/limiter"
)

func TestWithRateLimit(t *testing.T) {
	handler := func(req *shttp.RequestContext) *shttp.Response {
		return shttp.OK()
	}

	opt := &limiter.Options{Limit: 1, Duration: time.Second, Burst: 5}
	mdw := shttp.WithRateLimit(handler, opt)
	req := &shttp.RequestContext{
		Request: &http.Request{
			RemoteAddr: "8.8.8.8",
			URL: &url.URL{
				Path: "my-path",
			},
		},
	}

	for i := 0; i < 10; i++ {
		rec := httptest.NewRecorder()
		req.SetWriter(rec)
		res := mdw(req)

		if i < opt.Burst {
			if res.Status != http.StatusOK {
				t.Fatalf("Was expecting to receive a 200 but received: %d", res.Status)
			}

			if rec.Header().Get("X-RateLimit-Limit") != "1/1s" {
				t.Fatalf("Invalid X-RateLimit-Limit header: %s", rec.Header().Get("X-RateLimit-Limit"))
			}

			remaining, _ := strconv.Atoi(rec.Header().Get("X-RateLimit-Remaining"))

			if remaining <= 0 {
				t.Fatalf("X-RateLimit-Remaining should be greater than 0 but received: %d", remaining)
			}

		} else {
			if res.Status != http.StatusTooManyRequests {
				t.Fatalf("Was expecting to receive a 429 but received: %d", res.Status)
			}

			limit := res.Headers.Get("X-RateLimit-Limit")

			if limit != "1/1s" {
				t.Fatalf("Invalid X-RateLimit-Limit header: %s", limit)
			}

			remaining := res.Headers.Get("X-RateLimit-Remaining")

			if remaining != "0" {
				t.Fatalf("Invalid X-RateLimit-Remaining header: %s", remaining)
			}
		}

		time.Sleep(time.Millisecond)
	}
}
