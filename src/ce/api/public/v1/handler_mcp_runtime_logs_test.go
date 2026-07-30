package publicapiv1_test

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/ce/api/applog"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
)

// mockRuntimeLogs inserts two runtime log entries for the given deployment.
func (s *HandlerMCPSuite) mockRuntimeLogs(env *factory.MockEnv, depl *factory.MockDeployment) []*applog.Log {
	logs := []*applog.Log{
		{
			AppID:         env.AppID,
			DeploymentID:  depl.ID,
			EnvironmentID: env.ID,
			Label:         "info",
			Data:          "first log line",
			Timestamp:     1665379845,
		},
		{
			AppID:         env.AppID,
			DeploymentID:  depl.ID,
			EnvironmentID: env.ID,
			Label:         "info",
			Data:          "second log line",
			Timestamp:     1696919445,
		},
	}

	s.Require().NoError(applog.NewStore().InsertLogs(context.Background(), logs))

	return logs
}

func (s *HandlerMCPSuite) Test_GetRuntimeLogs_Success() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	depl := s.MockDeployment(mockEnv)
	key := s.userKey(usr)

	s.mockRuntimeLogs(mockEnv, depl)

	resp := s.post(key.Value, mcpToolCall(1, "get_runtime_logs", map[string]any{
		"envId":        mockEnv.ID.String(),
		"deploymentId": depl.ID.String(),
	}))

	env := s.rpcOK(resp)
	data := s.toolContent(env)

	logs, ok := data["logs"].([]any)
	s.True(ok)
	s.Len(logs, 2)
	s.Equal("first log line", logs[0].(map[string]any)["data"])
	s.Equal(depl.ID.String(), logs[0].(map[string]any)["deploymentId"])
	s.Equal(false, data["hasNextPage"])
}

func (s *HandlerMCPSuite) Test_GetRuntimeLogs_Sort() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	depl := s.MockDeployment(mockEnv)
	key := s.userKey(usr)

	s.mockRuntimeLogs(mockEnv, depl)

	resp := s.post(key.Value, mcpToolCall(1, "get_runtime_logs", map[string]any{
		"envId":        mockEnv.ID.String(),
		"deploymentId": depl.ID.String(),
		"sort":         "desc",
	}))

	env := s.rpcOK(resp)
	logs := s.toolContent(env)["logs"].([]any)

	s.Len(logs, 2)
	s.Equal("second log line", logs[0].(map[string]any)["data"])
}

func (s *HandlerMCPSuite) Test_GetRuntimeLogs_InvalidSort() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	depl := s.MockDeployment(mockEnv)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "get_runtime_logs", map[string]any{
		"envId":        mockEnv.ID.String(),
		"deploymentId": depl.ID.String(),
		"sort":         "sideways",
	}))

	env := s.rpcOK(resp)
	s.True(env["result"].(map[string]any)["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_GetRuntimeLogs_MissingDeploymentId() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	mockEnv := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "get_runtime_logs", map[string]any{
		"envId": mockEnv.ID.String(),
	}))

	env := s.rpcOK(resp)
	s.True(env["result"].(map[string]any)["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_GetRuntimeLogs_Forbidden_NotMember() {
	usr1 := s.MockUser()
	usr2 := s.MockUser()
	appl := s.MockApp(usr1)
	mockEnv := s.MockEnv(appl)
	depl := s.MockDeployment(mockEnv)
	key := s.userKey(usr2)

	s.mockRuntimeLogs(mockEnv, depl)

	resp := s.post(key.Value, mcpToolCall(1, "get_runtime_logs", map[string]any{
		"envId":        mockEnv.ID.String(),
		"deploymentId": depl.ID.String(),
	}))

	env := s.rpcOK(resp)
	s.True(env["result"].(map[string]any)["isError"].(bool))
}
