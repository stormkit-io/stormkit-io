package hosting

import (
	"net/http"
	"net/url"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
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
		// The session's eml claim is encrypted. It is decrypted only for display:
		// grant() forwards the encrypted value into the access token, where the
		// edge middleware expects it. An undecryptable claim renders no email.
		email := utils.DecryptToString(eml, emlKey(o.secret()))
		data["user"] = map[string]any{"id": uid, "email": email}
		data["continue_url"] = o.req.URL().RequestURI()
	} else if target, ok := o.loginTarget(); ok {
		data["login_url"] = target
	}

	data["sign_in"] = o.signInData()

	return data
}

// oauthSignInMessages explains each sign-in outcome the page can be sent back
// with. A code the server does not know shows nothing.
var oauthSignInMessages = map[string]string{
	oauthSignInInvalidEmail: "Enter a valid email address.",
	oauthSignInExpired:      "This sign-in link has expired or was already used. Request a new one.",
	oauthSignInFailed:       "Sign-in did not complete. Please try again.",
}

// signInData lists the sign-in methods the page can offer, each bringing the
// visitor back to this authorize request, and the outcome of the last attempt.
// Only enabled providers are listed, so the page can branch on them with
// {{with}}.
func (o *oauthServer) signInData() map[string]any {
	q := o.req.URL().Query()
	outcome := q.Get(oauthSignInParam)

	q.Del(oauthSignInParam)
	q.Del(oauthLoginRetryParam)

	returnTo := url.QueryEscape((&url.URL{Path: oauthAuthorizePath, RawQuery: q.Encode()}).String())

	data := map[string]any{
		"link_sent": outcome == oauthSignInLinkSent,
		"error":     oauthSignInMessages[outcome],
	}

	providers, err := skauth.NewStore().Providers(o.req.Context(), skauth.ProvidersArgs{EnvID: o.req.Host.Config.EnvID})

	if err != nil {
		slog.Errorf("oauth: cannot list sign-in providers for env %s: %v", o.req.Host.Config.EnvID.String(), err)
		return data
	}

	for _, prv := range providers {
		if prv == nil || !prv.Status {
			continue
		}

		switch prv.Name {
		case skauth.ProviderGoogle, skauth.ProviderX, skauth.ProviderGitHub:
			data[prv.Name] = "/_stormkit/auth/" + prv.Name + "?return_to=" + returnTo
		case skauth.ProviderMagicLink:
			data["magic_link"] = "/_stormkit/auth/magic?return_to=" + returnTo
		}
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
