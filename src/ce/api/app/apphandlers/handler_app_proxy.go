package apphandlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type proxyRequest struct {
	URL string `json:"url"`
}

// handlerAppProxy proxies the given request and makes a get request
// to receive the status code.
func handlerAppProxy(req *app.RequestContext) *shttp.Response {
	pr := &proxyRequest{}

	if err := req.Post(pr); err != nil {
		return shttp.Error(err)
	}

	if !strings.HasPrefix(pr.URL, "http") {
		pr.URL = fmt.Sprintf("https://%s", pr.URL)
	}

	res, err := shttp.Head(pr.URL).Do()
	var statusCode int

	if err != nil {
		errString := err.Error()

		if strings.Contains(errString, "no such host") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(errString, ".stormkit.dev") && strings.Contains(errString, "stream error") {
			statusCode = http.StatusNotFound
		} else {
			statusCode = http.StatusInternalServerError
		}
	}

	if res != nil {
		statusCode = res.StatusCode
	}

	return &shttp.Response{
		Data: map[string]int{
			"status": statusCode,
		},
	}
}
