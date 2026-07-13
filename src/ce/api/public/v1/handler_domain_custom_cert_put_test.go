package publicapiv1_test

import (
	"context"
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
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type HandlerCertPutSuite struct {
	suite.Suite
	*factory.Factory
	conn      databasetest.TestDB
	mockCache *mocks.CacheInterface
}

func (s *HandlerCertPutSuite) SetupSuite() {
	s.mockCache = &mocks.CacheInterface{}
	appcache.DefaultCacheService = s.mockCache
	admin.SetMockLicense()
}

func (s *HandlerCertPutSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerCertPutSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerCertPutSuite) TearDownSuite() {
	appcache.DefaultCacheService = nil
	admin.ResetMockLicense()
}

func (s *HandlerCertPutSuite) Test_Success() {
	usr := s.Factory.MockUser()
	app := s.Factory.MockApp(usr, nil)
	env := s.Factory.MockEnv(app)
	ctx := context.Background()
	domain := &buildconf.DomainModel{
		AppID:    app.ID,
		EnvID:    env.ID,
		Name:     "www.stormkit.io",
		Token:    null.StringFrom("my-token"),
		Verified: true,
	}

	s.NoError(buildconf.DomainStore().Insert(ctx, domain))

	s.mockCache.On("Reset", types.ID(0), domain.Name).Return(nil)

	certKey := "-----BEGIN PRIVATE KEY-----my-key"
	certVal := "-----BEGIN CERTIFICATE-----my-cert"

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPut,
		"/v1/domains/cert",
		map[string]any{
			"envId":     env.ID.String(),
			"appId":     app.ID.String(),
			"domainId":  domain.ID.String(),
			"certValue": certVal,
			"certKey":   certKey,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	domain, err := buildconf.DomainStore().DomainByID(context.Background(), domain.ID)

	s.NoError(err)
	s.NotNil(domain)
	s.NotNil(domain.CustomCert)
	s.Equal(certVal, domain.CustomCert.Value)
	s.Equal(certKey, domain.CustomCert.Key)

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.Nil(err)
	s.Len(audits, 1)
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "UPDATE:DOMAIN",
		UserID:      usr.ID,
		UserDisplay: usr.Display(),
		TeamID:      app.TeamID,
		AppID:       app.ID,
		EnvID:       env.ID,
		EnvName:     env.Name,
		Diff: &audit.Diff{
			Old: audit.DiffFields{
				DomainName:      "www.stormkit.io",
				DomainCertValue: "",
				DomainCertKey:   "",
			},
			New: audit.DiffFields{
				DomainName:      "www.stormkit.io",
				DomainCertValue: certVal,
				DomainCertKey:   certKey,
			},
		},
	}, audits[0])
}

func (s *HandlerCertPutSuite) Test_InvalidKey() {
	usr := s.Factory.MockUser()
	app := s.Factory.MockApp(usr, nil)
	env := s.Factory.MockEnv(app)

	certKey := "my-key"
	certVal := "-----BEGIN CERTIFICATE-----my-cert"

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPut,
		"/v1/domains/cert",
		map[string]any{
			"envId":     env.ID.String(),
			"appId":     app.ID.String(),
			"domainId":  "1",
			"certValue": certVal,
			"certKey":   certKey,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"Invalid private key provided."}`, response.String())
}

func (s *HandlerCertPutSuite) Test_InvalidCert() {
	usr := s.Factory.MockUser()
	app := s.Factory.MockApp(usr, nil)
	env := s.Factory.MockEnv(app)

	certKey := "-----BEGIN PRIVATE KEY-----my-key"
	certVal := "my-cert"

	s.mockCache.On("Reset", types.ID(0), "example.org").Return(nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodPut,
		"/v1/domains/cert",
		map[string]any{
			"envId":     env.ID.String(),
			"appId":     app.ID.String(),
			"domainId":  "1",
			"certValue": certVal,
			"certKey":   certKey,
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error":"Invalid certificate provided."}`, response.String())
}

func TestHandlerCertPut(t *testing.T) {
	suite.Run(t, &HandlerCertPutSuite{})
}
