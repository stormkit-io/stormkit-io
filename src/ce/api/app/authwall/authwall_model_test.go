package authwall_test

import (
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stretchr/testify/suite"
)

type PublicPathsSuite struct {
	suite.Suite
}

func (s *PublicPathsSuite) Test_IsPublicPath() {
	paths := []string{"/api/webhooks/*", "/robots.txt"}

	for path, public := range map[string]bool{
		"/robots.txt":                   true,
		"/api/webhooks/stripe":          true,
		"/api/webhooks/github/push":     true,
		"/api/webhooks/":                true,
		"/api/webhooks":                 false,
		"/api/webhooksx":                false,
		"/robots.txt/":                  false,
		"/Robots.txt":                   false,
		"/admin":                        false,
		"/api/webhooks/../admin":        false,
		"/api/webhooks/./stripe":        false,
		"/api/webhooks/stripe/../../..": false,
	} {
		s.Equal(public, authwall.IsPublicPath(paths, path), path)
	}

	s.False(authwall.IsPublicPath(nil, "/robots.txt"))
}

func (s *PublicPathsSuite) Test_ValidatePublicPaths() {
	paths, msg := authwall.ValidatePublicPaths([]string{" /a ", "/a", "", "/b/*"})

	s.Empty(msg)
	s.Equal([]string{"/a", "/b/*"}, paths)

	for _, invalid := range []string{
		"a",
		"/a*",
		"/*/a",
		"/a/**",
		"/*",
		"/a/../b",
		"/a/./b",
		"/a?b=1",
		"/a#b",
		"/a b",
		"/" + strings.Repeat("a", authwall.PublicPathMaxLength),
	} {
		_, msg := authwall.ValidatePublicPaths([]string{invalid})
		s.NotEmpty(msg, invalid)
	}

	tooMany := []string{}

	for i := 0; i <= authwall.PublicPathsMax; i++ {
		tooMany = append(tooMany, "/p"+strings.Repeat("a", i))
	}

	_, msg = authwall.ValidatePublicPaths(tooMany)
	s.NotEmpty(msg)
}

func TestPublicPathsSuite(t *testing.T) {
	suite.Run(t, new(PublicPathsSuite))
}
