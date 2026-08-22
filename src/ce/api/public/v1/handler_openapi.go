package publicapiv1

import (
	_ "embed"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// openAPISpec is the machine-readable description of this API. It is embedded
// rather than generated so that the same bytes can be served here and shipped
// as a static file on the marketing site; openapi_test.go keeps it in sync with
// the routes registered in services.go.
//
//go:embed openapi.json
var openAPISpec []byte

// OpenAPISpec returns the raw OpenAPI document.
func OpenAPISpec() []byte {
	return openAPISpec
}

// handlerOpenAPI serves the OpenAPI document. It is intentionally
// unauthenticated: an agent has to be able to discover the API surface before
// it holds a key.
func handlerOpenAPI(req *shttp.RequestContext) *shttp.Response {
	return &shttp.Response{
		Status: http.StatusOK,
		Data:   openAPISpec,
		Headers: http.Header{
			"Content-Type":  []string{"application/json; charset=utf-8"},
			"Cache-Control": []string{"public, max-age=3600"},
		},
	}
}
