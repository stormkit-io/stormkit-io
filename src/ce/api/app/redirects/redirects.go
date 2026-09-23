package redirects

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type Redirect struct {
	From    string            `json:"from"`
	To      string            `json:"to"`
	Assets  bool              `json:"assets,omitempty"` // Whether to include assets in the wildcard redirect or not
	Status  int               `json:"status,omitempty"`
	Hosts   []string          `json:"hosts,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Data    *Loader           `json:"data,omitempty"`
}

// Loader declares an upstream JSON document to fetch at request time and
// interpolate into the page the rule rewrites to. It turns a single route of an
// otherwise static deployment dynamic without moving the markup out of the
// build: the document stays in the deployment, the data comes from the API.
type Loader struct {
	// URL is the document to fetch. Wildcard captures from the rule's `from`
	// are available as $1, $2 ... so one rule serves every record.
	URL string `json:"url"`

	// PassthroughStatus propagates a non-2xx upstream status to the visitor, so
	// a record the API no longer has answers 404 rather than an empty shell.
	PassthroughStatus bool `json:"passthroughStatus,omitempty"`
}

// Validate checks each redirect rule for correctness and returns a list of
// human-readable error strings. Returns nil when all rules are valid.
func Validate(rules []Redirect) []string {
	var errors []string

	for i, r := range rules {
		prefix := fmt.Sprintf("redirect[%d]", i)

		if strings.TrimSpace(r.From) == "" {
			errors = append(errors, fmt.Sprintf("%s: 'from' is required", prefix))
		}

		if strings.TrimSpace(r.To) == "" {
			errors = append(errors, fmt.Sprintf("%s: 'to' is required", prefix))
		}

		if r.Status != 0 && http.StatusText(r.Status) == "" {
			errors = append(errors, fmt.Sprintf("%s: status %d is not a valid HTTP status code", prefix, r.Status))
		}

		if r.Data != nil {
			if strings.TrimSpace(r.Data.URL) == "" {
				errors = append(errors, fmt.Sprintf("%s: 'data.url' is required", prefix))
			} else if !strings.HasPrefix(r.Data.URL, "https://") && os.Getenv("STORMKIT_PAGE_DATA_INSECURE") != "true" {
				errors = append(errors, fmt.Sprintf("%s: 'data.url' must be an https URL", prefix))
			}

			// A loader interpolates the document the rule rewrites to. An
			// absolute target is proxied and a 3xx never has a body, so neither
			// has a document to interpolate.
			if strings.HasPrefix(r.To, "http") || (r.Status >= 300 && r.Status <= 308) {
				errors = append(errors, fmt.Sprintf("%s: 'data' is only supported on a rewrite to a local path", prefix))
			}
		}
	}

	if len(errors) == 0 {
		return nil
	}

	return errors
}

type MatchArgs struct {
	URL           *url.URL
	HostName      string
	Redirects     []Redirect
	APIPathPrefix string
	APILocation   string
}

type MatchReturn struct {
	Proxy    bool
	Status   int
	Redirect string
	Rewrite  string
	Pattern  string

	// Data carries the rule's loader with its URL resolved against this
	// request, and is set only on a rewrite.
	Data *Loader
}

func Match(args MatchArgs) *MatchReturn {
	url := args.URL
	addr := fmt.Sprintf("%s://%s", url.Scheme, args.HostName)
	apiPath := args.APIPathPrefix

	for _, redirect := range args.Redirects {
		if len(redirect.Hosts) > 0 && !utils.InSliceString(redirect.Hosts, args.HostName) {
			continue
		}

		if redirect.From == "" || redirect.To == "" {
			continue
		}

		isAsset := strings.Contains(url.Path, ".") && !strings.HasSuffix(url.Path, ".html")
		isApi := strings.HasPrefix(url.Path, apiPath) && args.APILocation != ""

		if (isAsset && !redirect.Assets) || isApi {
			continue
		}

		// stormkit.io => www.stormkit.io
		if redirect.From == args.HostName {
			to := strings.Split(redirect.To, "/*")[0]
			target := strings.Replace(addr, redirect.From, to, 1) + url.Path

			if len(url.RawQuery) > 0 {
				target = target + "?" + url.RawQuery
			}

			return &MatchReturn{
				Redirect: target,
				Status:   redirect.Status,
			}
		}

		path := url.RawPath

		if path == "" {
			path = url.Path
		}

		pattern := strings.Replace(redirect.From, "*", "(.*)", -1)
		pattern = strings.TrimRight(strings.TrimLeft(pattern, "^"), "$")
		pattern = fmt.Sprintf("^%s$", pattern)
		matched, _ := regexp.MatchString(pattern, path)

		if matched {
			var target string

			from := strings.Split(redirect.From, "*")[0]

			// There are two ways to replace a string:
			// 1. By using the wildcard: `*`
			// 2. By provided a regexp pattern: "$1/my-text"
			//
			// see TestRedirects function for examples
			if strings.Contains(redirect.To, "*") || !strings.Contains(redirect.To, "$1") {
				to := strings.Split(redirect.To, "*")

				if len(to) >= 2 {
					target = strings.Replace(url.Path, from, to[0], 1)
				} else {
					target = to[0]
				}
			} else {
				re := regexp.MustCompile(pattern)
				target = re.ReplaceAllString(url.Path, redirect.To)
			}

			if len(url.RawQuery) > 0 {
				target = target + "?" + url.RawQuery
			}

			is3xx := (redirect.Status%300) < 8 && redirect.Status != 0 // 300 - 308
			isAbsolute := strings.HasPrefix(redirect.To, "http")

			if is3xx {
				// If the target is an absolute URL leave it as is otherwise add the domain address
				if !isAbsolute {
					target = strings.TrimSuffix(addr, "/") + "/" + strings.TrimPrefix(target, "/")
				}

				return &MatchReturn{
					Redirect: target,
					Status:   redirect.Status,
					Pattern:  pattern,
				}
			}

			if isAbsolute {
				return &MatchReturn{
					Proxy:    true,
					Redirect: target,
					Pattern:  pattern,
				}
			}

			loader, ok := resolveLoader(resolveLoaderParams{
				Loader:  redirect.Data,
				Pattern: pattern,
				Path:    url.Path,
			})

			// A capture that would walk the loader URL out of its own path is not
			// a record this rule serves.
			if !ok {
				continue
			}

			return &MatchReturn{
				Rewrite: target,
				Pattern: pattern,
				Data:    loader,
			}
		}
	}

	return nil
}

type resolveLoaderParams struct {
	Loader  *Loader
	Pattern string
	Path    string
}

var loaderCaptureRef = regexp.MustCompile(`\$(\d+)`)

// resolveLoader expands the wildcard captures of the matched path into the
// loader URL, so a rule written once serves every record it matches. Each
// capture is escaped segment by segment: a visitor's path may add segments to
// the loader URL, never a query, a fragment or a "..". It returns false when a
// capture holds a "." or ".." segment.
func resolveLoader(p resolveLoaderParams) (*Loader, bool) {
	if p.Loader == nil {
		return nil, true
	}

	re, err := regexp.Compile(p.Pattern)

	if err != nil {
		return nil, true
	}

	captures := re.FindStringSubmatch(p.Path)

	for i := 1; i < len(captures); i++ {
		segments := strings.Split(captures[i], "/")

		for j, segment := range segments {
			if segment == "." || segment == ".." {
				return nil, false
			}

			segments[j] = url.PathEscape(segment)
		}

		captures[i] = strings.Join(segments, "/")
	}

	resolved := *p.Loader
	resolved.URL = loaderCaptureRef.ReplaceAllStringFunc(p.Loader.URL, func(ref string) string {
		index, _ := strconv.Atoi(ref[1:])

		if index < len(captures) {
			return captures[index]
		}

		return ""
	})

	return &resolved, true
}
