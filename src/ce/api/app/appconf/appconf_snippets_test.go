package appconf_test

import (
	"testing"

	"github.com/dlclark/regexp2"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stretchr/testify/suite"
)

type AppconfSnippets struct {
	suite.Suite
}

func (s *AppconfSnippets) Test_SnippetRules_RequestPath() {
	snippets := appconf.SnippetsHTML(appconf.Snippets{
		{
			Content:  "S1",
			Location: "body",
			Prepend:  true,
		},
		{
			Content:  "S2",
			Location: "head",
			Prepend:  true,
			Rules: &buildconf.SnippetRule{
				Path: "^/(my|your)/.*/end",
			},
		},
	}, appconf.SnippetFilters{
		RequestPath: "/your/awesome/end",
	})

	s.NotNil(snippets)
	s.Equal(appconf.SnippetInjection{
		BodyPrepend: "S1",
		HeadPrepend: "S2",
	}, *snippets)
}

func (s *AppconfSnippets) Test_SnippetRules_RequestPath_Negate() {
	filters := appconf.Snippets{
		{
			Content:  "S2",
			Location: "head",
			Prepend:  true,
			Rules: &buildconf.SnippetRule{
				PathCompiled: regexp2.MustCompile(`^(?!/home/courses/details|/lx)`, regexp2.None),
			},
		},
	}

	// path => isEmpty?
	paths := map[string]bool{
		"/lx":                   true,
		"/home/courses/details": true,
		"/home/courses":         false,
		"/another-path":         false,
		"/some-other-courses":   false,
		"/some-other-home":      false,
	}

	for p, val := range paths {
		injection := appconf.SnippetsHTML(filters, appconf.SnippetFilters{
			RequestPath: p,
		})

		s.Equal(val, injection.IsEmpty(), "Expected snippet for path %s to be %v", p, val)
	}
}

func (s *AppconfSnippets) Test_Interpolate_ReplacesRequestID() {
	injection := appconf.SnippetsHTML(appconf.Snippets{
		{
			Content:     `<script data-sk-rid="{{SK_REQUEST_ID}}"></script>`,
			Location:    "head",
			Interpolate: true,
		},
	}, appconf.SnippetFilters{RequestID: "abc-123"})

	s.Equal(`<script data-sk-rid="abc-123"></script>`, injection.HeadAppend)
}

func (s *AppconfSnippets) Test_Interpolate_DisabledLeavesTokenVerbatim() {
	injection := appconf.SnippetsHTML(appconf.Snippets{
		{
			Content:     `<script data-sk-rid="{{SK_REQUEST_ID}}"></script>`,
			Location:    "head",
			Interpolate: false,
		},
	}, appconf.SnippetFilters{RequestID: "abc-123"})

	s.Equal(`<script data-sk-rid="{{SK_REQUEST_ID}}"></script>`, injection.HeadAppend)
}

func (s *AppconfSnippets) Test_Interpolate_LeavesUnknownTokensUntouched() {
	injection := appconf.SnippetsHTML(appconf.Snippets{
		{
			// Non-exact spacing and an unknown token must both survive even when
			// the snippet opts into interpolation.
			Content:     `{{ SK_REQUEST_ID }} {{FOO}} {{SK_REQUEST_ID}}`,
			Location:    "head",
			Interpolate: true,
		},
	}, appconf.SnippetFilters{RequestID: "x"})

	s.Equal(`{{ SK_REQUEST_ID }} {{FOO}} x`, injection.HeadAppend)
}

func (s *AppconfSnippets) Test_Interpolate_MultipleSnippetsAndOccurrences() {
	injection := appconf.SnippetsHTML(appconf.Snippets{
		{
			Content:     `a={{SK_REQUEST_ID}};b={{SK_REQUEST_ID}}`,
			Location:    "head",
			Interpolate: true,
		},
		{
			Content:     `c={{SK_REQUEST_ID}}`,
			Location:    "body",
			Interpolate: true,
		},
	}, appconf.SnippetFilters{RequestID: "id"})

	s.Equal(`a=id;b=id`, injection.HeadAppend)
	s.Equal(`c=id`, injection.BodyAppend)
}

func TestAppconfSnippetsSuite(t *testing.T) {
	suite.Run(t, &AppconfSnippets{})
}
