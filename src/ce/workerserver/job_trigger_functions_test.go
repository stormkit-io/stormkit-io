package jobs_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/adhocore/gronx"
	"github.com/hibiken/asynq"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	jobs "github.com/stormkit-io/stormkit-io/src/ce/workerserver"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/tasks"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

type JobTriggerFunctionsSuite struct {
	suite.Suite
	*factory.Factory
	conn           databasetest.TestDB
	mockClient     mocks.TaskClient
	originalClient func() tasks.TaskClient
	mockRequest    *mocks.RequestInterface
}

func (s *JobTriggerFunctionsSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockClient = mocks.TaskClient{}
	s.originalClient = tasks.Client
	s.mockRequest = &mocks.RequestInterface{}
	shttp.DefaultRequest = s.mockRequest
	tasks.Client = func() tasks.TaskClient {
		return &s.mockClient
	}
}

func (s *JobTriggerFunctionsSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	tasks.Client = s.originalClient
	shttp.DefaultRequest = nil
}

func (s *JobTriggerFunctionsSuite) generateMockMessage() []byte {
	currentTime := time.Now().UTC()
	tenMinutesAgo := currentTime.Add(-time.Minute * 10)
	twoMinutesAgo := currentTime.Add(-time.Minute * 2)
	inTwoHours := currentTime.Add(time.Hour * 2)

	t1 := utils.UnixFrom(tenMinutesAgo)
	t2 := utils.UnixFrom(twoMinutesAgo)
	t3 := utils.UnixFrom(inTwoHours)

	tf1 := s.MockTriggerFunction(nil, map[string]any{
		"NextRunAt": t1,
		"Options": functiontrigger.Options{
			URL: "https://example-1.org",
			Headers: shttp.Headers{
				"content-type": "application/json",
			},
		},
	})

	tf2 := s.MockTriggerFunction(nil, map[string]any{
		"NextRunAt": t2,
		"Options": functiontrigger.Options{
			URL:     "https://example-2.org",
			Method:  "PATCH",
			Payload: []byte("Hello World!"),
			Headers: shttp.Headers{
				"content-type": "text/html",
			},
		},
	})

	// This should not be included
	s.MockTriggerFunction(nil, map[string]any{"NextRunAt": t3})

	messages := []jobs.FunctionTriggerMessage{}
	triggers := []*factory.MockFunctionTrigger{tf1, tf2}

	for _, tf := range triggers {
		nextRunAt, err := gronx.NextTickAfter(tf.Cron, time.Now().UTC(), false)
		s.NoError(err)

		messages = append(messages, jobs.FunctionTriggerMessage{
			ID:        tf.ID,
			URL:       tf.Options.URL,
			Payload:   tf.Options.Payload,
			Headers:   tf.Options.Headers,
			Method:    tf.Options.Method,
			NextRunAt: utils.UnixFrom(nextRunAt),
		})
	}

	payload, err := json.Marshal(messages)
	s.NoError(err)
	return payload
}

func (s *JobTriggerFunctionsSuite) Test_CreatingIndividualTasks() {
	payload := s.generateMockMessage()

	s.mockClient.On("Enqueue", mock.Anything).Return(nil, nil)

	s.NoError(jobs.InvokeDueFunctionTriggers(context.Background()))

	s.mockClient.AssertCalled(s.T(), "Enqueue", mock.MatchedBy(func(task *asynq.Task) bool {
		return s.Equal(task.Payload(), payload) && s.Equal(task.Type(), tasks.TriggerFunctionHttp)
	}))
}

func (s *JobTriggerFunctionsSuite) Test_ConsumingTasks() {
	t := asynq.NewTask("", s.generateMockMessage())

	// First request
	s.mockRequest.On("URL", "https://example-1.org").Return(s.mockRequest).Once()
	s.mockRequest.On("Method", "GET").Return(s.mockRequest).Once()
	s.mockRequest.On("Headers", shttp.HeadersFromMap(map[string]string{"content-type": "application/json"})).Return(s.mockRequest).Once()
	s.mockRequest.On("Payload", []byte(nil)).Return(s.mockRequest).Once()
	s.mockRequest.On("Do").Return(&shttp.HTTPResponse{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("my-response-1")),
			Header:     make(http.Header),
		},
	}, nil).Once()

	// Second request
	s.mockRequest.On("URL", "https://example-2.org").Return(s.mockRequest).Once()
	s.mockRequest.On("Method", "PATCH").Return(s.mockRequest).Once()
	s.mockRequest.On("Headers", shttp.HeadersFromMap(map[string]string{"content-type": "text/html"})).Return(s.mockRequest).Once()
	s.mockRequest.On("Payload", []byte("Hello World!")).Return(s.mockRequest).Once()
	s.mockRequest.On("Do").Return(&shttp.HTTPResponse{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("my-response-2")),
			Header:     make(http.Header),
		},
	}, nil).Once()

	s.NoError(jobs.HandleFunctionTrigger(context.Background(), t))
}

func (s *JobTriggerFunctionsSuite) Test_HandleFunctionTrigger_PartialFailure() {
	// Reuse existing generator (two messages, both due)
	payload := s.generateMockMessage()
	task := asynq.NewTask("", payload)

	// First request succeeds
	s.mockRequest.On("URL", "https://example-1.org").Return(s.mockRequest).Once()
	s.mockRequest.On("Method", "GET").Return(s.mockRequest).Once()
	s.mockRequest.On("Headers", shttp.HeadersFromMap(map[string]string{"content-type": "application/json"})).Return(s.mockRequest).Once()
	s.mockRequest.On("Payload", []byte(nil)).Return(s.mockRequest).Once()
	s.mockRequest.On("Do").Return(&shttp.HTTPResponse{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok-1")),
			Header:     make(http.Header),
		},
	}, nil).Once()

	// Second request fails (network / client error simulation)
	s.mockRequest.On("URL", "https://example-2.org").Return(s.mockRequest).Once()
	s.mockRequest.On("Method", "PATCH").Return(s.mockRequest).Once()
	s.mockRequest.On("Headers", shttp.HeadersFromMap(map[string]string{"content-type": "text/html"})).Return(s.mockRequest).Once()
	s.mockRequest.On("Payload", []byte("Hello World!")).Return(s.mockRequest).Once()
	s.mockRequest.On("Do").Return(nil, errors.New("boom")).Once()

	s.NoError(jobs.HandleFunctionTrigger(context.Background(), task))

	// We only assert no panic/error; deeper DB assertions can be added if store getters are available.
	// Ensures code handles mixed success/failure gracefully.
}

func TestJobTriggerFunctionSuite(t *testing.T) {
	suite.Run(t, &JobTriggerFunctionsSuite{})
}
