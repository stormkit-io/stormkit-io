package publicapiv1_test

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
	"testing"

	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
)

// muxVarPattern matches the regexp constraint gorilla mux allows inside a path
// variable ("{id:[0-9]+}"), which OpenAPI paths do not carry.
var muxVarPattern = regexp.MustCompile(`\{([^:}]+):[^}]+\}`)

type openAPIOperation struct {
	OperationID string `json:"operationId"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Parameters  []struct {
		Ref    string          `json:"$ref"`
		Name   string          `json:"name"`
		In     string          `json:"in"`
		Schema json.RawMessage `json:"schema"`
	} `json:"parameters"`
	RequestBody *struct {
		Content map[string]struct {
			Schema json.RawMessage `json:"schema"`
		} `json:"content"`
	} `json:"requestBody"`
	Responses map[string]struct {
		Ref         string `json:"$ref"`
		Description string `json:"description"`
		Content     map[string]struct {
			Schema json.RawMessage `json:"schema"`
		} `json:"content"`
	} `json:"responses"`
}

type openAPIDoc struct {
	OpenAPI string `json:"openapi"`
	Info    struct {
		Title       string `json:"title"`
		Version     string `json:"version"`
		Description string `json:"description"`
	} `json:"info"`
	Servers []struct {
		URL string `json:"url"`
	} `json:"servers"`
	Paths      map[string]map[string]openAPIOperation `json:"paths"`
	Components struct {
		SecuritySchemes map[string]json.RawMessage `json:"securitySchemes"`
		Parameters      map[string]json.RawMessage `json:"parameters"`
		Responses       map[string]json.RawMessage `json:"responses"`
		Schemas         map[string]json.RawMessage `json:"schemas"`
	} `json:"components"`
}

type OpenAPISuite struct {
	suite.Suite

	doc openAPIDoc
}

func (s *OpenAPISuite) SetupTest() {
	s.Require().NoError(json.Unmarshal(publicapiv1.OpenAPISpec(), &s.doc))
}

func (s *OpenAPISuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(publicapiv1.Services).Router().Handler()
}

// operations returns every operation in the document keyed as "METHOD:/path",
// matching the key format the router reports for registered handlers.
func (s *OpenAPISuite) operations() map[string]openAPIOperation {
	ops := map[string]openAPIOperation{}

	for path, methods := range s.doc.Paths {
		for method, op := range methods {
			ops[strings.ToUpper(method)+":"+path] = op
		}
	}

	return ops
}

func (s *OpenAPISuite) Test_IsValidDocument() {
	s.Equal("3.1.0", s.doc.OpenAPI)
	s.Equal("Stormkit API", s.doc.Info.Title)
	s.NotEmpty(s.doc.Info.Version)
	s.NotEmpty(s.doc.Info.Description)
	s.NotEmpty(s.doc.Servers)
	s.Contains(s.doc.Components.SecuritySchemes, "ApiKeyAuth")
}

// Test_EveryRouteIsDocumented is the guard that keeps the specification honest:
// a new public endpoint fails this test until it is described.
func (s *OpenAPISuite) Test_EveryRouteIsDocumented() {
	ops := s.operations()

	for _, key := range shttp.NewRouter().RegisterService(publicapiv1.Services).HandlerKeys() {
		normalized := muxVarPattern.ReplaceAllString(key, "{$1}")

		s.Containsf(ops, normalized, "route %s is not described in openapi.json", key)
	}
}

// Test_EveryOperationIsRouted catches the opposite drift: an operation that
// describes an endpoint the API no longer serves.
func (s *OpenAPISuite) Test_EveryOperationIsRouted() {
	registered := map[string]bool{}

	for _, key := range shttp.NewRouter().RegisterService(publicapiv1.Services).HandlerKeys() {
		registered[muxVarPattern.ReplaceAllString(key, "{$1}")] = true
	}

	for key := range s.operations() {
		s.Truef(registered[key], "openapi.json describes %s, which is not a registered route", key)
	}
}

// Test_OperationsAreFunctionCallingReady asserts the properties an LLM needs to
// turn an operation into a callable tool: a unique id, a description, typed
// parameters and a response schema.
func (s *OpenAPISuite) Test_OperationsAreFunctionCallingReady() {
	seen := map[string]string{}

	for key, op := range s.operations() {
		s.NotEmptyf(op.OperationID, "%s has no operationId", key)
		s.NotEmptyf(op.Summary, "%s has no summary", key)
		s.NotEmptyf(op.Description, "%s has no description", key)

		if previous, ok := seen[op.OperationID]; ok {
			s.Failf("duplicate operationId", "%s and %s share operationId %s", previous, key, op.OperationID)
		}

		seen[op.OperationID] = key

		for _, param := range op.Parameters {
			if param.Ref != "" {
				continue
			}

			s.NotEmptyf(param.Name, "%s has an unnamed parameter", key)
			s.NotEmptyf(param.In, "%s: parameter %s has no location", key, param.Name)
			s.NotEmptyf(param.Schema, "%s: parameter %s has no schema", key, param.Name)
		}

		if op.RequestBody != nil {
			s.NotEmptyf(op.RequestBody.Content, "%s has a request body without content", key)

			for mediaType, media := range op.RequestBody.Content {
				s.NotEmptyf(media.Schema, "%s: request body %s has no schema", key, mediaType)
			}
		}

		success, ok := op.Responses["200"]
		s.Truef(ok, "%s documents no 200 response", key)
		s.NotEmptyf(success.Description, "%s: the 200 response has no description", key)
		s.NotEmptyf(success.Content, "%s: the 200 response has no content", key)

		for mediaType, media := range success.Content {
			s.NotEmptyf(media.Schema, "%s: the 200 response %s has no schema", key, mediaType)
		}
	}
}

// Test_ErrorResponsesAreDocumented asserts every authenticated operation tells
// an agent what a failure looks like, so it can recover instead of guessing.
func (s *OpenAPISuite) Test_ErrorResponsesAreDocumented() {
	for key, op := range s.operations() {
		if key == "GET:/v1/openapi.json" {
			continue
		}

		s.Containsf(op.Responses, "403", "%s does not document a 403 response", key)
		s.Containsf(op.Responses, "500", "%s does not document a 500 response", key)
	}

	s.Contains(s.doc.Components.Schemas, "Error")
	s.Contains(s.doc.Components.Responses, "Forbidden")
	s.Contains(s.doc.Components.Responses, "NotFound")
}

// Test_ReferencesResolve makes sure no $ref points at a component that does not
// exist — a broken reference makes the whole document unusable to a generator.
func (s *OpenAPISuite) Test_ReferencesResolve() {
	refs := regexp.MustCompile(`"#/components/(parameters|responses|schemas)/([A-Za-z0-9_]+)"`).
		FindAllStringSubmatch(string(publicapiv1.OpenAPISpec()), -1)

	s.NotEmpty(refs)

	for _, ref := range refs {
		switch ref[1] {
		case "parameters":
			s.Containsf(s.doc.Components.Parameters, ref[2], "unresolved parameter reference %s", ref[2])
		case "responses":
			s.Containsf(s.doc.Components.Responses, ref[2], "unresolved response reference %s", ref[2])
		case "schemas":
			s.Containsf(s.doc.Components.Schemas, ref[2], "unresolved schema reference %s", ref[2])
		}
	}
}

// Test_IsServedWithoutAuthentication verifies the endpoint an agent hits first
// needs no credentials.
func (s *OpenAPISuite) Test_IsServedWithoutAuthentication() {
	response := shttptest.Request(s.handler(), shttp.MethodGet, "/v1/openapi.json", nil)

	s.Equal(http.StatusOK, response.Code)
	s.Equal("application/json; charset=utf-8", response.Header().Get("Content-Type"))

	var served openAPIDoc
	s.Require().NoError(json.Unmarshal(response.Body.Bytes(), &served))
	s.Equal("Stormkit API", served.Info.Title)
}

func TestOpenAPISuite(t *testing.T) {
	suite.Run(t, new(OpenAPISuite))
}
