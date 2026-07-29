package functiontrigger_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type InterpolateSuite struct {
	suite.Suite
}

func (s *InterpolateSuite) Test_Interpolate_ReplacesReferencesInAllFields() {
	vars := map[string]string{
		"CRON_SECRET": "s3cr3t",
		"BASE_URL":    "https://api.example.com",
	}

	opts := functiontrigger.Options{
		Method:  "POST",
		URL:     "$BASE_URL/cron?token=${CRON_SECRET}",
		Headers: shttp.Headers{"Authorization": "Bearer $CRON_SECRET"},
		Payload: []byte(`{"key":"$CRON_SECRET"}`),
	}

	out := opts.Interpolate(vars)

	s.Equal("https://api.example.com/cron?token=s3cr3t", out.URL)
	s.Equal("Bearer s3cr3t", out.Headers["Authorization"])
	s.Equal(`{"key":"s3cr3t"}`, string(out.Payload))
	s.Equal("POST", out.Method)
}

func (s *InterpolateSuite) Test_Interpolate_LeavesUnknownReferencesLiteral() {
	opts := functiontrigger.Options{
		URL:     "https://example.com/$NOT_SET",
		Headers: shttp.Headers{"X-Token": "$MISSING"},
	}

	out := opts.Interpolate(map[string]string{"OTHER": "value"})

	s.Equal("https://example.com/$NOT_SET", out.URL)
	s.Equal("$MISSING", out.Headers["X-Token"])
}

func (s *InterpolateSuite) Test_Interpolate_DoesNotMutateReceiver() {
	opts := functiontrigger.Options{
		URL:     "$BASE_URL",
		Headers: shttp.Headers{"Authorization": "Bearer $CRON_SECRET"},
	}

	opts.Interpolate(map[string]string{"BASE_URL": "https://x", "CRON_SECRET": "s3cr3t"})

	s.Equal("$BASE_URL", opts.URL)
	s.Equal("Bearer $CRON_SECRET", opts.Headers["Authorization"])
}

func (s *InterpolateSuite) Test_Interpolate_NoVarsIsNoOp() {
	opts := functiontrigger.Options{URL: "https://example.com/$CRON_SECRET"}

	out := opts.Interpolate(nil)

	s.Equal("https://example.com/$CRON_SECRET", out.URL)
}

func TestInterpolateSuite(t *testing.T) {
	suite.Run(t, new(InterpolateSuite))
}
