package jobs_test

import (
	"context"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type JobAnalyticsEventsSuite struct {
	suite.Suite
	*factory.Factory

	conn     databasetest.TestDB
	domainID types.ID
}

func (s *JobAnalyticsEventsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

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
	s.domainID = domain.ID

	now := utils.UnixFrom(time.Now())

	event := func(name, visitor string, domainID types.ID) analytics.Event {
		return analytics.Event{
			AppID:       app.ID,
			EnvID:       env.ID,
			DomainID:    domainID,
			VisitorID:   null.NewString(visitor, visitor != ""),
			EventName:   name,
			RequestPath: null.StringFrom("/"),
			EventTS:     now,
		}
	}

	insertion := event("product_insertion", "v1", domain.ID)
	insertion.RequestID = null.StringFrom("11111111-1111-1111-1111-111111111111")
	insertion.Metadata = null.StringFrom(`{"price": 42}`)

	// trip_creation events carry a `ref` property for breakdown tests.
	withRef := func(name, visitor string, ref string) analytics.Event {
		e := event(name, visitor, domain.ID)
		e.Metadata = null.StringFrom(`{"ref": "` + ref + `"}`)
		return e
	}

	events := []analytics.Event{
		withRef("trip_creation", "v1", "mobile"),
		withRef("trip_creation", "v2", "web"),
		withRef("trip_creation", "v1", "mobile"), // repeat visitor
		withRef("trip_creation", "", "web"),      // server-side, no identity
		insertion,
		event("trip_creation", "v3", 0), // other domain -> excluded from this domain's read
	}

	s.NoError(analytics.NewStore().InsertEvents(context.Background(), events))
}

func (s *JobAnalyticsEventsSuite) Test_InsertEvents_PersistsRequestIDAndMetadata() {
	var requestID, metadata string

	err := s.conn.
		QueryRow(`SELECT request_id::text, metadata::text FROM analytics_events WHERE event_name = 'product_insertion'`).
		Scan(&requestID, &metadata)

	s.NoError(err)
	s.Equal("11111111-1111-1111-1111-111111111111", requestID)
	s.JSONEq(`{"price": 42}`, metadata)
}

func (s *JobAnalyticsEventsSuite) Test_VisitorID_StableWithinDayAndDistinct() {
	a := analytics.VisitorID(analytics.VisitorIDParams{IP: "1.2.3.4", UserAgent: "Mozilla/5.0"})
	b := analytics.VisitorID(analytics.VisitorIDParams{IP: "1.2.3.4", UserAgent: "Mozilla/5.0"})
	c := analytics.VisitorID(analytics.VisitorIDParams{IP: "5.6.7.8", UserAgent: "Mozilla/5.0"})

	s.Equal(a, b)
	s.NotEqual(a, c)
	s.NotEmpty(a)
}

func (s *JobAnalyticsEventsSuite) AfterTest(suiteName, _ string) {
	s.conn.CloseTx()
}

func (s *JobAnalyticsEventsSuite) Test_SyncAnalyticsEvents() {
	ctx := context.Background()

	s.NoError(jobs.SyncAnalyticsEvents(ctx))

	events, err := analytics.NewStore().Events(ctx, analytics.EventsArgs{
		DomainID: s.domainID,
		Span:     analytics.SPAN_30D,
	})

	s.NoError(err)

	counts := map[string]analytics.EventCount{}

	for _, e := range events {
		counts[e.Name] = e
	}

	// total counts every event; unique_actors ignores NULL visitor_id.
	s.Equal(4, counts["trip_creation"].TotalCount)
	s.Equal(2, counts["trip_creation"].UniqueActors)

	s.Equal(1, counts["product_insertion"].TotalCount)
	s.Equal(1, counts["product_insertion"].UniqueActors)

	// The event without a domain is not aggregated.
	s.Len(events, 2)
}

func (s *JobAnalyticsEventsSuite) Test_EventBreakdown_GroupsByProperty() {
	breakdown, err := analytics.NewStore().EventBreakdown(context.Background(), analytics.EventBreakdownArgs{
		DomainID:  s.domainID,
		EventName: "trip_creation",
		Property:  "ref",
		Span:      analytics.SPAN_30D,
	})

	s.NoError(err)

	counts := map[string]analytics.EventCount{}

	for _, b := range breakdown {
		counts[b.Name] = b
	}

	// mobile: v1 twice (total 2, unique 1); web: v2 + null visitor (total 2, unique 1).
	s.Equal(2, counts["mobile"].TotalCount)
	s.Equal(1, counts["mobile"].UniqueActors)
	s.Equal(2, counts["web"].TotalCount)
	s.Equal(1, counts["web"].UniqueActors)
	s.Len(breakdown, 2)
}

func (s *JobAnalyticsEventsSuite) Test_EventPropertyKeys_ReturnsDistinctKeys() {
	keys, err := analytics.NewStore().EventPropertyKeys(context.Background(), analytics.EventBreakdownArgs{
		DomainID:  s.domainID,
		EventName: "trip_creation",
		Span:      analytics.SPAN_30D,
	})

	s.NoError(err)
	s.Equal([]string{"ref"}, keys)
}

func TestJobAnalyticsEventsSuite(t *testing.T) {
	suite.Run(t, &JobAnalyticsEventsSuite{})
}
