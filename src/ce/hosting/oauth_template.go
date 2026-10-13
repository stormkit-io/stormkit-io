package hosting

import (
	"mime"
	"net/http"
	"net/url"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

// oauthTemplatePath is the deployment file that, when present, replaces the
// built-in sign-in and consent screens of the OAuth server. Like every file
// under appconf.ReservedPathPrefix it is never served at its own path, so its
// source does not leak.
const oauthTemplatePath = "/_stormkit/oauth.html"

// oauthTemplateParams describes one render of the app's OAuth page.
type oauthTemplateParams struct {
	File   *appconf.StaticFile
	Status int
	Data   map[string]any
}

// loginTemplate returns the deployment's OAuth page, or nil when it ships none.
func (o *oauthServer) loginTemplate() *appconf.StaticFile {
	return o.req.Host.Config.ReservedFiles[oauthTemplatePath]
}

// templateData is the context the app's OAuth page is rendered with. A value
// that does not apply to the visitor is left out, so the page can branch on it
// with {{if}} or {{with}}.
func (o *oauthServer) templateData(p authzParams, client oauthClient) map[string]any {
	uid, eml, signedIn := o.sessionIdentity()

	data := map[string]any{
		"client_name": clientDisplayName(p, client),
		"client_id":   p.clientID,
		"scopes":      scopeList(p.scope),
		"signed_in":   signedIn,
		"cancel_url":  denyURL(p),
	}

	if u, err := url.Parse(p.redirectURI); err == nil {
		data["redirect_host"] = u.Host
	}

	if signedIn {
		data["user"] = map[string]any{"id": uid, "email": eml}
		data["continue_url"] = o.req.URL().RequestURI()
	} else if target, ok := o.loginTarget(); ok {
		data["login_url"] = target
	}

	return data
}

// renderTemplate renders the app's OAuth page. The page carries the same
// anti-framing headers as the built-in screens. Its referrer policy is
// same-origin rather than no-referrer: a form POST with no-referrer sends
// "Origin: null", which grant() rejects as cross-origin.
func (o *oauthServer) renderTemplate(p oauthTemplateParams) *shttp.Response {
	cnf := o.req.Host.Config

	file, err := integrations.Client().GetFile(integrations.GetFileArgs{
		Location:     cnf.StorageLocation,
		DeploymentID: cnf.DeploymentID,
		FileName:     p.File.FileName,
	})

	if err != nil || file == nil {
		slog.Errorf("oauth: cannot load %s for deployment %s: %v", oauthTemplatePath, cnf.DeploymentID.String(), err)
		return o.builtInErrorPage("The sign-in page could not be loaded.")
	}

	content, err := pageTemplate{Body: string(file.Content)}.render(p.Data)

	if err != nil {
		return o.builtInErrorPage("The sign-in page could not be rendered: " + err.Error())
	}

	headers := shttp.HeadersFromMap(p.File.Headers)

	for k, v := range map[string]string{
		"Content-Type":            "text/html; charset=utf-8",
		"Referrer-Policy":         "same-origin",
		"Cache-Control":           "no-store",
		"X-Frame-Options":         "DENY",
		"Content-Security-Policy": "frame-ancestors 'none'",
	} {
		headers.Set(k, v)
	}

	return &shttp.Response{
		Status:  p.Status,
		Headers: headers,
		Data:    content,
	}
}

// isFormPost reports whether the request is an HTML form submission, as opposed
// to the built-in consent page's fetch call.
func (o *oauthServer) isFormPost() bool {
	mediaType, _, _ := mime.ParseMediaType(o.req.Header.Get("Content-Type"))

	return mediaType == "application/x-www-form-urlencoded" || mediaType == "multipart/form-data"
}

// formResponse turns grant()'s JSON answer into one a browser form can follow:
// a redirect to the client on success, or an error page otherwise.
func (o *oauthServer) formResponse(res *shttp.Response) *shttp.Response {
	body, _ := res.Data.(map[string]any)

	if redirect, ok := body["redirect"].(string); ok && res.Status == http.StatusOK {
		return &shttp.Response{
			Status:   http.StatusFound,
			Redirect: &redirect,
			Headers: shttp.HeadersFromMap(map[string]string{
				"Cache-Control": "no-store",
			}),
		}
	}

	msg, _ := body["error_description"].(string)

	if msg == "" {
		msg = "Authorization failed. Please try again."
	}

	page := o.errorPage(msg)
	page.Status = res.Status

	return page
}
