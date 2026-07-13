package authhandlers

import (
	"bytes"
	"net/http"

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

	buf := &bytes.Buffer{}
	err := responseTmpl.Execute(buf, map[string]any{
		"message": cr.htmlMessage,
		"json":    cr.postMessage,
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
