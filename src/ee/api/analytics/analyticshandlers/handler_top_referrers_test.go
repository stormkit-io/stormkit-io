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

type HandlerReferrersSuite struct {
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

func (s *HandlerReferrersSuite) SetupSuite() {
	s.conn = databasetest.InitTx("referrers_suite")
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

	t1 := time.Now()
	now := time.Date(t1.Year(), t1.Month(), t1.Day(), 15, 30, 0, 0, t1.Location())
	day := time.Hour * 25
	layout := time.DateOnly

	s.t0DaysAgo = now.Format(layout)
	s.t5DaysAgo = now.Add(-5 * day).Format(layout)
	s.t30DaysAgo = now.Add(-30 * day).Format(layout)
	s.t45DaysAgo = now.Add(-45 * day).Format(layout)

	// Daily table
	_, err := s.conn.Exec(`
		INSERT INTO
			analytics_referrers (aggregate_date, referrer, request_path, visit_count, domain_id, referrer_hash, request_path_hash)
		VALUES
			-- Domain 1
			($3, 'google.com',  '/',         1580, $1, decode(md5('google.com'), 'hex'), decode(md5('/'), 'hex')),
			($4, 'yahoo.com',   '/vs-blog',  725,  $1, decode(md5('yahoo.com'), 'hex'), decode(md5('/vs-blog'), 'hex')),
			($5, '',            '/vs-blog',  2500, $1, decode(md5(''), 'hex'), decode(md5('/vs-blog'), 'hex')),
			($6, 'reddit.com',  '/vs-blog',  5200, $1, decode(md5('reddit.com'), 'hex'), decode(md5('/vs-blog'), 'hex')),
			($6, 'example.org', '/vs-blog',  5200, $1, decode(md5('example.org'), 'hex'), decode(md5('/vs-blog'), 'hex')),
			-- Domain 2
			($3, 'google.com',  '/',         4900, $2, decode(md5('google.com'), 'hex'), decode(md5('/'), 'hex'))
		ON CONFLICT
			(aggregate_date, referrer_hash, request_path_hash, domain_id)
		DO UPDATE SET
			visit_count = EXCLUDED.visit_count
	`, s.domainID, domain2.ID, s.t0DaysAgo, s.t5DaysAgo, s.t30DaysAgo, s.t45DaysAgo)

	s.NoError(err)
}

func (s *HandlerReferrersSuite) TearDownSuite() {
	admin.ResetMockLicense()
	s.conn.CloseTx()
}

func (s *HandlerReferrersSuite) Test_Success_Referrers() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(analyticshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf(
			"/analytics/referrers?envId=%s&domainId=%d",
			s.env.ID.String(),
			s.domainID,
		),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)

	expected := `{
		"google.com": 1580,
		"yahoo.com": 725
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func (s *HandlerReferrersSuite) Test_Success_ReferrersForPath() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(analyticshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf(
			"/analytics/referrers?envId=%s&domainId=%d&requestPath=/",
			s.env.ID.String(),
			s.domainID,
		),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)

	expected := `{
		"google.com": 1580
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerReferrers(t *testing.T) {
	suite.Run(t, &HandlerReferrersSuite{})
}
