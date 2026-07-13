package jobs_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type JobDomainsSuite struct {
	suite.Suite
	*factory.Factory
	conn        databasetest.TestDB
	mockRequest *mocks.RequestInterface
	currentMin  func() int
}

func (s *JobDomainsSuite) SetupSuite() {
	s.currentMin = jobs.CurrentMinute
}

func (s *JobDomainsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockRequest = &mocks.RequestInterface{}
	utils.NewUnix = factory.MockNewUnix
}

func (s *JobDomainsSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	shttp.DefaultRequest = nil
	utils.NewUnix = factory.OriginalNewUnix
	jobs.CurrentMinute = s.currentMin
}

func (s *JobDomainsSuite) Test_PingDomains() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	jobs.CurrentMinute = func() int { return 1 }

	domain := &buildconf.DomainModel{
		AppID:      app.ID,
		EnvID:      env.ID,
		Name:       "www.stormkit.io",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
		Token:      null.StringFrom("my-custom-token"),
	}

	headers := map[string]string{
		"User-Agent": "StormkitBot/1.0 (+https://www.stormkit.io)",
	}

	s.mockRequest.On("Method", http.MethodHead).Return(s.mockRequest).Once()
	s.mockRequest.On("URL", "https://www.stormkit.io").Return(s.mockRequest).Once()
	s.mockRequest.On("Headers", shttp.HeadersFromMap(headers)).Return(s.mockRequest).Once()
	s.mockRequest.On("Do").Return(nil, nil).Once()

	s.NoError(buildconf.DomainStore().Insert(context.Background(), domain))
	s.NoError(jobs.PingDomains(context.Background()))

	domains, err := buildconf.DomainStore().Domains(context.Background(), buildconf.DomainFilters{
		EnvID: env.ID,
	})

	s.NoError(err)
	s.Len(domains, 1)
	s.NotNil(*domains[0].LastPing)
	s.Equal(buildconf.PingResult{
		Status:     http.StatusOK,
		LastPingAt: utils.NewUnix(), // This is mocked so it should return the same value
	}, *domains[0].LastPing)
}

func TestJobDomains(t *testing.T) {
	suite.Run(t, &JobDomainsSuite{})
}
