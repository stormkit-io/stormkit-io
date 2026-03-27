package buildconf

import (
	"context"
	"crypto/sha256"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dlclark/regexp2"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type SnippetRule struct {
	Hosts        []string `json:"hosts,omitempty"`
	Path         string   `json:"path,omitempty"` // Accepts POSIX Regexp
	PathCompiled *regexp2.Regexp
}

// Scan implements the Scanner interface.
func (sr *SnippetRule) Scan(value any) error {
	if value != nil {
		return json.Unmarshal(value.([]byte), &sr)
	}

	return nil
}

// Value implements the Sql Driver interface.
func (sr *SnippetRule) Value() (driver.Value, error) {
	if sr == nil {
		return nil, nil
	}

	return json.Marshal(sr)
}

// Snippet represents a snippet.
type Snippet struct {
	// ID represents the internal id of the snippet. This is not a real auto_increment ID.
	// There is an internal counter stored alongside the Snippets object. Since we use
	// a `jsonb` column in the database, we cannot use the traditional IDs. We need these
	// IDs to facilitate REST operations.
	ID types.ID `json:"id,string"`

	// AppID contains the snippet application id.
	AppID types.ID `json:"appId,omitempty"`

	// EnvID contains the snippet environment id.
	EnvID types.ID `json:"envId,omitempty"`

	// Enabled specifies whether the snippet is enabled or not.
	Enabled bool `json:"enabled"`

	// Prepend specifies whether the snippet is prepended to the parent.
	// If this value is true, it will be inserted as the first child,
	// otherwise appended as the last child.
	Prepend bool `json:"prepend"`

	// Content is the snippet content.
	Content string `json:"content"`

	// Where to insert this snippet: head | body.
	Location string `json:"location"`

	// Rules contains the name of the domain that this snippet should be used.
	Rules *SnippetRule `json:"rules,omitempty"`

	// Title is the snippet title. It's used only by the Stormkit UI to provide
	// a meaningful description for the user.
	Title string `json:"title"`
}

// Snippets to be injected into the document.
type Snippets struct {
	Head   []Snippet `json:"head"`
	Body   []Snippet `json:"body"`
	LastID int64     `json:"lastId,omitempty"`
}

// Scan implements the Scanner interface.
func (s *Snippets) Scan(value any) error {
	if value != nil {
		if b, ok := value.([]byte); ok {
			json.Unmarshal(b, s)
		}
	}

	return nil
}

// Value implements the Sql Driver interface.
func (s *Snippets) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}

	if len(s.Body) == 0 && len(s.Head) == 0 {
		return nil, nil
	}

	return json.Marshal(s)
}

func (s *Snippet) ContentHash() string {
	h := sha256.New()
	h.Write([]byte(s.Content))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// JSON returns a plain map representation of the snippet suitable for JSON responses.
func (s *Snippet) JSON() map[string]any {
	var rules map[string]any

	if s.Rules != nil {
		rules = map[string]any{
			"hosts": s.Rules.Hosts,
			"path":  s.Rules.Path,
		}
	}

	return map[string]any{
		"id":       s.ID.String(),
		"location": s.Location,
		"prepend":  s.Prepend,
		"enabled":  s.Enabled,
		"title":    s.Title,
		"content":  s.Content,
		"rules":    rules,
	}
}

const SnippetLocationHead = "head"
const SnippetLocationBody = "body"

// ValidateSnippet validates a snippet's fields and trims whitespace in-place.
// Returns a slice of human-readable error messages; returns nil when the snippet is valid.
func ValidateSnippet(snippet *Snippet) []string {
	var errs []string

	if snippet.Location != SnippetLocationHead && snippet.Location != SnippetLocationBody {
		errs = append(errs, "Location must be either 'head' or 'body'.")
	}

	snippet.Title = strings.TrimSpace(snippet.Title)
	snippet.Content = strings.TrimSpace(snippet.Content)

	if snippet.Title == "" {
		errs = append(errs, "Snippet title is a required field.")
	}

	if snippet.Content == "" {
		errs = append(errs, "Snippet content is a required field.")
	}

	if snippet.Rules != nil && snippet.Rules.Path != "" {
		if _, err := regexp2.Compile(snippet.Rules.Path, regexp2.None); err != nil {
			errs = append(errs, "Snippet path must be a valid regular expression.")
		}
	}

	return errs
}

// ValidateSnippetDomains checks that every host referenced in the given snippets
// exists as a domain for the environment. Returns an error listing any missing hosts.
func ValidateSnippetDomains(snippets []*Snippet, envID types.ID) error {
	hosts := []string{}

	for _, snippet := range snippets {
		if snippet.Rules != nil && len(snippet.Rules.Hosts) > 0 {
			for _, host := range snippet.Rules.Hosts {
				if host != "*.dev" {
					hosts = append(hosts, host)
				}
			}
		}
	}

	if len(hosts) == 0 {
		return nil
	}

	missingHosts, err := SnippetsStore().MissingHosts(context.Background(), hosts, envID)

	if err != nil {
		slog.Errorf("error while fetching missing hosts: %s", err.Error())
		return err
	}

	if len(missingHosts) == 0 {
		return nil
	}

	return fmt.Errorf("Invalid or missing domain name(s): %s", strings.Join(missingHosts, ", "))
}
