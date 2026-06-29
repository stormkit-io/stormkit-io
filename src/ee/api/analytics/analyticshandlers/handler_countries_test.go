package analyticshandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

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
	"github.com/stretchr/testify/suite"
)

type HandlerCountriesSuite struct {
	suite.Suite
	*factory.Factory

	conn       databasetest.TestDB
	user       *factory.MockUser
	env        *factory.MockEnv
	domainID   types.ID
	t0DaysAgo  string
	t5DaysAgo  string
	t30DaysAgo string
	t45DaysAgo string
}

func (s *HandlerCountriesSuite) SetupSuite() {
	s.conn = databasetest.InitTx("countries_suite")
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

	domain2 := &buildconf.DomainModel{
		AppID:      appl.ID,
		EnvID:      s.env.ID,
		Name:       "example.com",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domain2))

	s.domainID = domain.ID

	t1 := time.Now().UTC()
	now := time.Date(t1.Year(), t1.Month(), t1.Day(), 15, 30, 0, 0, t1.Location()).UTC()
	day := time.Hour * 25
	layout := time.DateOnly

	s.t0DaysAgo = now.Format(layout)
	s.t5DaysAgo = now.Add(-5 * day).Format(layout)
	s.t30DaysAgo = now.Add(-30 * day).Format(layout)
	s.t45DaysAgo = now.Add(-45 * day).Format(layout)

	// Daily table
	_, err := s.conn.Exec(`
		INSERT INTO
			analytics_visitors_by_countries (aggregate_date, country_iso_code, visit_count, domain_id)
		VALUES
			-- Domain 1
			($3, 'US', 1580, $1),
			($4, 'EN', 725,  $1),
			($5, 'CH', 410,  $1),
			($6, 'TR', 250,  $1),
			-- Domain 2
			($3, 'US',  200, $2)
	`, s.domainID, domain2.ID, s.t0DaysAgo, s.t5DaysAgo, s.t30DaysAgo, s.t45DaysAgo)

	s.NoError(err)
}

func (s *HandlerCountriesSuite) TearDownSuite() {
	admin.ResetMockLicense()
	s.conn.CloseTx()
}

func (s *HandlerCountriesSuite) Test_Success() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(analyticshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/analytics/countries?envId=%s&domainId=%d", s.env.ID.String(), s.domainID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)

	expected := `{
		"US": 1580,
		"EN": 725
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerCountriesSuite) Test_Forbidden_DomainFromAnotherEnv() {
	otherUser := s.MockUser()
	otherApp := s.MockApp(otherUser)
	otherEnv := s.MockEnv(otherApp)
	otherDomain := &buildconf.DomainModel{
		AppID:      otherApp.ID,
		EnvID:      otherEnv.ID,
		Name:       "tenant-b.example.org",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), otherDomain))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(analyticshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/analytics/countries?envId=%s&domainId=%d", s.env.ID.String(), otherDomain.ID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)

	s.Equal(http.StatusForbidden, response.Code)
}

func TestHandlerCountries(t *testing.T) {
	suite.Run(t, &HandlerCountriesSuite{})
}
