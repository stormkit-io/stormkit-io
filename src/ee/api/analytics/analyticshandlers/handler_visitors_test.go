package analyticshandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics/analyticshandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type HandlerVisitorsSuite struct {
	suite.Suite
	*factory.Factory

	conn       databasetest.TestDB
	user       *factory.MockUser
	env        *factory.MockEnv
	domainID   types.ID
	t0DaysAgo  string
	t5DaysAgo  string
	t7DaysAgo  string
	t15DaysAgo string
	t30DaysAgo string
	t45DaysAgo string
}

func (s *HandlerVisitorsSuite) SetupSuite() {
	s.conn = databasetest.InitTx("visitors_suite")
	s.Factory = factory.New(s.conn)

	admin.SetMockLicense()

	s.user = s.MockUser()
	appl := s.MockApp(s.user)
	s.env = s.MockEnv(appl)
	domain := &buildconf.DomainModel{
		AppID:      appl.ID,
		EnvID:      s.env.ID,
		Name:       "example.org",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domain))

	// Insert a secondary domain just to make sure we're fetching only relevant data
	domain2 := &buildconf.DomainModel{
		AppID:      appl.ID,
		EnvID:      s.env.ID,
		Name:       "example.com",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domain2))

	s.domainID = domain.ID

	t1 := time.Now()
	now := time.Date(t1.Year(), t1.Month(), t1.Day(), 15, 30, 0, 0, t1.Location())
	day := time.Hour * 25
	layout := "2006-01-02"

	s.t0DaysAgo = now.Format(layout)
	s.t5DaysAgo = now.Add(-5 * day).Format(layout)
	s.t7DaysAgo = now.Add(-7 * day).Format(layout)
	s.t15DaysAgo = now.Add(-15 * day).Format(layout)
	s.t30DaysAgo = now.Add(-30 * day).Format(layout)
	s.t45DaysAgo = now.Add(-45 * day).Format(layout)

	// Daily table
	_, err := s.conn.Exec(`
		INSERT INTO
			analytics_visitors_agg_200 (aggregate_date, total_visitors, unique_visitors, domain_id)
		VALUES
			-- Domain 1
			($3, 15, 5, $1),
			($4, 5, 2, $1),
			($5, 7, 3, $1),
			($6, 15, 7, $1),
			($7, 30, 15, $1),
			($8, 45, 22, $1),
			-- Domain 2
			($3, 24010, 600, $2)
	`, s.domainID, domain2.ID, s.t0DaysAgo, s.t5DaysAgo, s.t7DaysAgo, s.t15DaysAgo, s.t30DaysAgo, s.t45DaysAgo)

	s.NoError(err)

	// Hourly table
	_, err = s.conn.Exec(`
		INSERT INTO
			analytics_visitors_agg_hourly_200 (aggregate_date, total_visitors, unique_visitors, domain_id)
		VALUES
			-- Domain 1
			($3, 15, 5, $1),
			-- Domain 2
			($3, 24010, 600, $2)
	`, s.domainID, domain2.ID, now)

	s.NoError(err)
}

func (s *HandlerVisitorsSuite) TearDownSuite() {
	admin.ResetMockLicense()
	s.conn.CloseTx()
}

func (s *HandlerVisitorsSuite) Test_Success_30d_Unique() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(analyticshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/analytics/visitors?envId=%d&ts=30d&domainId=%d", s.env.ID, s.domainID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"%s": { "total": 5, "unique": 2 },
		"%s": { "total": 7, "unique": 3 },
		"%s": { "total": 15, "unique": 7 },
		"%s": { "total": 30, "unique": 15 }
	}`, s.t5DaysAgo, s.t7DaysAgo, s.t15DaysAgo, s.t30DaysAgo)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerVisitorsSuite) Test_Success_7d_Unique() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(analyticshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/analytics/visitors?envId=%d&ts=7d&domainId=%d", s.env.ID, s.domainID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"%s": { "total": 5, "unique": 2 },
		"%s": { "total": 7, "unique": 3 }
	}`, s.t5DaysAgo, s.t7DaysAgo)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerVisitorsSuite) Test_Success_24h_Unique() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(analyticshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/analytics/visitors?envId=%d&ts=24h&domainId=%d", s.env.ID, s.domainID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)

	expected := fmt.Sprintf(`{
		"%s 15:00": { "total": 15, "unique": 5 }
	}`, s.t0DaysAgo)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerVisitors(t *testing.T) {
	suite.Run(t, &HandlerVisitorsSuite{})
}
