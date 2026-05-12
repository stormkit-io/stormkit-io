package hosting

import (
	"fmt"
	"net/http"
	"strings"

	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/lib/html"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type skAuthMiddleware struct {
	req *RequestContext
}

func (m *skAuthMiddleware) injectEnvID() {
	reqURL := m.req.URL()
	q := reqURL.Query()
	q.Set("envId", m.req.Host.Config.EnvID.String())
	reqURL.RawQuery = q.Encode()
	m.req.ResetQuery()
}

func (m *skAuthMiddleware) handleRegisterLogin(path string) (*shttp.Response, error) {
	if m.req.Method != http.MethodPost {
		return &shttp.Response{
			Status: http.StatusMethodNotAllowed,
			Data:   map[string]any{"errors": []string{"method not allowed"}},
		}, nil
	}

	// Inject the environment ID from the host config so the handler can
	// look up the environment without requiring it in the request body.
	m.injectEnvID()

	if path == "/_stormkit/auth/login" {
		return publicapiv1.HandlerAuthEmailLogin(m.req.RequestContext), nil
	}

	return publicapiv1.HandlerAuthEmailRegister(m.req.RequestContext), nil
}

func (m *skAuthMiddleware) handleVerify() (*shttp.Response, error) {
	if m.req.Method != http.MethodGet {
		return &shttp.Response{
			Status: http.StatusMethodNotAllowed,
			Data:   map[string]any{"errors": []string{"method not allowed"}},
		}, nil
	}

	m.injectEnvID()

	resp := publicapiv1.HandlerAuthVerify(m.req.RequestContext)

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
	m.injectEnvID()

	resp := publicapiv1.HandlerAuthMagicLinkRequest(m.req.RequestContext)

	if resp.Status != 0 && resp.Status != http.StatusOK {
		return resp, nil
	}

	return &shttp.Response{Status: http.StatusCreated}, nil
}

func (m *skAuthMiddleware) handleMagicLinkVerify() (*shttp.Response, error) {
	m.injectEnvID()

	resp := publicapiv1.HandlerAuthMagicLinkVerify(m.req.RequestContext)

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

	head := fmt.Sprintf(
		`<script>localStorage.setItem('skauth', JSON.stringify('%s'));window.location.href="%s?verified=true";</script>`,
		data["token"],
		m.req.Host.Config.SKAuth.SuccessURL,
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

	path := req.URL().Path

	if !strings.HasPrefix(path, "/_stormkit/auth") {
		return nil, nil
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
