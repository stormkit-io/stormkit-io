package hosting

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"golang.org/x/sync/singleflight"
)

const (
	// pageDataTimeout caps how long a visitor waits on the loader. The document
	// is already on disk; the fetch is the only thing standing between the
	// request and a response, so it fails fast rather than holding the page.
	pageDataTimeout = 2 * time.Second

	// pageDataMaxBody is the largest document the loader will read. A loader
	// feeds a page's placeholders, not a payload.
	pageDataMaxBody = 1 << 20
)

// pageDataToken matches {{data}}, {{data.field}} and {{data.field:-fallback}}.
// The default is the literal text up to the closing braces: no nesting, no
// second :-, which keeps the parser dumb enough to be predictable.
var pageDataToken = regexp.MustCompile(`\{\{\s*data(?:\.([A-Za-z0-9_.\-]+))?\s*(?::-([^}]*))?\}\}`)

// pageDataInsecure relaxes both guards below. A self-hosted instance whose API
// answers on the private network next to it has no public address to give, and
// neither does a development machine.
func pageDataInsecure() bool {
	return os.Getenv("STORMKIT_PAGE_DATA_INSECURE") == "true"
}

// pageDataClient refuses to dial anything that is not a public address, so a
// loader URL cannot be pointed at the instance's own metadata service or at
// another service on the private network.
var pageDataClient = &http.Client{
	Timeout: pageDataTimeout,
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout: pageDataTimeout,
			Control: func(network, address string, _ syscall.RawConn) error {
				if pageDataInsecure() {
					return nil
				}

				host, _, err := net.SplitHostPort(address)

				if err != nil {
					return err
				}

				ip := net.ParseIP(host)

				if ip == nil {
					return fmt.Errorf("page data: cannot resolve %s", address)
				}

				if !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
					return fmt.Errorf("page data: %s is not a public address", ip)
				}

				return nil
			},
		}).DialContext,
	},
}

// pageDataFlight collapses concurrent fetches of the same document by the same
// visitor into one upstream request. Each visitor still reaches the API on its
// own, with its own User-Agent and address, so the API can count views and tell
// an unfurler from a person. Nothing outlives the request that made it: caching
// belongs to the API.
var pageDataFlight singleflight.Group

// pageData renders a deployment's own document with values fetched from an
// upstream JSON endpoint. The markup, its asset URLs and its design tokens stay
// in the build; only the values come from the API.
type pageData struct {
	loader  *redirects.Loader
	visitor pageDataVisitor
}

// pageDataVisitor is what the loader passes on about the request that asked for
// the page. Without it every fetch would look like Stormkit to the API.
type pageDataVisitor struct {
	userAgent string
	ip        string
}

// loaderResult reports what the fetch found. Status is the upstream status,
// carried so a rule with PassthroughStatus can answer with it.
type loaderResult struct {
	document map[string]any
	status   int
	err      error
}

// fetch returns the loader's document. Concurrent callers for the same URL
// from the same visitor share one request.
func (p *pageData) fetch(ctx context.Context) loaderResult {
	key := strings.Join([]string{p.loader.URL, p.visitor.userAgent, p.visitor.ip}, "\x00")

	shared, _, _ := pageDataFlight.Do(key, func() (any, error) {
		// Detached from the winner's request: if that visitor disconnects, the
		// callers waiting on the shared fetch must not inherit the cancellation.
		// The client's own timeout still bounds it.
		return p.request(context.WithoutCancel(ctx)), nil
	})

	return shared.(loaderResult)
}

func (p *pageData) request(ctx context.Context) loaderResult {
	if !strings.HasPrefix(p.loader.URL, "https://") && !pageDataInsecure() {
		return loaderResult{err: fmt.Errorf("page data: %s is not an https URL", p.loader.URL)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.loader.URL, nil)

	if err != nil {
		return loaderResult{err: err}
	}

	req.Header.Set("Accept", "application/json")

	// An empty User-Agent is sent as empty rather than as Go's default, which
	// would make a visitor without one look like Stormkit.
	req.Header.Set("User-Agent", p.visitor.userAgent)

	if p.visitor.ip != "" {
		req.Header.Set("X-Forwarded-For", p.visitor.ip)
		req.Header.Set("X-Real-IP", p.visitor.ip)
	}

	res, err := pageDataClient.Do(req)

	if err != nil {
		return loaderResult{err: err}
	}

	defer res.Body.Close()

	body, err := io.ReadAll(io.LimitReader(res.Body, pageDataMaxBody))

	if err != nil {
		return loaderResult{status: res.StatusCode, err: err}
	}

	document := map[string]any{}

	// A non-2xx body is not expected to be the document. The status is what the
	// caller acts on; an unparseable body at that point is not an error.
	if err := json.Unmarshal(body, &document); err != nil && res.StatusCode < 300 {
		return loaderResult{status: res.StatusCode, err: err}
	}

	return loaderResult{document: document, status: res.StatusCode}
}

// render replaces every token in the document with its value. A value that is
// absent, null or empty falls back to the token's default, and every value the
// upstream supplied is HTML-escaped: it lands in attributes as often as in text,
// and an unescaped quote there is an injection.
func (p *pageData) render(body string, document map[string]any) string {
	return pageDataToken.ReplaceAllStringFunc(body, func(token string) string {
		groups := pageDataToken.FindStringSubmatch(token)

		key := groups[1]

		// An absent :- and an empty one mean the same thing here: the fallback
		// is the empty string. A token that resolves to nothing renders as
		// nothing rather than leaking its own braces to the browser.
		fallback := strings.TrimSpace(groups[2])

		// {{data}} is the whole document, for handing the page its own state in
		// a <script type="application/json"> block. json.Marshal escapes <, >
		// and & by default, which is what keeps that block from closing early.
		if key == "" {
			encoded, err := json.Marshal(document)

			if err != nil {
				return fallback
			}

			return string(encoded)
		}

		value, ok := p.lookup(document, key)

		if !ok || value == nil || value == "" {
			return fallback
		}

		return html.EscapeString(p.stringify(value))
	})
}

func (p *pageData) lookup(document map[string]any, key string) (any, bool) {
	var current any = document

	for _, segment := range strings.Split(key, ".") {
		node, ok := current.(map[string]any)

		if !ok {
			return nil, false
		}

		current, ok = node[segment]

		if !ok {
			return nil, false
		}
	}

	return current, true
}

func (p *pageData) stringify(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case bool:
		return strconv.FormatBool(typed)
	case float64:
		// JSON has one number type. An integral value reads as an id or a count
		// on the page, so it is rendered without a decimal point.
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		encoded, err := json.Marshal(typed)

		if err != nil {
			return ""
		}

		return string(encoded)
	}
}

// applyPageData fetches the request's loader document and renders it into the
// static file that was matched. It returns the rendered body, or a response
// when the rule asked for the upstream status to be passed through.
func (r *RequestServer) applyPageData(content []byte, headers http.Header) ([]byte, *shttp.Response) {
	loader := r.req.Loader

	if loader == nil || !isHTMLContentType(headers.Get("Content-Type")) {
		return content, nil
	}

	ctx := context.Background()
	renderer := &pageData{loader: loader}

	if r.req.Request != nil {
		ctx = r.req.Context()
		renderer.visitor = pageDataVisitor{
			userAgent: r.req.Header.Get("User-Agent"),
			ip:        r.req.RemoteIP(),
		}
	}

	result := renderer.fetch(ctx)

	if result.err != nil {
		// The document is already on disk and the page still renders with its
		// defaults. Serving it beats answering with an error because a
		// dependency was slow.
		slog.Errorf("page data loader failed for %s: %s", loader.URL, result.err.Error())
		return []byte(renderer.render(string(content), map[string]any{})), nil
	}

	if result.status >= 300 {
		if loader.PassthroughStatus {
			return content, &shttp.Response{Status: result.status}
		}

		// An error body is the API's, not the record's: rendering it would put
		// "Not found" in the page's title.
		return []byte(renderer.render(string(content), map[string]any{})), nil
	}

	return []byte(renderer.render(string(content), result.document)), nil
}
