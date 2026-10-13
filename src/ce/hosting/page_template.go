package hosting

import (
	"bytes"
	"html/template"
)

// pageTemplate renders a deployment's own HTML file as an html/template. The
// file is the app's markup, so it is trusted as a template; the data is not,
// and html/template escapes it for the context it lands in — text, attribute,
// URL or script. Data loaders and the OAuth sign-in page share it, so both
// accept the same syntax and functions.
type pageTemplate struct {
	// Body is the template source.
	Body string

	// Funcs are added to the built-in functions (default).
	Funcs template.FuncMap
}

// render executes the template against data. A missing key renders as its zero
// value rather than "<no value>".
func (t pageTemplate) render(data any) ([]byte, error) {
	funcs := template.FuncMap{"default": t.fallback}

	for name, fn := range t.Funcs {
		funcs[name] = fn
	}

	tmpl, err := template.New("page").
		Option("missingkey=zero").
		Funcs(funcs).
		Parse(t.Body)

	if err != nil {
		return nil, err
	}

	var out bytes.Buffer

	if err := tmpl.Execute(&out, data); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// fallback backs the template's default function: {{.title | default "Videos"}}.
// It fires on an absent, null or empty value and never on false or zero, which
// are real answers — unlike the built-in or, which treats them as empty.
func (t pageTemplate) fallback(fallback, value any) any {
	if value == nil || value == "" {
		return fallback
	}

	return value
}
