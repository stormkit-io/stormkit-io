package publicapiv1

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"unicode"

	"github.com/adhocore/gronx"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// functionTriggerRequest is the body shared by create and update. Every field
// but the ID is a pointer so that "not mentioned" is distinguishable from
// "set to the zero value": an update applies only the fields it carries and
// leaves the rest of the trigger alone, and a field is cleared by sending its
// empty value explicitly.
type functionTriggerRequest struct {
	ID            types.ID `json:"id,string"`
	Cron          *string  `json:"cron"`
	Description   *string  `json:"description"`
	Documentation *string  `json:"documentation"`
	Status        *bool    `json:"status"`
	Options       *struct {
		Headers *shttp.Headers `json:"headers"`
		Method  *string        `json:"method"`
		Payload *string        `json:"payload"`
		URL     *string        `json:"url"`
	} `json:"options"`
}

// applyTo overlays the fields the request carries onto a trigger, leaving every
// omitted field at its current value.
func (r *functionTriggerRequest) applyTo(ft *functiontrigger.FunctionTrigger) {
	if r.Cron != nil {
		ft.Cron = *r.Cron
	}

	// A whitespace-only description would otherwise render as a blank line
	// under the trigger's URL in the listing.
	if r.Description != nil {
		ft.Description = strings.TrimSpace(*r.Description)
	}

	if r.Documentation != nil {
		ft.Documentation = *r.Documentation
	}

	if r.Status != nil {
		ft.Status = *r.Status
	}

	if r.Options == nil {
		return
	}

	if r.Options.Method != nil {
		ft.Options.Method = *r.Options.Method
	}

	if r.Options.URL != nil {
		ft.Options.URL = *r.Options.URL
	}

	if r.Options.Payload != nil {
		ft.Options.Payload = []byte(*r.Options.Payload)
	}

	if r.Options.Headers != nil {
		ft.Options.Headers = *r.Options.Headers
	}
}

// maxDocumentationLength bounds the free-form notes. The column is plain text
// and every trigger listing returns the field in full, so it is capped at a
// size that is generous for a runbook and far short of a paste-the-logs
// accident.
const maxDocumentationLength = 64 * 1024

// maxDescriptionLength keeps the description a label rather than a second
// documentation field: it is rendered inline next to the trigger in listings,
// where anything longer stops being scannable.
const maxDescriptionLength = 200

// hasDisplayControlRune reports whether s carries a rune that changes how the
// rest of the string renders rather than adding a glyph of its own. The
// description is shown inline next to the trigger's URL and returned verbatim
// to MCP clients, so a bidi override or an escape sequence could misrepresent
// which endpoint a trigger calls.
func hasDisplayControlRune(s string) bool {
	for _, r := range s {
		if unicode.IsControl(r) ||
			unicode.Is(unicode.Cf, r) ||
			unicode.Is(unicode.Zl, r) ||
			unicode.Is(unicode.Zp, r) {
			return true
		}
	}

	return false
}

// validateFunctionTrigger checks the trigger as it will be stored, so that an
// update is validated against the merged record rather than the partial body
// that produced it.
func validateFunctionTrigger(ft *functiontrigger.FunctionTrigger) map[string]string {
	errors := map[string]string{}

	if !gronx.New().IsValid(ft.Cron) {
		errors["cron"] = "Invalid cron format"
	}

	if len(ft.Documentation) > maxDocumentationLength {
		errors["documentation"] = fmt.Sprintf(
			"Documentation cannot exceed %d characters", maxDocumentationLength,
		)
	}

	if len(ft.Description) > maxDescriptionLength {
		errors["description"] = fmt.Sprintf(
			"Description cannot exceed %d characters", maxDescriptionLength,
		)
	} else if strings.ContainsAny(ft.Description, "\r\n") {
		errors["description"] = "Description must be a single line"
	} else if hasDisplayControlRune(ft.Description) {
		errors["description"] = "Description must not contain control characters"
	}

	parsedURL, err := url.Parse(ft.Options.URL)

	if err != nil ||
		parsedURL.Host == "" ||
		parsedURL.Scheme == "" ||
		!strings.EqualFold(parsedURL.Scheme, "http") && !strings.EqualFold(parsedURL.Scheme, "https") {
		errors["url"] = "Invalid URL"
	}

	if len(errors) == 0 {
		return nil
	}

	return errors
}

func handlerFunctionTriggerCreate(req *RequestContext) *shttp.Response {
	tf := &functionTriggerRequest{}

	if err := req.Post(tf); err != nil {
		return shttp.Error(err)
	}

	record := &functiontrigger.FunctionTrigger{EnvID: req.Env.ID}
	tf.applyTo(record)

	if errs := validateFunctionTrigger(record); errs != nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data:   errs,
		}
	}

	if err := functiontrigger.NewStore().Insert(req.Context(), record); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusCreated,
		Data: map[string]any{
			"trigger": record.ToMap(),
		},
	}
}
