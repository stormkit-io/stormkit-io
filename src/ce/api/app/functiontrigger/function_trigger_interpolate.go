package functiontrigger

import (
	"regexp"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// varRefPattern matches an environment-variable reference in the shell style
// $NAME or ${NAME}, where NAME is a standard identifier. It is used to
// interpolate a trigger's URL, header values and payload at run time.
var varRefPattern = regexp.MustCompile(`\$\{[a-zA-Z_][a-zA-Z0-9_]*\}|\$[a-zA-Z_][a-zA-Z0-9_]*`)

// interpolate replaces every $NAME / ${NAME} reference in s with the matching
// value from vars. A reference with no matching key is left untouched, so an
// unresolved reference passes through literally rather than collapsing to an
// empty string.
func interpolate(s string, vars map[string]string) string {
	if s == "" || len(vars) == 0 {
		return s
	}

	return varRefPattern.ReplaceAllStringFunc(s, func(match string) string {
		name := match[1:] // strip the leading $

		if name[0] == '{' {
			name = name[1 : len(name)-1] // strip the surrounding { }
		}

		if val, ok := vars[name]; ok {
			return val
		}

		return match
	})
}

// Interpolate returns a copy of the options with $NAME / ${NAME} references in
// the URL, header values and payload replaced by the matching environment
// variable. Header keys are left untouched; only values are interpolated. The
// receiver is not mutated, so the stored trigger keeps its raw references and
// secrets are resolved only at run time.
func (o Options) Interpolate(vars map[string]string) Options {
	if len(vars) == 0 {
		return o
	}

	out := o
	out.URL = interpolate(o.URL, vars)

	if len(o.Payload) > 0 {
		out.Payload = []byte(interpolate(string(o.Payload), vars))
	}

	if len(o.Headers) > 0 {
		headers := make(shttp.Headers, len(o.Headers))

		for k, v := range o.Headers {
			headers[k] = interpolate(v, vars)
		}

		out.Headers = headers
	}

	return out
}
