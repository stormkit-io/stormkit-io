package shttp_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

// RouterJSONErrorsSuite covers the responses an API client gets when it asks
// for something that is not there. An agent cannot act on "404 page not found"
// written as text/plain, so unmatched routes answer with the same JSON shape as
// the handlers.
type RouterJSONErrorsSuite struct {
	suite.Suite
}

func (s *RouterJSONErrorsSuite) handler() http.Handler {
	router := shttp.NewRouter()
	router.WithJSONErrors()

	router.NewService().
		NewEndpoint("/v1/ping").
		Handler(shttp.MethodGet, "", func(*shttp.RequestContext) *shttp.Response {
			return shttp.OK()
		})

	return router.Handler()
}

func (s *RouterJSONErrorsSuite) errorBody(body []byte) shttp.APIErrorBody {
	var parsed shttp.APIErrorBody
	s.Require().NoError(json.Unmarshal(body, &parsed))

	return parsed
}

func (s *RouterJSONErrorsSuite) Test_UnknownEndpoint() {
	res := shttptest.Request(s.handler(), shttp.MethodGet, "/v1/nope", nil)

	s.Equal(http.StatusNotFound, res.Code)
	s.Equal("application/json", res.Header().Get("Content-Type"))

	body := s.errorBody(res.Body.Bytes())
	s.Equal("unknown-endpoint", body.Code)
	s.NotEmpty(body.Error)
	s.Contains(body.Error, "/v1/openapi.json")
}

func (s *RouterJSONErrorsSuite) Test_MethodNotAllowed() {
	res := shttptest.Request(s.handler(), shttp.MethodDelete, "/v1/ping", nil)

	s.Equal(http.StatusMethodNotAllowed, res.Code)
	s.Equal("application/json", res.Header().Get("Content-Type"))

	body := s.errorBody(res.Body.Bytes())
	s.Equal("method-not-allowed", body.Code)
	s.NotEmpty(body.Error)
}

func (s *RouterJSONErrorsSuite) Test_KnownEndpointIsUnaffected() {
	res := shttptest.Request(s.handler(), shttp.MethodGet, "/v1/ping", nil)

	s.Equal(http.StatusOK, res.Code)
}

// Forbidden answers API-key callers, end users of a customer's app and
// signed-in dashboard users alike, so it carries a body without naming a
// credential none of them may hold.
func (s *RouterJSONErrorsSuite) Test_ForbiddenCarriesAJsonBody() {
	res := shttp.Forbidden()

	body, ok := res.Data.(shttp.APIErrorBody)

	s.Require().True(ok)
	s.Equal(http.StatusForbidden, res.Status)
	s.Equal("forbidden", body.Code)
	s.NotEmpty(body.Error)
	s.NotContains(body.Error, "API key")
	s.Empty(body.Docs)
}

// The API-key variant is the one that may name the credential.
func (s *RouterJSONErrorsSuite) Test_ForbiddenAPIKeyNamesTheCredential() {
	body, ok := shttp.ForbiddenAPIKey().Data.(shttp.APIErrorBody)

	s.Require().True(ok)
	s.Equal("forbidden", body.Code)
	s.Contains(body.Error, "API key")
	s.Equal(shttp.DocsAuthenticationURL, body.Docs)
}

// NotAllowed answers admin logins and webhook callbacks, so it must not tell
// the caller to send an API key.
func (s *RouterJSONErrorsSuite) Test_NotAllowedDoesNotNameACredential() {
	data, ok := shttp.NotAllowed().Data.(map[string]any)

	s.Require().True(ok)
	s.Equal(false, data["user"])
	s.Equal("unauthorized", data["code"])
	s.NotContains(data["error"].(string), "API key")
}

func (s *RouterJSONErrorsSuite) Test_NotFoundJSON() {
	res := shttp.NotFoundJSON("The environment does not exist.")

	body, ok := res.Data.(shttp.APIErrorBody)

	s.Require().True(ok)
	s.Equal(http.StatusNotFound, res.Status)
	s.Equal("not-found", body.Code)
	s.Equal("The environment does not exist.", body.Error)
}

func (s *RouterJSONErrorsSuite) Test_NotFoundJSON_DefaultMessage() {
	body, ok := shttp.NotFoundJSON().Data.(shttp.APIErrorBody)

	s.Require().True(ok)
	s.NotEmpty(body.Error)
}

// The hosting layer serves the deployment's own 404 page, so the bodyless
// NotFound it relies on must stay bodyless.
func (s *RouterJSONErrorsSuite) Test_NotFoundStaysBodyless() {
	res := shttp.NotFound()

	s.Equal(http.StatusNotFound, res.Status)
	s.Nil(res.Data)
}

func TestRouterJSONErrorsSuite(t *testing.T) {
	suite.Run(t, new(RouterJSONErrorsSuite))
}
