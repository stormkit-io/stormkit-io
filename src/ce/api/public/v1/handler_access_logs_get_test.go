package publicapiv1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/accesslog"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
)

type HandlerAccessLogsGetSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
	user *factory.MockUser
	app  *factory.MockApp
	env  *factory.MockEnv
}

func (s *HandlerAccessLogsGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	s.user = s.MockUser()
	s.app = s.MockApp(s.user)
	s.env = s.MockEnv(s.app)

	logs := []accesslog.AccessLog{
		{
			AppID:       s.app.ID,
			EnvID:       s.env.ID,
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
			AppID:       s.app.ID,
			EnvID:       s.env.ID,
			HostName:    "example.com",
			RequestTS:   utils.NewUnix(),
			Method:      http.MethodPost,
			RequestPath: "/robots.txt",
			StatusCode:  http.StatusNotFound,
			ClientIP:    "198.51.100.9",
			UserAgent:   "Googlebot/2.1",
			IsBot:       true,
		},
	}

	s.Require().NoError(accesslog.NewStore().InsertLogs(context.Background(), logs))
}

func (s *HandlerAccessLogsGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
}

func (s *HandlerAccessLogsGetSuite) request(query, auth string) shttptest.Response {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/v1/access-logs?envId=%s%s", s.env.ID.String(), query),
		nil,
		map[string]string{"Authorization": auth},
	)
}

// parse returns the request paths of the returned access logs, plus the
// pagination object.
func (s *HandlerAccessLogsGetSuite) parse(raw string) ([]string, map[string]any) {
	var body struct {
		AccessLogs []map[string]any `json:"accessLogs"`
		Pagination map[string]any   `json:"pagination"`
	}

	s.Require().NoError(json.Unmarshal([]byte(raw), &body))

	paths := make([]string, 0, len(body.AccessLogs))

	for _, l := range body.AccessLogs {
		paths = append(paths, l["path"].(string))
	}

	return paths, body.Pagination
}

func (s *HandlerAccessLogsGetSuite) paths(res shttptest.Response) ([]string, map[string]any) {
	return s.parse(res.String())
}

func (s *HandlerAccessLogsGetSuite) Test_Success() {
	res := s.request("", usertest.Authorization(s.user.ID))
	body := res.String()

	s.Equal(http.StatusOK, res.Code)

	paths, pagination := s.parse(body)

	s.ElementsMatch([]string{"/humans.txt", "/robots.txt"}, paths)
	s.Equal(map[string]any{"hasNextPage": false}, pagination)
	// Raw client IP and user-agent are surfaced to the app owner.
	s.Contains(body, "203.0.113.7")
	s.Contains(body, "Googlebot/2.1")
}

func (s *HandlerAccessLogsGetSuite) Test_Success_Filters() {
	for _, tc := range []struct {
		query    string
		expected []string
	}{
		{"&isBot=false", []string{"/humans.txt"}},
		{"&isBot=true", []string{"/robots.txt"}},
		{"&method=POST", []string{"/robots.txt"}},
		{"&status=404", []string{"/robots.txt"}},
		{"&clientIp=203.0.113.7", []string{"/humans.txt"}},
		{"&path=/humans", []string{"/humans.txt"}},
		{"&hostName=example.com", []string{"/humans.txt", "/robots.txt"}},
		{"&hostName=other.com", []string{}},
	} {
		res := s.request(tc.query, usertest.Authorization(s.user.ID))
		paths, _ := s.paths(res)

		s.Equal(http.StatusOK, res.Code, tc.query)
		s.ElementsMatch(tc.expected, paths, tc.query)
	}
}

// Entries older than the default window are only returned when `from` is given.
func (s *HandlerAccessLogsGetSuite) Test_Success_DefaultWindow() {
	old := accesslog.AccessLog{
		AppID:       s.app.ID,
		EnvID:       s.env.ID,
		HostName:    "example.com",
		RequestTS:   utils.UnixFrom(time.Now().Add(-48 * time.Hour)),
		Method:      http.MethodGet,
		RequestPath: "/ancient.txt",
		StatusCode:  http.StatusOK,
	}

	s.Require().NoError(accesslog.NewStore().InsertLogs(context.Background(), []accesslog.AccessLog{old}))

	res := s.request("", usertest.Authorization(s.user.ID))
	paths, _ := s.paths(res)

	s.NotContains(paths, "/ancient.txt")

	from := time.Now().Add(-72 * time.Hour).Unix()
	res = s.request(fmt.Sprintf("&from=%d", from), usertest.Authorization(s.user.ID))
	paths, _ = s.paths(res)

	s.Contains(paths, "/ancient.txt")
}

func (s *HandlerAccessLogsGetSuite) Test_Success_HasNextPage() {
	res := s.request("&limit=1", usertest.Authorization(s.user.ID))
	paths, pagination := s.paths(res)

	s.Equal(http.StatusOK, res.Code)
	s.Len(paths, 1)
	s.Equal(true, pagination["hasNextPage"])

	// A single opaque cursor carries both halves of the sort key, so the next
	// page is a keyset seek on (request_timestamp, log_id) rather than a scan.
	s.NotEmpty(pagination["cursor"])

	// The cursor marks the last returned entry; paging with it returns the rest.
	res = s.request(
		fmt.Sprintf("&limit=1&cursor=%s", pagination["cursor"]),
		usertest.Authorization(s.user.ID),
	)

	next, _ := s.paths(res)

	s.Len(next, 1)
	s.NotEqual(paths[0], next[0])
}

// Logs are scoped to the environment the API key authorizes, not to the app.
func (s *HandlerAccessLogsGetSuite) Test_Success_ScopedToEnvironment() {
	otherEnv := s.MockEnv(s.app, map[string]any{"Name": "staging"})

	s.Require().NoError(accesslog.NewStore().InsertLogs(context.Background(), []accesslog.AccessLog{{
		AppID:       s.app.ID,
		EnvID:       otherEnv.ID,
		HostName:    "staging.example.com",
		RequestTS:   utils.NewUnix(),
		Method:      http.MethodGet,
		RequestPath: "/staging-only.txt",
		StatusCode:  http.StatusOK,
	}}))

	res := s.request("", usertest.Authorization(s.user.ID))
	paths, _ := s.paths(res)

	s.ElementsMatch([]string{"/humans.txt", "/robots.txt"}, paths)
}

// A caller cannot widen the scope past the environment its key authorizes.
func (s *HandlerAccessLogsGetSuite) Test_Success_IgnoresAppIdQueryParam() {
	otherApp := s.MockApp(s.user)
	otherEnv := s.MockEnv(otherApp)

	s.Require().NoError(accesslog.NewStore().InsertLogs(context.Background(), []accesslog.AccessLog{{
		AppID:       otherApp.ID,
		EnvID:       otherEnv.ID,
		HostName:    "other.example.com",
		RequestTS:   utils.NewUnix(),
		Method:      http.MethodGet,
		RequestPath: "/other-app.txt",
		StatusCode:  http.StatusOK,
	}}))

	res := s.request(
		fmt.Sprintf("&appId=%s", otherApp.ID.String()),
		usertest.Authorization(s.user.ID),
	)
	paths, _ := s.paths(res)

	s.ElementsMatch([]string{"/humans.txt", "/robots.txt"}, paths)
}

func (s *HandlerAccessLogsGetSuite) Test_Forbidden_NotMember() {
	other := s.MockUser()

	res := s.request("", usertest.Authorization(other.ID))

	s.Equal(http.StatusNotFound, res.Code)
}

func TestHandlerAccessLogsGet(t *testing.T) {
	suite.Run(t, &HandlerAccessLogsGetSuite{})
}
