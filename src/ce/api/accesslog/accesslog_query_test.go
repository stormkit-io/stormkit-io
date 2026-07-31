package accesslog_test

import (
	"net/url"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/accesslog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type QuerySuite struct {
	suite.Suite
}

// The cursor has to survive the round trip at microsecond fidelity: truncating
// it to whole seconds would skip every entry sharing the cursor's second.
func (s *QuerySuite) Test_Cursor_RoundTrip() {
	ts := time.Date(2026, 7, 31, 8, 32, 0, 830779000, time.UTC)

	entry := accesslog.AccessLog{
		ID:        types.ID(3971512),
		RequestTS: utils.UnixFrom(ts),
	}

	cursor := entry.Cursor()
	s.Require().NotEmpty(cursor)

	params := accesslog.SelectLogsParamsFromQuery(url.Values{"cursor": {cursor}})

	s.Equal(types.ID(3971512), params.BeforeID)
	s.Require().True(params.BeforeTS.Valid)
	s.Equal(ts.UnixMicro(), params.BeforeTS.Time.UnixMicro())
}

// A cursor that cannot be decoded reads as "no cursor" rather than failing the
// request or silently filtering on half of the sort key.
func (s *QuerySuite) Test_Cursor_Malformed() {
	for _, cursor := range []string{
		"",
		"not-base64!!",
		utils.EncodeToString([]byte("missing-separator")),
		utils.EncodeToString([]byte("0.123")),
		utils.EncodeToString([]byte("abc.123")),
	} {
		params := accesslog.SelectLogsParamsFromQuery(url.Values{"cursor": {cursor}})

		s.Zero(params.BeforeID, cursor)
		s.False(params.BeforeTS.Valid, cursor)
	}
}

// Both time bounds are always set so the query prunes to the day partitions it
// actually needs.
func (s *QuerySuite) Test_TimeBounds_Default() {
	params := accesslog.SelectLogsParamsFromQuery(url.Values{})

	s.Require().True(params.From.Valid)
	s.Require().True(params.To.Valid)
	s.WithinDuration(time.Now().Add(-accesslog.DefaultWindow), params.From.Time, time.Minute)
	s.WithinDuration(time.Now(), params.To.Time, time.Minute)
}

func TestQuery(t *testing.T) {
	suite.Run(t, &QuerySuite{})
}
