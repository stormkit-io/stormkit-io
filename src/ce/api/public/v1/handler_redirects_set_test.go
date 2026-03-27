package publicapiv1_test

import (
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerRedirectsSetSuite struct {
	suite.Suite
	*factory.Factory

	conn             databasetest.TestDB
	mockCacheService *mocks.CacheInterface
}

func (s *HandlerRedirectsSetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockCacheService = &mocks.CacheInterface{}
	appcache.DefaultCacheService = s.mockCacheService
}

func (s *HandlerRedirectsSetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	s.mockCacheService.AssertExpectations(s.T())
	appcache.DefaultCacheService = nil
}

func (s *HandlerRedirectsSetSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

func (s *HandlerRedirectsSetSuite) post(keyValue string, body any) shttptest.Response {
	return shttptest.RequestWithHeaders(s.handler(), shttp.MethodPost, "/v1/redirects", body,
		map[string]string{"Authorization": keyValue})
}

func (s *HandlerRedirectsSetSuite) TestSuccess() {
	reds := []redirects.Redirect{
		{From: "/path", To: "/new-path", Status: http.StatusFound},
		{From: "*", To: "/index.html"},
	}

	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, map[string]any{
		"Data": &buildconf.BuildConf{
			Redirects:     reds,
			RedirectsFile: "/my_file",
		},
	})

	key := s.MockAPIKey(nil, env)

	s.Equal(reds, env.Data.Redirects)
	s.Equal("main", env.Branch)
	s.Equal("/my_file", env.Data.RedirectsFile)

	s.mockCacheService.On("Reset", env.ID).Return(nil)

	response := s.post(key.Value, map[string]any{
		"redirects": []map[string]any{
			{"from": "/path", "to": "/new-path"},
		},
	})

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{"redirects": [{"from":"/path","to":"/new-path"}]}`, response.String())
}

// Test_InvalidRedirect_MissingFrom verifies that a redirect rule without a 'from' field is rejected.
func (s *HandlerRedirectsSetSuite) Test_InvalidRedirect_MissingFrom() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, nil)
	key := s.MockAPIKey(nil, env)

	response := s.post(key.Value, map[string]any{
		"redirects": []map[string]any{
			{"from": "", "to": "/new-path"},
		},
	})

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "'from' is required")
}

// Test_InvalidRedirect_MissingTo verifies that a redirect rule without a 'to' field is rejected.
func (s *HandlerRedirectsSetSuite) Test_InvalidRedirect_MissingTo() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, nil)
	key := s.MockAPIKey(nil, env)

	response := s.post(key.Value, map[string]any{
		"redirects": []map[string]any{
			{"from": "/old"},
		},
	})

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "'to' is required")
}

// Test_InvalidRedirect_BadStatus verifies that a redirect rule with an invalid HTTP status code is rejected.
func (s *HandlerRedirectsSetSuite) Test_InvalidRedirect_BadStatus() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app, nil)
	key := s.MockAPIKey(nil, env)

	response := s.post(key.Value, map[string]any{
		"redirects": []map[string]any{
			{"from": "/old", "to": "/new", "status": 999},
		},
	})

	s.Equal(http.StatusBadRequest, response.Code)
	s.Contains(response.String(), "999")
}

func TestHandlerRedirectsSet(t *testing.T) {
	suite.Run(t, &HandlerRedirectsSetSuite{})
}
