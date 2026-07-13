package limiter_test

import (
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp/limiter"
)

func TestNewStore(t *testing.T) {
	opts := &limiter.Options{
		Limit:    1000,
		Duration: time.Hour,
	}

	store := limiter.NewStore(opts)

	if store.Limit != opts.Limit {
		t.Fatalf("Limits are not equal")
	}

	if store.Duration != time.Hour {
		t.Fatalf("Durations are not equal")
	}

	if store.Burst != 10 {
		t.Fatalf("Bursts are not equal")
	}
}
