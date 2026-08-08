package publicapiv1_test

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/mocks"
)

// These tests hang off HandlerMCPSuite (defined in handler_mcp_test.go) so they
// reuse its helpers: post, userKey, rpcOK and toolContent.

func (s *HandlerMCPSuite) Test_CreateTrigger_AndList() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "create_trigger", map[string]any{
		"envId":  env.ID.String(),
		"cron":   "*/5 * * * *",
		"status": true,
		"method": "POST",
		"url":    "https://www.example.org/cron",
		"headers": map[string]any{
			"Content-Type": "application/json",
		},
	}))

	trigger := s.toolContent(s.rpcOK(resp))["trigger"].(map[string]any)
	s.NotEmpty(trigger["id"])

	stored, err := functiontrigger.NewStore().List(context.Background(), env.ID)
	s.Require().NoError(err)
	s.Len(stored, 1)
	s.Equal("https://www.example.org/cron", stored[0].Options.URL)
	s.Equal("Content-Type:application/json", stored[0].Options.Headers.String())

	listResp := s.post(key.Value, mcpToolCall(2, "list_triggers", map[string]any{
		"envId": env.ID.String(),
	}))

	triggers := s.toolContent(s.rpcOK(listResp))["triggers"].([]any)
	s.Len(triggers, 1)
}

func (s *HandlerMCPSuite) Test_CreateTrigger_InvalidCron() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "create_trigger", map[string]any{
		"envId": env.ID.String(),
		"cron":  "X * * * *",
		"url":   "https://www.example.org/cron",
	}))

	result := s.rpcOK(resp)["result"].(map[string]any)
	s.True(result["isError"].(bool))
}

func (s *HandlerMCPSuite) Test_UpdateTrigger() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	tf := s.MockTriggerFunction(env)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "update_trigger", map[string]any{
		"envId":     env.ID.String(),
		"triggerId": tf.ID.String(),
		"cron":      "5 5 * * *",
		"status":    true,
		"method":    "GET",
		"url":       "https://www.example.org/updated",
	}))

	s.rpcOK(resp)

	record, err := functiontrigger.NewStore().ByID(context.Background(), tf.ID)
	s.Require().NoError(err)
	s.Equal("5 5 * * *", record.Cron)
	s.Equal("https://www.example.org/updated", record.Options.URL)
}

func (s *HandlerMCPSuite) Test_Trigger_Documentation() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.userKey(usr)

	documentation := "Rebuilds the **search index**. Safe to disable."

	resp := s.post(key.Value, mcpToolCall(1, "create_trigger", map[string]any{
		"envId":         env.ID.String(),
		"cron":          "*/5 * * * *",
		"status":        true,
		"method":        "POST",
		"url":           "https://www.example.org/cron",
		"documentation": documentation,
	}))

	trigger := s.toolContent(s.rpcOK(resp))["trigger"].(map[string]any)
	s.Equal(documentation, trigger["documentation"])

	stored, err := functiontrigger.NewStore().List(context.Background(), env.ID)
	s.Require().NoError(err)
	s.Len(stored, 1)
	s.Equal(documentation, stored[0].Documentation)

	updateResp := s.post(key.Value, mcpToolCall(2, "update_trigger", map[string]any{
		"envId":         env.ID.String(),
		"triggerId":     stored[0].ID.String(),
		"cron":          "*/5 * * * *",
		"status":        true,
		"url":           "https://www.example.org/cron",
		"documentation": "Now documented differently.",
	}))

	s.rpcOK(updateResp)

	record, err := functiontrigger.NewStore().ByID(context.Background(), stored[0].ID)
	s.Require().NoError(err)
	s.Equal("Now documented differently.", record.Documentation)
	s.True(record.Status)
}

// Test_UpdateTrigger_Partial verifies that an update changes only the fields it
// carries: a caller changing the cron alone keeps the documentation, status,
// URL, headers and payload it never mentioned.
func (s *HandlerMCPSuite) Test_UpdateTrigger_Partial() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	key := s.userKey(usr)

	documentation := "Rebuilds the search index. Ping #data if it fails."

	s.rpcOK(s.post(key.Value, mcpToolCall(1, "create_trigger", map[string]any{
		"envId":         env.ID.String(),
		"cron":          "*/5 * * * *",
		"status":        true,
		"method":        "POST",
		"url":           "https://www.example.org/cron",
		"payload":       `{"full":true}`,
		"headers":       map[string]any{"authorization": "Bearer sk-1"},
		"documentation": documentation,
	})))

	stored, err := functiontrigger.NewStore().List(context.Background(), env.ID)
	s.Require().NoError(err)
	s.Require().Len(stored, 1)

	s.rpcOK(s.post(key.Value, mcpToolCall(2, "update_trigger", map[string]any{
		"envId":     env.ID.String(),
		"triggerId": stored[0].ID.String(),
		"cron":      "*/10 * * * *",
	})))

	record, err := functiontrigger.NewStore().ByID(context.Background(), stored[0].ID)
	s.Require().NoError(err)
	s.Equal("*/10 * * * *", record.Cron)
	s.Equal(documentation, record.Documentation)
	s.True(record.Status)
	s.Equal("POST", record.Options.Method)
	s.Equal("https://www.example.org/cron", record.Options.URL)
	s.Equal(`{"full":true}`, string(record.Options.Payload))
	s.Equal("Bearer sk-1", record.Options.Headers["authorization"])

	// An explicit empty value still clears a field.
	s.rpcOK(s.post(key.Value, mcpToolCall(3, "update_trigger", map[string]any{
		"envId":         env.ID.String(),
		"triggerId":     stored[0].ID.String(),
		"documentation": "",
		"status":        false,
	})))

	record, err = functiontrigger.NewStore().ByID(context.Background(), stored[0].ID)
	s.Require().NoError(err)
	s.Empty(record.Documentation)
	s.False(record.Status)
	s.Equal("*/10 * * * *", record.Cron)
}

func (s *HandlerMCPSuite) Test_DeleteTrigger() {
	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	tf := s.MockTriggerFunction(env)
	key := s.userKey(usr)

	resp := s.post(key.Value, mcpToolCall(1, "delete_trigger", map[string]any{
		"envId":     env.ID.String(),
		"triggerId": tf.ID.String(),
	}))

	s.rpcOK(resp)

	record, err := functiontrigger.NewStore().ByID(context.Background(), tf.ID)
	s.Require().NoError(err)
	s.Nil(record)
}

func (s *HandlerMCPSuite) Test_DeleteTrigger_Forbidden_NotMember() {
	owner := s.MockUser()
	other := s.MockUser()
	appl := s.MockApp(owner)
	env := s.MockEnv(appl)
	tf := s.MockTriggerFunction(env)
	key := s.userKey(other)

	resp := s.post(key.Value, mcpToolCall(1, "delete_trigger", map[string]any{
		"envId":     env.ID.String(),
		"triggerId": tf.ID.String(),
	}))

	result := s.rpcOK(resp)["result"].(map[string]any)
	s.True(result["isError"].(bool))

	record, err := functiontrigger.NewStore().ByID(context.Background(), tf.ID)
	s.Require().NoError(err)
	s.NotNil(record)
}

func (s *HandlerMCPSuite) Test_InvokeTrigger_AndLogs() {
	mockRequest := &mocks.RequestInterface{}
	shttp.DefaultRequest = mockRequest
	defer func() { shttp.DefaultRequest = nil }()

	usr := s.MockUser()
	appl := s.MockApp(usr)
	env := s.MockEnv(appl)
	tf := s.MockTriggerFunction(env, map[string]any{
		"Options": functiontrigger.Options{URL: "https://example.org"},
	})
	key := s.userKey(usr)

	mockRequest.On("URL", "https://example.org").Return(mockRequest).Once()
	mockRequest.On("Method", "GET").Return(mockRequest).Once()
	mockRequest.On("Headers", shttp.Headers(nil).Make()).Return(mockRequest).Once()
	mockRequest.On("Payload", []byte(nil)).Return(mockRequest).Once()
	mockRequest.On("Do").Return(&shttp.HTTPResponse{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("pong")),
			Header:     make(http.Header),
		},
	}, nil).Once()

	resp := s.post(key.Value, mcpToolCall(1, "invoke_trigger", map[string]any{
		"envId":     env.ID.String(),
		"triggerId": tf.ID.String(),
	}))

	s.rpcOK(resp)

	logsResp := s.post(key.Value, mcpToolCall(2, "get_trigger_logs", map[string]any{
		"envId":     env.ID.String(),
		"triggerId": tf.ID.String(),
	}))

	logs := s.toolContent(s.rpcOK(logsResp))["logs"].([]any)
	s.Len(logs, 1)
}
