package authwallhandlers

import (
	"net/http"
	"net/url"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/lib/html"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// sendMagicLink emails a sign-in link. It runs in the background, so the
// response takes as long whether or not the email has a login.
var sendMagicLink = func(p authwall.SendMagicLinkParams) {
	go func() {
		if err := (authwall.MagicLinkEmail{}).Send(p); err != nil {
			slog.Errorf("error while sending auth wall magic link: %s", err.Error())
		}
	}()
}

// magicLinkAvailable reports whether sign-in links can be sent.
var magicLinkAvailable = authwall.MagicLinkEmail{}.Available

// magicLinkFlow signs visitors in to an environment's wall with a link sent to
// their email: the login form asks for the link, the link opens a page that
// asks the visitor to continue, and continuing signs them in. The page is
// there because email scanners open links before the visitor does; they do
// not submit forms, so they cannot use the link up.
type magicLinkFlow struct {
	req    *shttp.RequestContext
	envID  types.ID
	tokens authwall.Token
}

func newMagicLinkFlow(req *shttp.RequestContext, envID string) magicLinkFlow {
	id := utils.StringToID(envID)

	return magicLinkFlow{req: req, envID: id, tokens: authwall.Token{EnvID: id}}
}

// handlerAuthMagic sends a sign-in link to the email submitted on the login
// form, when a login with that email may use one and opens the wall.
func handlerAuthMagic(req *shttp.RequestContext) *shttp.Response {
	flow := newMagicLinkFlow(req, req.FormValue("envId"))
	form := flow.tokens.ParseForm(req.FormValue("token"))

	if form == nil {
		return shttp.BadRequest(map[string]any{
			"error": "The login form has expired. Go back, reload the page and try again.",
		})
	}

	referrer := loginRedirect{returnTo: form.ReturnTo, referer: req.Referer()}.target()

	if form.Expired {
		return failedLoginResponse(referrer, "invalid_token")
	}

	email := req.FormValue("email")

	if !utils.IsValidEmail(email) {
		return failedLoginResponse(referrer, "invalid_email")
	}

	if !magicLinkAvailable() {
		return failedLoginResponse(referrer, "magic_link_unavailable")
	}

	aw, err := authwall.Store().MagicLinkLogin(req.Context(), authwall.MagicLinkLoginParams{
		EnvID: flow.envID,
		Email: email,
	})

	if err != nil {
		return shttp.Error(err)
	}

	// The page is the same whether or not the email has a login, so it does
	// not tell visitors which emails do.
	if aw != nil {
		if res := flow.send(aw, referrer); res != nil {
			return res
		}
	}

	return flow.page(http.StatusOK, html.Templates["magic_link_sent"], map[string]any{
		"email":     email,
		"minutes":   authwall.MagicLinkMaxMins,
		"return_to": referrer,
	})
}

// handlerAuthMagicContinue shows the page a sign-in link opens. It does not
// use the link up.
func handlerAuthMagicContinue(req *shttp.RequestContext) *shttp.Response {
	flow := magicLinkFlow{req: req}
	id := req.Query().Get("id")
	link, err := authwall.MagicLinks{}.Get(req.Context(), id)

	if err != nil {
		return shttp.Error(err)
	}

	if link == nil {
		return flow.invalid()
	}

	return flow.page(http.StatusOK, html.Templates["magic_link_continue"], map[string]any{
		"api_host": admin.MustConfig().ApiURL(""),
		"id":       id,
		"email":    link.Email,
		"host":     flow.host(link.ReturnTo),
	})
}

// handlerAuthMagicVerify uses a sign-in link up and signs the visitor in.
func handlerAuthMagicVerify(req *shttp.RequestContext) *shttp.Response {
	flow := magicLinkFlow{req: req}
	link, err := authwall.MagicLinks{}.Use(req.Context(), req.FormValue("id"))

	if err != nil {
		return shttp.Error(err)
	}

	if link == nil {
		return flow.invalid()
	}

	// The login may have been removed, lost the magic link or lost access to
	// the app since the link was sent.
	store := authwall.Store()
	aw, err := store.MagicLinkLogin(req.Context(), authwall.MagicLinkLoginParams{
		EnvID: link.EnvID,
		Email: link.Email,
	})

	if err != nil {
		return shttp.Error(err)
	}

	if aw == nil || aw.LoginID != link.LoginID {
		return flow.invalid()
	}

	session, err := authwall.Token{EnvID: link.EnvID}.Session(authwall.Session{
		Email:    aw.LoginEmail,
		Metadata: aw.Metadata,
	})

	if err != nil || session == "" {
		return failedLoginResponse(link.ReturnTo, "token_generation_failed")
	}

	if err := store.UpdateLastLogin(req.Context(), aw.LoginID); err != nil {
		slog.Errorf("error while updating last login: %s", err.Error())
	}

	return &shttp.Response{
		Redirect: addQueryParamToURL(link.ReturnTo, "stormkit_success", session),
	}
}

// send emails a sign-in link for the login that returns the visitor to
// returnTo. The email is made of what Stormkit stores, never of what the
// visitor submitted.
func (f magicLinkFlow) send(aw *authwall.AuthWall, returnTo string) *shttp.Response {
	ctx := f.req.Context()
	id, err := authwall.MagicLinks{}.Create(ctx, authwall.MagicLink{
		EnvID:    f.envID,
		LoginID:  aw.LoginID,
		Email:    aw.LoginEmail,
		ReturnTo: returnTo,
	})

	if err != nil {
		return shttp.Error(err)
	}

	name, err := authwall.Store().EnvName(ctx, f.envID)

	if err != nil {
		return shttp.Error(err)
	}

	sendMagicLink(authwall.SendMagicLinkParams{
		To:   aw.LoginEmail,
		Name: name,
		URL:  admin.MustConfig().ApiURL("/auth-wall/magic?id=" + url.QueryEscape(id)),
	})

	return nil
}

// host returns the host of the protected page.
func (f magicLinkFlow) host(returnTo string) string {
	if u, err := url.Parse(returnTo); err == nil && u.Host != "" {
		return u.Host
	}

	return "the deployment"
}

// invalid is the page of a link that has expired or was already used.
func (f magicLinkFlow) invalid() *shttp.Response {
	return f.page(http.StatusBadRequest, html.Templates["magic_link_invalid"], map[string]any{
		"minutes": authwall.MagicLinkMaxMins,
	})
}

func (f magicLinkFlow) page(status int, content string, data map[string]any) *shttp.Response {
	return &shttp.Response{
		Status: status,
		Data: html.MustRender(html.RenderArgs{
			PageTitle:   "Stormkit - Sign in",
			PageContent: content,
			ContentData: data,
		}),
		Headers: http.Header{
			"Content-Type": []string{"text/html; charset=utf-8"},
			// The pages carry the sign-in token.
			"Cache-Control":   []string{"no-store"},
			"Referrer-Policy": []string{"no-referrer"},
		},
	}
}
