package functiontriggerhandlers_test

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger/functiontriggerhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type HandlerFunctionTriggerInvokeSuite struct {
	suite.Suite
	*factory.Factory

	conn        databasetest.TestDB
	mockRequest *mocks.RequestInterface
}

func (s *HandlerFunctionTriggerInvokeSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockRequest = &mocks.RequestInterface{}
	shttp.DefaultRequest = s.mockRequest
	admin.SetMockLicense()
}

func (s *HandlerFunctionTriggerInvokeSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	shttp.DefaultRequest = nil
	admin.ResetMockLicense()
}

func (s *HandlerFunctionTriggerInvokeSuite) Test_Invoke() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	tf := s.MockTriggerFunction(env, map[string]any{
		"Options": functiontrigger.Options{URL: "https://example.org"},
	})

	s.mockRequest.On("URL", "https://example.org").Return(s.mockRequest).Once()
	s.mockRequest.On("Method", "GET").Return(s.mockRequest).Once()
	s.mockRequest.On("Headers", shttp.Headers(nil).Make()).Return(s.mockRequest).Once()
	s.mockRequest.On("Payload", []byte(nil)).Return(s.mockRequest).Once()
	s.mockRequest.On("Do").Return(&shttp.HTTPResponse{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("pong")),
			Header:     make(http.Header),
		},
	}, nil).Once()

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(functiontriggerhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/apps/trigger/invoke",
		map[string]any{
			"id":    tf.ID.String(),
			"envId": env.ID.String(),
			"appId": app.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	logs, err := functiontrigger.NewStore().Logs(context.Background(), tf.ID)

	s.NoError(err)
	s.Len(logs, 1)
}

func (s *HandlerFunctionTriggerInvokeSuite) Test_Permission() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	app2 := s.MockApp(usr)
	env2 := s.MockEnv(app2)
	tf2 := s.MockTriggerFunction(env2)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(functiontriggerhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/apps/trigger/invoke",
		map[string]any{
			"id":    tf2.ID.String(),
			"envId": env.ID.String(),
			"appId": app.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusNotFound, response.Code)

	logs, err := functiontrigger.NewStore().Logs(context.Background(), tf2.ID)

	s.NoError(err)
	s.Len(logs, 0)
}

func TestHandlerInvokeTrigger(t *testing.T) {
	suite.Run(t, &HandlerFunctionTriggerInvokeSuite{})
}
