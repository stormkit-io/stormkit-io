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
	"github.com/stormkit-io/stormkit-io/src/lib/types"
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
			ID:          tf.ID,
			URL:         tf.Options.URL,
			Payload:     tf.Options.Payload,
			Headers:     tf.Options.Headers,
			Method:      tf.Options.Method,
			NextRunAt:   utils.UnixFrom(nextRunAt),
			ScheduledAt: tf.NextRunAt,
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

type expectTargetParams struct {
	URL         string
	Method      string
	ContentType string
	Payload     []byte
	Response    *shttp.HTTPResponse
	Err         error
}

// expectTarget wires the request mock for one of generateMockMessage's targets.
func (s *JobTriggerFunctionsSuite) expectTarget(p expectTargetParams) {
	s.mockRequest.On("URL", p.URL).Return(s.mockRequest).Once()
	s.mockRequest.On("Method", p.Method).Return(s.mockRequest).Once()
	s.mockRequest.On("Headers", shttp.HeadersFromMap(map[string]string{"content-type": p.ContentType})).Return(s.mockRequest).Once()
	s.mockRequest.On("Payload", p.Payload).Return(s.mockRequest).Once()
	s.mockRequest.On("Do").Return(p.Response, p.Err).Once()
}

func unavailableResponse() *shttp.HTTPResponse {
	return &shttp.HTTPResponse{
		Response: &http.Response{
			StatusCode: http.StatusServiceUnavailable,
			Body:       io.NopCloser(strings.NewReader("Service not yet started, retry in a bit.")),
			Header:     make(http.Header),
		},
	}
}

func okResponse() *shttp.HTTPResponse {
	return &shttp.HTTPResponse{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("ok")),
			Header:     make(http.Header),
		},
	}
}

// Test_HandleFunctionTrigger_RetryWindow covers what a run that never happened
// does to nextRunAt. The fixture's first trigger is ten minutes overdue, past
// the five minute retry window, and the second two minutes, still inside it, so
// every case asserts both the "give up" and the "stay due" side of the bound.
func (s *JobTriggerFunctionsSuite) Test_HandleFunctionTrigger_RetryWindow() {
	type target struct {
		response *shttp.HTTPResponse
		err      error
		advances bool
	}

	cases := []struct {
		name   string
		first  target
		second target
	}{
		{
			name:   "a 503 stays due while a sibling that ran advances",
			first:  target{response: okResponse(), advances: true},
			second: target{response: unavailableResponse()},
		},
		{
			name:   "a 503 past the window advances, one inside it stays due",
			first:  target{response: unavailableResponse(), advances: true},
			second: target{response: unavailableResponse()},
		},
		{
			name:   "a transport error is bounded by the same window",
			first:  target{err: errors.New("connection refused"), advances: true},
			second: target{err: errors.New("connection refused")},
		},
	}

	for _, c := range cases {
		s.Run(c.name, func() {
			s.mockRequest = &mocks.RequestInterface{}
			shttp.DefaultRequest = s.mockRequest

			payload := s.generateMockMessage()
			task := asynq.NewTask("", payload)

			messages := []jobs.FunctionTriggerMessage{}
			s.NoError(json.Unmarshal(payload, &messages))
			s.Require().Len(messages, 2)

			store := functiontrigger.NewStore()
			ctx := context.Background()

			before := map[types.ID]int64{}

			for _, m := range messages {
				tf, err := store.ByID(ctx, m.ID)
				s.Require().NoError(err)
				before[m.ID] = tf.NextRunAt.Unix()
			}

			s.expectTarget(expectTargetParams{
				URL:         "https://example-1.org",
				Method:      "GET",
				ContentType: "application/json",
				Response:    c.first.response,
				Err:         c.first.err,
			})

			s.expectTarget(expectTargetParams{
				URL:         "https://example-2.org",
				Method:      "PATCH",
				ContentType: "text/html",
				Payload:     []byte("Hello World!"),
				Response:    c.second.response,
				Err:         c.second.err,
			})

			s.NoError(jobs.HandleFunctionTrigger(ctx, task))

			for i, want := range []bool{c.first.advances, c.second.advances} {
				tf, err := store.ByID(ctx, messages[i].ID)
				s.Require().NoError(err)

				if want {
					s.Equal(messages[i].NextRunAt.Unix(), tf.NextRunAt.Unix(), "expected the trigger to advance to its next tick")
				} else {
					s.Equal(before[messages[i].ID], tf.NextRunAt.Unix(), "expected the trigger to stay due for the next sweep")
				}
			}
		})
	}
}

func TestJobTriggerFunctionSuite(t *testing.T) {
	suite.Run(t, &JobTriggerFunctionsSuite{})
}
