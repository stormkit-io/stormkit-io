package adminhandlers_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/accesslog"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type HandlerAccessLogsSuite struct {
	suite.Suite
	*factory.Factory
	conn databasetest.TestDB
}

func (s *HandlerAccessLogsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
}

func (s *HandlerAccessLogsSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAccessLogsSuite) seed() {
	app := s.MockApp(nil)
	env := s.MockEnv(app)

	logs := []accesslog.AccessLog{
		{
			AppID:       app.ID,
			EnvID:       env.ID,
			HostName:    "example.com",
			RequestTS:   utils.NewUnix(),
			Method:      http.MethodGet,
			RequestPath: "/humans.txt",
			StatusCode:  http.StatusOK,
			ClientIP:    "203.0.113.7",
			UserAgent:   "Mozilla/5.0",
			IsBot:       false,
		},
		{
			AppID:       app.ID,
			EnvID:       env.ID,
			HostName:    "example.com",
			RequestTS:   utils.NewUnix(),
			Method:      http.MethodGet,
			RequestPath: "/robots.txt",
			StatusCode:  http.StatusOK,
			ClientIP:    "198.51.100.9",
			UserAgent:   "Googlebot/2.1",
			IsBot:       true,
		},
	}

	s.NoError(accesslog.NewStore().InsertLogs(context.Background(), logs))
}

func (s *HandlerAccessLogsSuite) request(token, query string) shttptest.Response {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(adminhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/admin/access-logs"+query,
		nil,
		map[string]string{"Authorization": token},
	)
}

func (s *HandlerAccessLogsSuite) Test_Admin_ReturnsLogs() {
	admin := s.MockUser(map[string]any{"IsAdmin": true})
	s.seed()

	res := s.request(usertest.Authorization(admin.ID), "")
	body := res.String()

	s.Equal(http.StatusOK, res.Code)
	s.Contains(body, "/humans.txt")
	s.Contains(body, "/robots.txt")
	// Raw client IP and user-agent are surfaced to the admin.
	s.Contains(body, "203.0.113.7")
	s.Contains(body, "Googlebot/2.1")
}

func (s *HandlerAccessLogsSuite) Test_Admin_FiltersBots() {
	admin := s.MockUser(map[string]any{"IsAdmin": true})
	s.seed()

	res := s.request(usertest.Authorization(admin.ID), "?isBot=false")
	body := res.String()

	s.Equal(http.StatusOK, res.Code)
	s.Contains(body, "/humans.txt")
	s.NotContains(body, "/robots.txt")
}

func (s *HandlerAccessLogsSuite) Test_NonAdmin_Rejected() {
	usr := s.MockUser(map[string]any{"IsAdmin": false})
	s.seed()

	res := s.request(usertest.Authorization(usr.ID), "")

	s.Equal(http.StatusUnauthorized, res.Code)
}

func TestHandlerAccessLogs(t *testing.T) {
	suite.Run(t, &HandlerAccessLogsSuite{})
}
