package hosting

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/html"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type skAuthMiddleware struct {
	req *RequestContext
}

func (m *skAuthMiddleware) handleRegisterLogin(path string) (*shttp.Response, error) {
	if m.req.Method != http.MethodPost {
		return &shttp.Response{
			Status: http.StatusMethodNotAllowed,
			Data:   map[string]any{"errors": []string{"method not allowed"}},
		}, nil
	}

	if path == "/_stormkit/auth/login" {
		return m.login(), nil
	}

	return m.register(), nil
}

func (m *skAuthMiddleware) handleVerify() (*shttp.Response, error) {
	if m.req.Method != http.MethodGet {
		return &shttp.Response{
			Status: http.StatusMethodNotAllowed,
			Data:   map[string]any{"errors": []string{"method not allowed"}},
		}, nil
	}

	resp := m.verifyEmail()

	if resp.Status != 0 && resp.Status != http.StatusOK {
		errMsg := "verification failed"

		if d, ok := resp.Data.(map[string]any); ok {
			if errs, ok := d["errors"].([]string); ok && len(errs) > 0 {
				errMsg = errs[0]
			}
		}

		return m.renderVerifyPage(resp.Status, "", errMsg), nil
	}

	data, ok := resp.Data.(map[string]any)

	if !ok {
		return m.renderVerifyPage(http.StatusInternalServerError, "", "verification failed"), nil
	}

	head := fmt.Sprintf(
		`<script>localStorage.setItem('skauth', JSON.stringify('%s'));window.location.href="%s?verified=true";</script>`,
		data["token"],
		m.req.Host.Config.SKAuth.SuccessURL,
	)

	return m.renderVerifyPage(http.StatusOK, head, ""), nil
}

func (m *skAuthMiddleware) handleMagicLinkRequest() (*shttp.Response, error) {
	resp := m.magicLinkRequest()

	if resp.Status != 0 && resp.Status != http.StatusOK {
		return resp, nil
	}

	return &shttp.Response{Status: http.StatusCreated}, nil
}

func (m *skAuthMiddleware) handleMagicLinkVerify() (*shttp.Response, error) {
	resp := m.magicLinkVerify()

	if resp.Status != 0 && resp.Status != http.StatusOK {
		errMsg := "magic link verification failed"

		if d, ok := resp.Data.(map[string]any); ok {
			if errs, ok := d["errors"].([]string); ok && len(errs) > 0 {
				errMsg = errs[0]
			}
		}

		return m.renderVerifyPage(resp.Status, "", errMsg), nil
	}

	data, ok := resp.Data.(map[string]any)

	if !ok {
		return m.renderVerifyPage(http.StatusInternalServerError, "", "magic link verification failed"), nil
	}

	successURL := m.req.Host.Config.SKAuth.SuccessURL

	// Cross-origin: the POST came from a whitelisted frontend. Bounce
	// the user back to that origin with the session token in the URL
	// fragment so the destination SPA can persist it. We deliberately
	// do NOT touch localStorage here because this host is the wrong
	// origin for the token. SuccessURL is validated at config time to
	// start with '/', so origin (no trailing slash) + successURL joins
	// cleanly.
	if redirect, _ := data["redirect"].(string); redirect != "" {
		target := fmt.Sprintf("%s%s#skauth=%s", redirect, successURL, data["token"])
		head := fmt.Sprintf(`<script>window.location.href=%q;</script>`, target)

		return m.renderVerifyPage(http.StatusOK, head, ""), nil
	}

	head := fmt.Sprintf(
		`<script>localStorage.setItem('skauth', JSON.stringify(%q));window.location.href=%q;</script>`,
		data["token"],
		successURL+"?verified=true",
	)

	return m.renderVerifyPage(http.StatusOK, head, ""), nil
}

func (m *skAuthMiddleware) handleCallback() (*shttp.Response, error) {
	var head, content string

	code := m.req.Query().Get("code")
	status := http.StatusOK

	if code == "" {
		status = http.StatusBadRequest
		content = "code is missing"
	} else if sessionToken := rediscache.Client().Get(m.req.Context(), code).Val(); sessionToken != "" {
		head = fmt.Sprintf(
			`<script>localStorage.setItem('skauth', JSON.stringify('%s'));window.location.href="%s";</script>`,
			sessionToken,
			m.req.Host.Config.SKAuth.SuccessURL,
		)
	} else {
		content = "invalid session"
	}

	return &shttp.Response{
		Status: status,
		Headers: shttp.HeadersFromMap(map[string]string{
			"Content-Type": "text/html",
		}),
		Data: html.MustRender(html.RenderArgs{
			PageTitle:   "Stormkit - Auth",
			PageContent: content,
			PageHead:    head,
		}),
	}, nil
}

func (m *skAuthMiddleware) renderVerifyPage(status int, head, content string) *shttp.Response {
	return &shttp.Response{
		Status: status,
		Headers: shttp.HeadersFromMap(map[string]string{
			"Content-Type":    "text/html",
			"Referrer-Policy": "no-referrer",
			"Cache-Control":   "no-store",
		}),
		Data: html.MustRender(html.RenderArgs{
			PageTitle:   "Stormkit - Auth",
			PageHead:    head,
			PageContent: content,
		}),
	}
}

func WithSKAuth(req *RequestContext) (*shttp.Response, error) {
	if req.Host.Config.SKAuth == nil {
		return nil, nil
	}

	// Strip the header to prevent clients from spoofing it.
	req.Header.Del("X-User-Id")

	if bearer := user.ParseBearer(req.Header.Get("Authorization")); bearer != "" {
		claims := user.ParseJWT(&user.ParseJWTArgs{
			Bearer:  bearer,
			Secret:  req.Host.Config.SKAuth.Secret,
			MaxMins: req.Host.Config.SKAuth.TTL,
		})

		if claims != nil {
			if userID, ok := claims["uid"].(string); ok && userID != "" {
				req.Header.Set("X-User-Id", userID)
			}
		}
	}

	path := req.URL().Path

	if !strings.HasPrefix(path, "/_stormkit/auth") {
		return nil, nil
	}

	// CORS preflight — respond 204 so the browser proceeds with the actual
	// request. Per-path custom headers (e.g. Access-Control-Allow-Origin)
	// are layered on by finalize() afterwards. Without this short-circuit
	// the request would fall through to handlers that expect a JSON body
	// and reject OPTIONS with 400, breaking the preflight.
	if req.Method == http.MethodOptions {
		return &shttp.Response{Status: http.StatusNoContent}, nil
	}

	m := &skAuthMiddleware{req: req}

	if path == "/_stormkit/auth/register" || path == "/_stormkit/auth/login" {
		return m.handleRegisterLogin(path)
	}

	if path == "/_stormkit/auth/verify" {
		return m.handleVerify()
	}

	if path == "/_stormkit/auth/magic" {
		if m.req.Query().Get("token") != "" {
			return m.handleMagicLinkVerify()
		}

		return m.handleMagicLinkRequest()
	}

	return m.handleCallback()
}
