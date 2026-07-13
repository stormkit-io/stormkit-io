package snippetshandlers_test

import (
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/snippetshandlers"
	"github.com/stretchr/testify/suite"
)

type HelpersSnippetsSuite struct {
	suite.Suite
}

func (s *HelpersSnippetsSuite) Test_CalculateResetDomains() {
	var snippets []*buildconf.Snippet
	displayName := "my-app"

	// Update all
	snippets = []*buildconf.Snippet{}
	s.Equal(snippetshandlers.CalculateResetDomains(displayName, snippets), []string{})

	// Should update all dev domains and example.org
	snippets = []*buildconf.Snippet{
		{Rules: &buildconf.SnippetRule{Hosts: []string{"*.dev", "example.org"}}},
	}

	s.ElementsMatch(snippetshandlers.CalculateResetDomains(displayName, snippets), []string{"^my-app(?:--\\d+)?", "example.org"})

	// Should update only example.org
	snippets = []*buildconf.Snippet{
		{Rules: &buildconf.SnippetRule{Hosts: []string{"example.org"}}},
	}

	s.Equal(snippetshandlers.CalculateResetDomains(displayName, snippets), []string{
		"example.org",
	})
}

func TestHelpersSnippets(t *testing.T) {
	suite.Run(t, &HelpersSnippetsSuite{})
}
