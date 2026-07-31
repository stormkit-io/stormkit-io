package accesslog_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/accesslog"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type StoreSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *StoreSuite) SetupSuite() {
	s.conn = databasetest.InitTx("accesslog_store_suite")
	s.Factory = factory.New(s.conn)
}

func (s *StoreSuite) TearDownSuite() {
	s.conn.CloseTx()
}

func (s *StoreSuite) Test_InsertAndSelect() {
	app := s.MockApp(nil)
	env := s.MockEnv(app)
	ctx := context.Background()

	logs := []accesslog.AccessLog{
		{
			AppID:       app.ID,
			EnvID:       env.ID,
			DomainID:    42,
			HostName:    "example.com",
			RequestTS:   utils.NewUnix(),
			Method:      http.MethodGet,
			RequestPath: "/",
			StatusCode:  http.StatusOK,
			ClientIP:    "203.0.113.7",
			UserAgent:   "Mozilla/5.0 (Macintosh)",
			Referrer:    "https://www.google.com",
			IsBot:       false,
			BytesSent:   1234,
			RequestID:   null.StringFrom("11111111-1111-1111-1111-111111111111"),
		},
		{
			AppID:       app.ID,
			EnvID:       env.ID,
			DomainID:    42,
			HostName:    "example.com",
			RequestTS:   utils.NewUnix(),
			Method:      http.MethodGet,
			RequestPath: "/robots.txt",
			StatusCode:  http.StatusOK,
			ClientIP:    "198.51.100.9",
			UserAgent:   "Googlebot/2.1 (+http://www.google.com/bot.html)",
			IsBot:       true,
			BytesSent:   42,
		},
	}

	s.NoError(accesslog.NewStore().InsertLogs(ctx, logs))

	window := accesslog.SelectLogsParams{
		AppID: app.ID,
		From:  utils.UnixFrom(time.Now().Add(-time.Hour)),
		To:    utils.UnixFrom(time.Now().Add(time.Hour)),
	}

	all, err := accesslog.NewStore().SelectLogs(ctx, window)
	s.NoError(err)
	s.Len(all, 2)

	// The raw client IP and full user-agent are persisted unmasked — the inverse
	// of the analytics table, which stores a salted IP hash.
	byPath := map[string]accesslog.AccessLog{}
	for _, l := range all {
		byPath[l.RequestPath] = l
	}

	s.Equal("203.0.113.7", byPath["/"].ClientIP)
	s.Equal("Mozilla/5.0 (Macintosh)", byPath["/"].UserAgent)
	s.Equal("https://www.google.com", byPath["/"].Referrer)
	s.Equal(int64(1234), byPath["/"].BytesSent)
	s.False(byPath["/"].IsBot)
	s.True(byPath["/robots.txt"].IsBot)
	s.Equal("198.51.100.9", byPath["/robots.txt"].ClientIP)

	// is_bot filter narrows to bot traffic only.
	onlyBots := window
	isBot := true
	onlyBots.IsBot = &isBot

	bots, err := accesslog.NewStore().SelectLogs(ctx, onlyBots)
	s.NoError(err)
	s.Len(bots, 1)
	s.Equal("/robots.txt", bots[0].RequestPath)

	// path prefix filter.
	prefix := window
	prefix.Path = "/robots"

	robots, err := accesslog.NewStore().SelectLogs(ctx, prefix)
	s.NoError(err)
	s.Len(robots, 1)
	s.Equal("/robots.txt", robots[0].RequestPath)
}

// Paging walks the same (request_timestamp, log_id) tuple the query orders by,
// so entries sharing a timestamp are returned exactly once rather than being
// skipped or repeated across page boundaries.
func (s *StoreSuite) Test_SelectLogs_KeysetPagination() {
	app := s.MockApp(nil)
	env := s.MockEnv(app)
	ctx := context.Background()

	shared := utils.UnixFrom(time.Now().Add(-10 * time.Minute))
	older := utils.UnixFrom(time.Now().Add(-20 * time.Minute))

	logs := []accesslog.AccessLog{}

	for _, tc := range []struct {
		path string
		ts   utils.Unix
	}{
		{"/a", shared},
		{"/b", shared},
		{"/c", shared},
		{"/d", older},
		{"/e", older},
	} {
		logs = append(logs, accesslog.AccessLog{
			AppID:       app.ID,
			EnvID:       env.ID,
			HostName:    "example.com",
			RequestTS:   tc.ts,
			Method:      http.MethodGet,
			RequestPath: tc.path,
			StatusCode:  http.StatusOK,
		})
	}

	s.Require().NoError(accesslog.NewStore().InsertLogs(ctx, logs))

	params := accesslog.SelectLogsParams{
		AppID: app.ID,
		From:  utils.UnixFrom(time.Now().Add(-time.Hour)),
		To:    utils.UnixFrom(time.Now()),
		Limit: 2,
	}

	seen := []string{}

	for range logs {
		page, err := accesslog.NewStore().SelectLogs(ctx, params)
		s.Require().NoError(err)

		if len(page) == 0 {
			break
		}

		if len(page) > params.Limit {
			page = page[:params.Limit]
		}

		for _, l := range page {
			seen = append(seen, l.RequestPath)
		}

		last := page[len(page)-1]
		params.BeforeID = last.ID
		params.BeforeTS = last.RequestTS
	}

	s.Len(seen, len(logs))
	s.ElementsMatch([]string{"/a", "/b", "/c", "/d", "/e"}, seen)

	// Newest first: the entries sharing the later timestamp all precede the
	// older pair, regardless of how the pages happened to split.
	s.ElementsMatch([]string{"/a", "/b", "/c"}, seen[:3])
	s.ElementsMatch([]string{"/d", "/e"}, seen[3:])
}

func TestStore(t *testing.T) {
	suite.Run(t, &StoreSuite{})
}
