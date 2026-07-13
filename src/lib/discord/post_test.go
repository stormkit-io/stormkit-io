package discord_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/discord"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type DiscordSuite struct {
	suite.Suite
	mockRequest *mocks.RequestInterface
}

func (s *DiscordSuite) BeforeTest(_, _ string) {
	s.mockRequest = &mocks.RequestInterface{}
	shttp.DefaultRequest = s.mockRequest
}

func (s *DiscordSuite) AfterTest(_, _ string) {
	shttp.DefaultRequest = nil
}

func (s *DiscordSuite) TestPost() {
	webhook := "http://localhost/my-webhook"
	headers := make(http.Header)
	headers.Add("Content-Type", "application/json")

	payload := discord.Payload{
		Embeds: []discord.PayloadEmbed{
			{
				Title:     "New Signup",
				Timestamp: time.Date(2021, time.Month(9), 23, 9, 20, 10, 0, time.UTC).Format(time.RFC3339),
				Fields: []discord.PayloadField{
					{Name: "ID", Value: "1536"},
					{Name: "Name", Value: "John Doe"},
					{Name: "Email", Value: "john@doe.com"},
				}},
		},
	}

	s.mockRequest.On("URL", webhook).Return(s.mockRequest)
	s.mockRequest.On("Method", shttp.MethodPost).Return(s.mockRequest)
	s.mockRequest.On("Headers", headers).Return(s.mockRequest)
	s.mockRequest.On("Payload", payload).Return(s.mockRequest)
	s.mockRequest.On("Do").Return(nil, nil)

	discord.Notify(webhook, payload)
}

func TestDiscordSuite(t *testing.T) {
	suite.Run(t, &DiscordSuite{})
}
