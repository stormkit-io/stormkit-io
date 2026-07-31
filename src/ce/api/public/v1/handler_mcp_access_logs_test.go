package publicapiv1_test

import (
	"context"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/accesslog"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// mockAccessLogs inserts a human and a bot request for the given environment.
func (s *HandlerMCPSuite) mockAccessLogs(env *factory.MockEnv) {
	logs := []accesslog.AccessLog{
		{
			AppID:       env.AppID,
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
			AppID:       env.AppID,
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

	s.Require().NoError(accesslog.NewStore().InsertLogs(context.Background(), logs))
}

// accessLogPaths pulls the request paths out of a get_access_logs tool result.
func (s *HandlerMCPSuite) accessLogPaths(data map[string]any) []string {
	entries, ok := data["accessLogs"].([]any)
	s.Require().True(ok)

	paths := make([]string, 0, len(entries))

	for _, e := range entries {
		paths = append(paths, e.(map[string]any)["path"].(string))
	}

	return paths
}

func (s *HandlerMCPSuite) Test_GetAccessLogs_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	s.mockAccessLogs(mockEnv)

	resp := s.post(key.Value, mcpToolCall(1, "get_access_logs", map[string]any{
		"envId": mockEnv.ID.String(),
	}))

	data := s.toolContent(s.rpcOK(resp))

	s.ElementsMatch([]string{"/humans.txt", "/robots.txt"}, s.accessLogPaths(data))
	s.Equal(false, data["pagination"].(map[string]any)["hasNextPage"])
}

func (s *HandlerMCPSuite) Test_GetAccessLogs_FiltersBots() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	s.mockAccessLogs(mockEnv)

	resp := s.post(key.Value, mcpToolCall(1, "get_access_logs", map[string]any{
		"envId": mockEnv.ID.String(),
		"isBot": false,
	}))

	data := s.toolContent(s.rpcOK(resp))

	s.Equal([]string{"/humans.txt"}, s.accessLogPaths(data))
}

func (s *HandlerMCPSuite) Test_GetAccessLogs_MissingEnvId() {
	usr := s.MockUser()
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "get_access_logs", map[string]any{}))

	env := s.rpcOK(resp)
	s.True(env["result"].(map[string]any)["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_GetAccessLogs_Forbidden_NotMember() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	appl := s.MockApp(usr1)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr2)

	s.mockAccessLogs(mockEnv)

	resp := s.post(key.Value, mcpToolCall(1, "get_access_logs", map[string]any{
		"envId": mockEnv.ID.String(),
	}))

	env := s.rpcOK(resp)
	s.True(env["result"].(map[string]any)["isError"].(bool))
}
