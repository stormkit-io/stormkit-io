package hosting_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/hosting"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stretchr/testify/suite"
)

type HandlersSuite struct {
	suite.Suite
	conn databasetest.TestDB
}

func (s *HandlersSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
}

func (s *HandlersSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlersSuite) Test_InternalHandlers() {
	cnf := admin.MustConfig().DomainConfig
	cnf.Health = "http://health.example.org:8443"
	cnf.API = "http://api.example.org:8443"
	cnf.App = "http://stormkit.example.org:8443"

	handler := hosting.InternalHandlers(hosting.InternalHandlerOpts{
		Health: true,
		API:    true,
		App:    true,
	})

	custom := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		w.Write([]byte("Hello World"))
	})

	// For this case, we expect the custom handler to be called
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)

	handler(custom).ServeHTTP(w, r)
	s.Equal(http.StatusTeapot, w.Code)
	s.Equal("Hello World", w.Body.String())

	// For this case, we expect the stormkit ui handler (/stormkit/ui/index.html is not found so it returns 404)
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/auth", nil)
	r.Host = "stormkit.example.org"

	handler(custom).ServeHTTP(w, r)
	s.Equal(http.StatusNotFound, w.Code)
	s.Equal("", w.Body.String())
}

func TestHandlerSuite(t *testing.T) {
	suite.Run(t, new(HandlersSuite))
}
