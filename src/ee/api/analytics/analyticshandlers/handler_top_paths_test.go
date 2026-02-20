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

type HandlerTopPathsSuite struct {
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

func (s *HandlerTopPathsSuite) SetupSuite() {
	s.conn = databasetest.InitTx("top_paths_suite")
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
			analytics_referrers (aggregate_date, referrer, request_path, visit_count, domain_id, referrer_hash, request_path_hash)
		VALUES
			-- Domain 1
			($3, 'google.com',  '/',         1580, $1, decode(md5('google.com'), 'hex'), decode(md5('/'), 'hex')),
			($4, 'yahoo.com',   '/vs-blog',  725,  $1, decode(md5('yahoo.com'), 'hex'), decode(md5('/vs-blog'), 'hex')),
			($5, '',            '/privacy',  2500, $1, decode(md5(''), 'hex'), decode(md5('/privacy'), 'hex')),
			($6, 'reddit.com',  '/vs-blog',  5200, $1, decode(md5('reddit.com'), 'hex'), decode(md5('/vs-blog'), 'hex')),
			($6, 'example.org', '/vs-blog',  5200, $1, decode(md5('example.org'), 'hex'), decode(md5('/vs-blog'), 'hex')),
			-- Domain 2
			($3, 'google.com',  '/',         4900, $2, decode(md5('google.com'), 'hex'), decode(md5('/'), 'hex'))
	`, s.domainID, domain2.ID, s.t0DaysAgo, s.t5DaysAgo, s.t30DaysAgo, s.t45DaysAgo)

	s.NoError(err)
}

func (s *HandlerTopPathsSuite) TearDownSuite() {
	admin.ResetMockLicense()
	s.conn.CloseTx()
}

func (s *HandlerTopPathsSuite) Test_Success() {
	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(analyticshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/analytics/paths?envId=%s&domainId=%d", s.env.ID.String(), s.domainID),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(s.user.ID),
		},
	)

	expected := `{
		"/": 1580,
		"/vs-blog": 725
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(expected, response.String())
}

func TestHandlerTopPaths(t *testing.T) {
	suite.Run(t, &HandlerTopPathsSuite{})
}
