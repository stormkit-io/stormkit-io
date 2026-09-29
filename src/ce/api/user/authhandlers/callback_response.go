package authhandlers

import (
	"bytes"
	"net/http"
	"net/url"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type jsonMsg map[string]any

type callbackResponse struct {
	status      int
	err         error
	htmlMessage string
	postMessage jsonMsg
}

// cbResponse creates a new callbackResponse instance.
func cbResponse(status int) *callbackResponse {
	return &callbackResponse{
		status: status,
	}
}

// setError sets the error instance.
func (cr *callbackResponse) setError(err error) *callbackResponse {
	cr.err = err
	return cr
}

// html sets the html message.
func (cr *callbackResponse) html(html string) *callbackResponse {
	cr.htmlMessage = html
	return cr
}

// json sets the json message to post to the client.
func (cr *callbackResponse) json(data map[string]any) *callbackResponse {
	cr.postMessage = data
	return cr
}

// send sends the prepared response to the browser.
func (cr *callbackResponse) send() *shttp.Response {
	if cr.htmlMessage == "" {
		// Make sure that success parameter is there.
		success, _ := cr.postMessage["success"].(bool)
		cr.postMessage["success"] = success

		if cr.err != nil || !success {
			cr.htmlMessage = "Invalid authorization"
		} else {
			cr.htmlMessage = "Authorized"
		}
	}

	targetOrigin := cr.targetOrigin()

	if targetOrigin == "" {
		cr.htmlMessage = "The Stormkit app URL is not configured, so the result cannot be sent back to Stormkit."
	}

	buf := &bytes.Buffer{}
	err := responseTmpl.Execute(buf, map[string]any{
		"message":      cr.htmlMessage,
		"json":         cr.postMessage,
		"targetOrigin": targetOrigin,
	})

	data, execErr := buf.String(), err

	if execErr != nil {
		cr.status = http.StatusInternalServerError
		cr.err = execErr
	}

	headers := http.Header{}
	headers.Add("Content-Type", "text/html; charset=utf-8")

	return &shttp.Response{
		Headers: headers,
		Status:  cr.status,
		Error:   cr.err,
		Data:    data,
	}
}

// targetOrigin returns the origin of the Stormkit app, the only window allowed
// to receive the result. The result carries a session token, so posting it to
// any other opener would hand the session to whichever site opened the login
// window. It returns an empty string, and nothing is posted, when the app URL
// is not configured.
func (cr *callbackResponse) targetOrigin() string {
	appURL, err := url.Parse(admin.MustConfig().AppURL(""))

	if err != nil || appURL.Scheme == "" || appURL.Host == "" {
		return ""
	}

	return appURL.Scheme + "://" + appURL.Host
}
