package authhandlers_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user/authhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

func TestHandlerAuthGithubInstallationCallback_Success(t *testing.T) {
	response := shttptest.Request(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/auth/github/installation",
		nil,
	)

	if response.Code != http.StatusOK {
		t.Fatalf("Was expecting 200 but received: %d", response.Code)
	}

	if response.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("Wrong content type: %s", response.Header().Get("Content-Type"))
	}

	expected := `window.opener && window.opener.postMessage({"success":true}, "*")`

	if strings.Contains(response.String(), expected) == false {
		t.Fatalf("Was expecting a different response")
	}
}
