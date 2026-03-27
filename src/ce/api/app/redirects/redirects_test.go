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

func TestValidate(t *testing.T) {
	suite.Run(t, &ValidateSuite{})
}
