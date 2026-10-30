package functiontrigger_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type TriggerWallPassSuite struct {
	suite.Suite
	*factory.Factory

	conn        databasetest.TestDB
	mockRequest *mocks.RequestInterface
	env         *factory.MockEnv
}

func (s *TriggerWallPassSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockRequest = &mocks.RequestInterface{}
	shttp.DefaultRequest = s.mockRequest
	s.env = s.MockEnv(nil)

	_, err := s.conn.Exec(`
		INSERT INTO domains (app_id, env_id, domain_name, domain_verified)
		VALUES ($1, $2, 'www.example.org', true), ($1, $2, 'unverified.example.org', false);
	`, s.env.AppID, s.env.ID)

	s.Require().NoError(err)
	s.setWall(authwall.StatusAll)
}

func (s *TriggerWallPassSuite) AfterTest(_, _ string) {
	shttp.DefaultRequest = nil
	s.conn.CloseTx()
}

// setWall sets the mode of the suite's environment's wall.
func (s *TriggerWallPassSuite) setWall(status string) {
	err := authwall.Store().SetAuthWallConfig(context.Background(), s.env.ID, &authwall.Config{Status: status})
	s.Require().NoError(err)
}

// run runs a trigger of the suite's environment against url and returns the
// headers it was sent with. A run that carries a token must not follow
// redirects, which is what signed says to expect.
func (s *TriggerWallPassSuite) run(url string, signed bool) http.Header {
	var sent http.Header

	if signed {
		s.mockRequest.On("FollowRedirects", false).Return(s.mockRequest).Once()
	}

	s.mockRequest.On("URL", url).Return(s.mockRequest).Once()
	s.mockRequest.On("Method", shttp.MethodGet).Return(s.mockRequest).Once()
	s.mockRequest.On("Headers", mock.MatchedBy(func(h http.Header) bool {
		sent = h
		return true
	})).Return(s.mockRequest).Once()
	s.mockRequest.On("Payload", []byte(nil)).Return(s.mockRequest).Once()
	s.mockRequest.On("Do").Return(&shttp.HTTPResponse{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     make(http.Header),
		},
	}, nil).Once()

	log, err := functiontrigger.Run(functiontrigger.RunParams{
		Context:   context.Background(),
		TriggerID: types.ID(7),
		EnvID:     s.env.ID,
		URL:       url,
		Headers:   shttp.Headers{"Authorization": "Bearer secret"},
	})

	s.Require().NoError(err)
	s.NotContains(log.Request["headers"], authwall.HeaderTriggerToken)
	s.mockRequest.AssertExpectations(s.T())

	return sent
}

// Test_OwnDomain_SendsToken verifies that a run against the environment's own
// domain carries a token that opens the environment's wall, and only its wall.
func (s *TriggerWallPassSuite) Test_OwnDomain_SendsToken() {
	sent := s.run("https://WWW.example.org/api/cron", true)
	token := sent.Get(authwall.HeaderTriggerToken)

	s.Equal("Bearer secret", sent.Get("Authorization"))
	s.Equal(types.ID(7), authwall.Token{EnvID: s.env.ID}.ParseTrigger(token))
	s.Equal(types.ID(0), authwall.Token{EnvID: s.env.ID + 1}.ParseTrigger(token))
}

// Test_OtherHosts_NoToken verifies that hosts the environment does not own, and
// its own hosts over plain HTTP, never receive the token, since it would let
// whoever reads it through the wall.
func (s *TriggerWallPassSuite) Test_OtherHosts_NoToken() {
	for _, url := range []string{
		"https://api.thirdparty.com/hook",
		"https://unverified.example.org/api/cron",
		"https://www.example.org.evil.com/",
		"http://www.example.org/api/cron",
	} {
		sent := s.run(url, false)

		s.Empty(sent.Get(authwall.HeaderTriggerToken), url)
		s.Equal("Bearer secret", sent.Get("Authorization"), url)
	}
}

// Test_WallOff_NoToken verifies that an environment without a wall runs its
// triggers exactly as before: no token, and redirects are followed.
func (s *TriggerWallPassSuite) Test_WallOff_NoToken() {
	s.setWall(authwall.StatusDisabled)

	sent := s.run("https://www.example.org/api/cron", false)

	s.Empty(sent.Get(authwall.HeaderTriggerToken))
}

func TestTriggerWallPassSuite(t *testing.T) {
	suite.Run(t, new(TriggerWallPassSuite))
}
