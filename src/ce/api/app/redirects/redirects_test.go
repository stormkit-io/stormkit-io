package redirects_test

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stretchr/testify/suite"
)

type RedirectsSuite struct {
	suite.Suite
}

func (s *RedirectsSuite) Test_Redirect_Rewrite() {
	u := &url.URL{
		Scheme: "https",
		Path:   "/my-path",
		Host:   "stormkit.io",
	}

	match := redirects.Match(redirects.MatchArgs{
		URL:      u,
		HostName: "stormkit.io",
		Redirects: []redirects.Redirect{
			{From: "/my-path", To: "/my-new-path"},
		},
	})

	s.NotNil(match)
	s.Equal("/my-new-path", match.Rewrite)
}

func (s *RedirectsSuite) Test_Redirect_Extensions() {
	u := &url.URL{
		Scheme: "https",
		Path:   "/MyAwesomeFile.xsd",
		Host:   "stormkit.io",
	}

	match := redirects.Match(redirects.MatchArgs{
		URL: u,
		Redirects: []redirects.Redirect{
			{From: "/*.xsd", To: "/$1.xsd", Assets: true},
		},
	})

	s.NotNil(match)
	s.Equal("/MyAwesomeFile.xsd", match.Rewrite)
}

func (s *RedirectsSuite) Test_Redirect_TrailingSlash() {
	u := &url.URL{
		Scheme: "https",
		Path:   "/my-path",
		Host:   "stormkit.io",
	}

	match := redirects.Match(redirects.MatchArgs{
		URL:      u,
		HostName: "stormkit.io",
		Redirects: []redirects.Redirect{
			{From: "/my-path", To: "/my-path/", Status: http.StatusFound},
		},
	})

	s.NotNil(match)
	s.Equal("https://stormkit.io/my-path/", match.Redirect)
	s.Equal(http.StatusFound, match.Status)
}

func (s *RedirectsSuite) Test_Redirect_MatchHost() {
	u := &url.URL{
		Scheme: "https",
		Path:   "/my-path",
		Host:   "stormkit.io",
	}

	match := redirects.Match(redirects.MatchArgs{
		URL:      u,
		HostName: "stormkit.io",
		Redirects: []redirects.Redirect{
			{From: "/my-path", To: "/my-path/", Status: http.StatusFound, Hosts: []string{"example.org"}},
		},
	})

	s.Nil(match)

	match = redirects.Match(redirects.MatchArgs{
		URL:      u,
		HostName: "stormkit.io",
		Redirects: []redirects.Redirect{
			{From: "/my-path", To: "/my-path/", Status: http.StatusFound, Hosts: []string{"stormkit.io"}},
		},
	})

	s.NotNil(match)
	s.Equal("https://stormkit.io/my-path/", match.Redirect)
	s.Equal(http.StatusFound, match.Status)
}

func (s *RedirectsSuite) Test_Redirect_Rewrite_WithLoader() {
	u := &url.URL{
		Scheme: "https",
		Path:   "/v/abc123",
		Host:   "www.example.com",
	}

	match := redirects.Match(redirects.MatchArgs{
		URL:      u,
		HostName: "www.example.com",
		Redirects: []redirects.Redirect{
			{
				From: "/v/*",
				To:   "/videos.html",
				Data: &redirects.Loader{URL: "https://api.example.com/v/$1.json"},
			},
		},
	})

	s.Require().NotNil(match)
	s.Equal("/videos.html", match.Rewrite)
	s.Require().NotNil(match.Data)
	s.Equal("https://api.example.com/v/abc123.json", match.Data.URL)
}

// A visitor's path can add segments to the loader URL, but not a query or a
// fragment: those arrive percent-encoded and must stay that way.
func (s *RedirectsSuite) Test_Redirect_Loader_EscapesCaptures() {
	rules := []redirects.Redirect{
		{
			From:   "/v/*",
			To:     "/videos.html",
			Assets: true,
			Data:   &redirects.Loader{URL: "https://api.example.com/v/$1.json"},
		},
	}

	cases := map[string]string{
		"/v/2024/my-video": "https://api.example.com/v/2024/my-video.json",
		"/v/a%3Fx=1":       "https://api.example.com/v/a%3Fx=1.json",
		"/v/a%23frag":      "https://api.example.com/v/a%23frag.json",
		"/v/a%20b":         "https://api.example.com/v/a%20b.json",
	}

	for path, expected := range cases {
		u, err := url.Parse("https://www.example.com" + path)
		s.Require().NoError(err)

		match := redirects.Match(redirects.MatchArgs{URL: u, HostName: "www.example.com", Redirects: rules})

		s.Require().NotNil(match, path)
		s.Require().NotNil(match.Data, path)
		s.Equal(expected, match.Data.URL, path)
	}
}

// A capture that climbs out of the loader's path does not match the rule, so
// the request never reaches the API.
func (s *RedirectsSuite) Test_Redirect_Loader_RejectsDotSegments() {
	rules := []redirects.Redirect{
		{
			From:   "/v/*",
			To:     "/videos.html",
			Assets: true,
			Data:   &redirects.Loader{URL: "https://api.example.com/v/$1.json"},
		},
	}

	for _, path := range []string{"/v/..%2F..%2Fadmin", "/v/a/../../admin", "/v/./x", "/v/.."} {
		u, err := url.Parse("https://www.example.com" + path)
		s.Require().NoError(err)

		s.Nil(redirects.Match(redirects.MatchArgs{URL: u, HostName: "www.example.com", Redirects: rules}), path)
	}
}

// The rule carries the loader; the match carries it resolved. Mutating one
// request's copy must not rewrite the configuration every other request reads.
func (s *RedirectsSuite) Test_Redirect_Loader_DoesNotMutateRule() {
	rule := redirects.Redirect{
		From: "/v/*",
		To:   "/videos.html",
		Data: &redirects.Loader{URL: "https://api.example.com/v/$1.json"},
	}

	for _, path := range []string{"/v/first", "/v/second"} {
		match := redirects.Match(redirects.MatchArgs{
			URL:       &url.URL{Scheme: "https", Path: path, Host: "www.example.com"},
			HostName:  "www.example.com",
			Redirects: []redirects.Redirect{rule},
		})

		s.Require().NotNil(match.Data)
		s.Equal("https://api.example.com/v"+path[2:]+".json", match.Data.URL)
	}

	s.Equal("https://api.example.com/v/$1.json", rule.Data.URL)
}

func (s *RedirectsSuite) Test_Redirect_Loader_NotSetOnProxy() {
	match := redirects.Match(redirects.MatchArgs{
		URL:      &url.URL{Scheme: "https", Path: "/v/abc123", Host: "www.example.com"},
		HostName: "www.example.com",
		Redirects: []redirects.Redirect{
			{
				From: "/v/*",
				To:   "https://api.example.com/v/$1",
				Data: &redirects.Loader{URL: "https://api.example.com/v/$1.json"},
			},
		},
	})

	s.Require().NotNil(match)
	s.True(match.Proxy)
	s.Nil(match.Data)
}

func TestRedirects(t *testing.T) {
	suite.Run(t, &RedirectsSuite{})
}

type ValidateSuite struct {
	suite.Suite
}

func (s *ValidateSuite) Test_Valid() {
	errs := redirects.Validate([]redirects.Redirect{
		{From: "/old", To: "/new"},
		{From: "/old2", To: "/new2", Status: http.StatusMovedPermanently},
	})

	s.Nil(errs)
}

func (s *ValidateSuite) Test_Valid_EmptySlice() {
	s.Nil(redirects.Validate(nil))
	s.Nil(redirects.Validate([]redirects.Redirect{}))
}

func (s *ValidateSuite) Test_MissingFrom() {
	errs := redirects.Validate([]redirects.Redirect{
		{From: "", To: "/new"},
	})

	s.Require().Len(errs, 1)
	s.Contains(errs[0], "redirect[0]")
	s.Contains(errs[0], "'from' is required")
}

func (s *ValidateSuite) Test_MissingTo() {
	errs := redirects.Validate([]redirects.Redirect{
		{From: "/old", To: ""},
	})

	s.Require().Len(errs, 1)
	s.Contains(errs[0], "redirect[0]")
	s.Contains(errs[0], "'to' is required")
}

func (s *ValidateSuite) Test_InvalidStatus() {
	errs := redirects.Validate([]redirects.Redirect{
		{From: "/old", To: "/new", Status: 999},
	})

	s.Require().Len(errs, 1)
	s.Contains(errs[0], "redirect[0]")
	s.Contains(errs[0], "999")
}

func (s *ValidateSuite) Test_MultipleErrors() {
	errs := redirects.Validate([]redirects.Redirect{
		{From: "", To: ""},
		{From: "/ok", To: "/ok"},
		{From: "/bad", To: "/bad", Status: 0}, // status 0 means unset — valid
		{From: "", To: "/new2", Status: 1},
	})

	s.Require().Len(errs, 4) // redirect[0] missing from+to, redirect[3] missing from + invalid status
	s.Contains(errs[0], "redirect[0]")
	s.Contains(errs[1], "redirect[0]")
	s.Contains(errs[2], "redirect[3]")
	s.Contains(errs[3], "redirect[3]")
}

func (s *ValidateSuite) Test_Loader_Valid() {
	s.Nil(redirects.Validate([]redirects.Redirect{
		{From: "/v/*", To: "/videos.html", Data: &redirects.Loader{URL: "https://api.example.com/v/$1.json"}},
	}))
}

func (s *ValidateSuite) Test_Loader_RequiresHTTPSURL() {
	errs := redirects.Validate([]redirects.Redirect{
		{From: "/v/*", To: "/videos.html", Data: &redirects.Loader{}},
		{From: "/w/*", To: "/videos.html", Data: &redirects.Loader{URL: "http://api.example.com/w/$1.json"}},
	})

	s.Require().Len(errs, 2)
	s.Contains(errs[0], "'data.url' is required")
	s.Contains(errs[1], "must be an https URL")
}

// A self-hosted instance that relaxed the loader for its private network can
// save the same rule through the API that it can ship in redirects.json.
func (s *ValidateSuite) Test_Loader_AllowsHTTPWhenInsecure() {
	s.T().Setenv("STORMKIT_PAGE_DATA_INSECURE", "true")

	s.Nil(redirects.Validate([]redirects.Redirect{
		{From: "/v/*", To: "/videos.html", Data: &redirects.Loader{URL: "http://api.internal/v/$1.json"}},
	}))
}

// A proxied or redirected rule has no document of its own to interpolate.
func (s *ValidateSuite) Test_Loader_RejectedOnProxyAndRedirect() {
	errs := redirects.Validate([]redirects.Redirect{
		{From: "/v/*", To: "https://api.example.com/v/$1", Data: &redirects.Loader{URL: "https://api.example.com/v/$1.json"}},
		{From: "/w/*", To: "/videos.html", Status: 301, Data: &redirects.Loader{URL: "https://api.example.com/w/$1.json"}},
	})

	s.Require().Len(errs, 2)
	s.Contains(errs[0], "only supported on a rewrite to a local path")
	s.Contains(errs[1], "only supported on a rewrite to a local path")
}

func TestValidate(t *testing.T) {
	suite.Run(t, &ValidateSuite{})
}
