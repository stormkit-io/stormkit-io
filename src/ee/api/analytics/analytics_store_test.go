package analytics_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
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
	s.conn = databasetest.InitTx("analytics_store_suite")
	s.Factory = factory.New(s.conn)
}

func (s *StoreSuite) TearDownSuite() {
	s.conn.CloseTx()
}

func (s *StoreSuite) Test_InsertRecords() {
	app := s.MockApp(nil)
	env := s.MockEnv(app)
	domain := &buildconf.DomainModel{
		AppID:      app.ID,
		EnvID:      env.ID,
		Name:       "www.stormkit.io",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
		Token:      null.StringFrom("my-custom-token"),
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domain))

	records := []analytics.Record{
		{
			AppID:       app.ID,
			EnvID:       env.ID,
			VisitorIP:   "85.97.11.98",
			RequestTS:   utils.NewUnix(),
			RequestPath: "/",
			StatusCode:  http.StatusOK,
			Referrer:    null.NewString("https://www.google.com", true),
			UserAgent:   null.NewString("my-user-agent", true),
			DomainID:    domain.ID,
		},
		{
			AppID:       app.ID,
			EnvID:       env.ID,
			VisitorIP:   "invalid-ip",
			RequestTS:   utils.NewUnix(),
			RequestPath: "/",
			StatusCode:  http.StatusOK,
			Referrer:    null.NewString("www.yahoo.com", true),
			UserAgent:   null.NewString("my-user-agent", true),
			DomainID:    domain.ID,
		},
	}

	ctx := context.Background()
	err := analytics.NewStore().InsertRecords(ctx, records)

	s.NoError(err)

	rows, err := s.conn.PrepareOrPanic("SELECT visitor_ip, request_path, referrer FROM analytics").QueryContext(ctx)
	s.NoError(err)
	s.NotNil(rows)

	defer rows.Close()

	dbRecords := make([]struct {
		Referrer null.String
		RecordIP null.String
		Path     string
	}, 2)

	i := 0

	for rows.Next() {
		s.NoError(rows.Scan(&dbRecords[i].RecordIP, &dbRecords[i].Path, &dbRecords[i].Referrer))
		i = i + 1
	}

	// visitor_ip stores a salted hash, not the raw IP: 32 hex chars, never the
	// address itself. The raw IP is used only transiently for geo lookup.
	s.Regexp("^[a-f0-9]{32}$", dbRecords[0].RecordIP.ValueOrZero())
	s.NotContains(dbRecords[0].RecordIP.ValueOrZero(), "85.97.11.98")
	s.Equal("/", dbRecords[0].Path)
	s.Equal("google.com", dbRecords[0].Referrer.ValueOrZero())
	s.Equal("", dbRecords[1].RecordIP.ValueOrZero())
	s.Equal("/", dbRecords[1].Path)
	s.Equal("yahoo.com", dbRecords[1].Referrer.ValueOrZero())
}

func TestStore(t *testing.T) {
	suite.Run(t, &StoreSuite{})
}
