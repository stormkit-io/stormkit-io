package redirects

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
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

			return &MatchReturn{
				Rewrite: target,
				Pattern: pattern,
			}
		}
	}

	return nil
}
