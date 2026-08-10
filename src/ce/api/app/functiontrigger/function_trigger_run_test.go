package functiontrigger_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type TriggerFunctionRunSuite struct {
	suite.Suite
	mockRequest *mocks.RequestInterface
}

func (s *TriggerFunctionRunSuite) BeforeTest(_, _ string) {
	s.mockRequest = &mocks.RequestInterface{}
	shttp.DefaultRequest = s.mockRequest
}

func (s *TriggerFunctionRunSuite) AfterTest(_, _ string) {
	shttp.DefaultRequest = nil
}

func (s *TriggerFunctionRunSuite) run(body string) functiontrigger.TriggerLog {
	s.mockRequest.On("URL", "https://example.org").Return(s.mockRequest).Once()
	s.mockRequest.On("Method", shttp.MethodGet).Return(s.mockRequest).Once()
	s.mockRequest.On("Headers", shttp.HeadersFromMap(map[string]string{})).Return(s.mockRequest).Once()
	s.mockRequest.On("Payload", []byte(nil)).Return(s.mockRequest).Once()
	s.mockRequest.On("Do").Return(&shttp.HTTPResponse{
		Response: &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     make(http.Header),
		},
	}, nil).Once()

	log, err := functiontrigger.Run(functiontrigger.RunParams{URL: "https://example.org"})
	s.NoError(err)

	return log
}

func (s *TriggerFunctionRunSuite) Test_Run_KeepsSmallBodyIntact() {
	log := s.run("hello")

	s.Equal("hello", log.Response["body"])
}

// A trigger's target is an arbitrary user-supplied host and an unavailable one
// produces several log rows per tick, so the stored body must be bounded.
func (s *TriggerFunctionRunSuite) Test_Run_TruncatesLargeBody() {
	log := s.run(strings.Repeat("a", 128*1024))

	body, ok := log.Response["body"].(string)
	s.Require().True(ok)
	s.Equal(strings.Repeat("a", 64*1024)+"... (truncated)", body)
}

func TestTriggerFunctionRunSuite(t *testing.T) {
	suite.Run(t, &TriggerFunctionRunSuite{})
}
