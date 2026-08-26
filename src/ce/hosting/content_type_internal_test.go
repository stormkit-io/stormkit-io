package hosting

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

// IsHTMLContentTypeSuite pins the single normalisation every HTML decision in
// hosting shares (issue #495). Before it existed, snippet injection, analytics,
// cache policy and markdown conversion each spelled the test differently and
// could disagree about the same response.
type IsHTMLContentTypeSuite struct {
	suite.Suite
}

func (s *IsHTMLContentTypeSuite) Test_MatchesPlainAndParameterised() {
	s.True(isHTMLContentType("text/html"))
	s.True(isHTMLContentType("text/html; charset=utf-8"))
	s.True(isHTMLContentType("text/html;charset=UTF-8"))
}

func (s *IsHTMLContentTypeSuite) Test_MatchesRegardlessOfCase() {
	// Media types are case-insensitive (RFC 9110 §8.3.1), and custom header
	// rules pass whatever the deployment wrote through verbatim.
	s.True(isHTMLContentType("Text/HTML"))
	s.True(isHTMLContentType("TEXT/HTML; charset=utf-8"))
	s.True(isHTMLContentType("Text/Html"))
}

func (s *IsHTMLContentTypeSuite) Test_IgnoresSurroundingWhitespace() {
	s.True(isHTMLContentType("  text/html  "))
	s.True(isHTMLContentType("\ttext/html; charset=utf-8"))
}

func (s *IsHTMLContentTypeSuite) Test_RejectsNonHTML() {
	s.False(isHTMLContentType(""))
	s.False(isHTMLContentType("text/plain"))
	s.False(isHTMLContentType("application/json"))
	s.False(isHTMLContentType("text/markdown"))
	s.False(isHTMLContentType("application/xhtml+xml"))
}

func TestIsHTMLContentTypeSuite(t *testing.T) {
	suite.Run(t, new(IsHTMLContentTypeSuite))
}
