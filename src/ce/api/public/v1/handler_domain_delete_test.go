package publicapiv1_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type HandlerDeleteDomainSuite struct {
	suite.Suite
	*factory.Factory

	done chan string
	conn databasetest.TestDB

	mockCache *mocks.CacheInterface
}

func (s *HandlerDeleteDomainSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.done = make(chan string)
	s.mockCache = &mocks.CacheInterface{}
	appcache.DefaultCacheService = s.mockCache
	s.Factory = factory.New(s.conn)
}

func (s *HandlerDeleteDomainSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	appcache.DefaultCacheService = nil
}

func (s *HandlerDeleteDomainSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	domain := &buildconf.DomainModel{
		AppID:      app.ID,
		EnvID:      env.ID,
		Name:       "example.org",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
	}

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domain))

	s.mockCache.On("Reset", types.ID(0), "example.org").Return(nil)

	// Required for audits
	admin.SetMockLicense()

	defer func() { admin.ResetMockLicense() }()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf(
			"/v1/domains?domainId=%d&envId=%s&appId=%s",
			domain.ID,
			env.ID.String(),
			app.ID.String(),
		),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.Nil(err)
	s.Len(audits, 1)
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "DELETE:DOMAIN",
		TeamID:      app.TeamID,
		UserID:      usr.ID,
		UserDisplay: usr.Display(),
		AppID:       app.ID,
		EnvID:       env.ID,
		EnvName:     env.Name,
		Diff: &audit.Diff{
			Old: audit.DiffFields{
				DomainName: "example.org",
			},
		},
	}, audits[0])
}

func (s *HandlerDeleteDomainSuite) Test_DomainNotFound() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{})

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodDelete,
		fmt.Sprintf("/v1/domains?domainId=18138&envId=%s&appId=%s", env.ID.String(), app.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	a := assert.New(s.T())
	a.Equal(http.StatusNoContent, response.Code)
}

func TestHandlerDeleteDomain(t *testing.T) {
	suite.Run(t, &HandlerDeleteDomainSuite{})
}
