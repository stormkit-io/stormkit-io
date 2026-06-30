package router_test

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/router"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

type CorsSuite struct {
	suite.Suite

	ts   *httptest.Server
	r    *shttp.Router
	conn databasetest.TestDB
}

func (s *CorsSuite) SetupSuite() {
	s.r = shttp.NewRouter()
	s.r.RegisterMiddleware(router.WithCors)

	s.ts = httptest.NewServer(s.r.Handler())
	router.AllowedHosts = []string{s.ts.URL}
}

func (s *CorsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
}

func (s *CorsSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *CorsSuite) TearDownSuite() {
	s.ts.Close()
	router.AllowedHosts = []string{}
}

func (s *CorsSuite) Test_Allowed() {
	response := shttptest.RequestWithHeaders(
		s.r.Handler(),
		shttp.MethodOptions,
		"/app/deploy/callback",
		map[string]any{
			"deploymentId": 5920102,
			"token":        "vKlz42mapqCxaxbnL",
		},
		map[string]string{
			"Origin": s.ts.URL,
		},
	)

	allowedOrigin := response.Header().Get("Access-Control-Allow-Origin")
	allowedMethod := response.Header().Get("Access-Control-Allow-Methods")
	allowedHeader := response.Header().Get("Access-Control-Allow-Headers")
	connection := response.Header().Get("Connection")
	maxAge := response.Header().Get("Access-Control-Max-Age")

	s.Equal(http.StatusOK, response.Code)
	s.Equal(s.ts.URL, allowedOrigin)
	s.Equal(allowedMethod, strings.Join(router.AllowedMethods, ","))
	s.Equal(allowedHeader, strings.Join(router.AllowedHeaders, ","))
	s.Equal("86400", maxAge)
	s.Equal("keep-alive", connection)
}

func (s *CorsSuite) Test_Allowed_SelfHosted() {
	router.AllowedHosts = []string{}
	config.SetIsSelfHosted(true)
	admin.MustConfig().SetURL("https://example.org")
	allowed := router.Cors()
	s.Equal([]string{"^https://stormkit\\.example\\.org$"}, allowed)

	match, err := regexp.MatchString(allowed[0], "https://stormkit.example.org")
	s.True(match)
	s.NoError(err)

	match, err = regexp.MatchString(allowed[0], "https://stormkit--194171.example.org")
	s.False(match)
	s.NoError(err)
}

func (s *CorsSuite) Test_Allowed_WithPort() {
	router.AllowedHosts = []string{}
	{
		cnf := admin.MustConfig().Clone()
		cnf.DomainConfig.App = "https://sk.example.org:8443"
		admin.SetConfig(&cnf)
	}
	config.SetIsSelfHosted(true)

	allowed := router.Cors()
	s.Equal([]string{"^https://sk\\.example\\.org:8443$"}, allowed)

	match, err := regexp.MatchString(allowed[0], "https://sk.example.org:8443")
	s.True(match)
	s.NoError(err)

	match, err = regexp.MatchString(allowed[0], "https://another.example.org:8443")
	s.False(match)
	s.NoError(err)
}

func (s *CorsSuite) Test_NotAllowed() {
	response := shttptest.RequestWithHeaders(
		s.r.Handler(),
		shttp.MethodOptions,
		"/app/deploy/callback",
		map[string]interface{}{
			"deploymentId": 5920102,
			"token":        "vKlz42mapqCxaxbnL",
		},
		map[string]string{
			"Origin": "something-else",
		},
	)

	allowedOrigin := response.Header().Get("Access-Control-Allow-Origin")

	s.Equal(http.StatusNotFound, response.Code)
	s.Equal("", allowedOrigin)
}

func (s *CorsSuite) Test_AllowedOtherMethods() {
	response := shttptest.RequestWithHeaders(
		s.r.Handler(),
		shttp.MethodGet,
		"/not-found-url",
		nil,
		map[string]string{
			"Origin": s.ts.URL,
		},
	)

	allowedOrigin := response.Header().Get("Access-Control-Allow-Origin")

	s.Equal(http.StatusNotFound, response.Code)
	s.Equal(s.ts.URL, allowedOrigin)
}

func (s *CorsSuite) Test_ResetCors() {
	config.SetIsSelfHosted(true)
	{
		cnf := admin.MustConfig().Clone()
		cnf.DomainConfig.App = "https://sk.new-domain.org"
		admin.SetConfig(&cnf)
	}

	router.AllowedHosts = []string{"http://old-host"}
	router.ResetCors()

	s.Equal([]string{"^https://sk\\.new-domain\\.org$"}, router.AllowedHosts)
}

func TestCorsSuite(t *testing.T) {
	suite.Run(t, &CorsSuite{})
}
