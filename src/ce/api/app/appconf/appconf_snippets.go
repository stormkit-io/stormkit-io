package appconf

import (
	"strings"
)

type SnippetInjection struct {
	HeadAppend  string `json:"headAppend,omitempty"`
	HeadPrepend string `json:"headPrepend,omitempty"`
	BodyAppend  string `json:"bodyAppend,omitempty"`
	BodyPrepend string `json:"bodyPrepend,omitempty"`
}

// SnippetVarRequestID is the per-request system variable replaced with the server
// request id in snippets that opt into interpolation. It is the first (and so far
// only) entry of a closed registry of system variables; user-authored {{...}} that
// is not a defined system variable is left untouched.
const SnippetVarRequestID = "{{SK_REQUEST_ID}}"

type SnippetFilters struct {
	RequestPath string
	RequestID   string
}

// IsEmpty returns true if all locations are empty.
func (sn *SnippetInjection) IsEmpty() bool {
	return sn.HeadAppend == "" && sn.HeadPrepend == "" && sn.BodyAppend == "" && sn.BodyPrepend == ""
}

// From reads the given snippet object, filters the disabled ones
// and joins the snippets.
func SnippetsHTML(sn Snippets, filters ...SnippetFilters) *SnippetInjection {
	if len(sn) == 0 {
		return nil
	}

	f := SnippetFilters{}

	if len(filters) > 0 {
		f = filters[0]
	}

	s := &SnippetInjection{}

	headAppend := []string{}
	headPrepend := []string{}
	bodyAppend := []string{}
	bodyPrepend := []string{}

	for _, snippet := range sn {
		if snippet.Rules != nil && snippet.Rules.PathCompiled != nil {
			if match, _ := snippet.Rules.PathCompiled.MatchString(f.RequestPath); !match {
				continue
			}
		}

		content := snippet.Content

		if snippet.Interpolate {
			content = interpolateSnippetVars(content, f)
		}

		if snippet.Location == "head" {
			if snippet.Prepend {
				headPrepend = append(headPrepend, content)
			} else {
				headAppend = append(headAppend, content)
			}
		} else if snippet.Location == "body" {
			if snippet.Prepend {
				bodyPrepend = append(bodyPrepend, content)
			} else {
				bodyAppend = append(bodyAppend, content)
			}
		}
	}

	s.HeadAppend = strings.Join(headAppend, "")
	s.HeadPrepend = strings.Join(headPrepend, "")
	s.BodyAppend = strings.Join(bodyAppend, "")
	s.BodyPrepend = strings.Join(bodyPrepend, "")

	return s
}

// interpolateSnippetVars replaces defined system variables in a snippet's content.
// Only the closed registry of tokens is substituted; any other {{...}} is left as-is.
// Values come from the server (e.g. a request UUID), never from user input, so no
// HTML-escaping is required for the current registry.
func interpolateSnippetVars(content string, f SnippetFilters) string {
	if strings.Contains(content, SnippetVarRequestID) {
		content = strings.ReplaceAll(content, SnippetVarRequestID, f.RequestID)
	}

	return content
}
