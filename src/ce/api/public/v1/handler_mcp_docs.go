package publicapiv1

import (
	"bytes"
	"html/template"
	"net/http"
	"sort"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerMCPDocs serves a human-readable HTML reference for the MCP server.
// It is rendered from the live tool manifest (mcpAllTools), so the page always
// reflects exactly what the running build exposes — including tools that are
// only available on self-hosted instances. The endpoint is public so the
// documentation stays discoverable without an API key.
func handlerMCPDocs(req *shttp.RequestContext) *shttp.Response {
	renderer := &mcpDocsRenderer{baseURL: mcpDocsBaseURL(req)}

	html, err := renderer.render()

	if err != nil {
		return &shttp.Response{
			Status: http.StatusInternalServerError,
			Data:   map[string]any{"error": "failed to render documentation"},
			Error:  err,
		}
	}

	return &shttp.Response{
		Status:  http.StatusOK,
		Data:    html,
		Headers: http.Header{"Content-Type": []string{"text/html; charset=utf-8"}},
	}
}

// mcpDocsBaseURL derives the public base URL of the current instance from the
// incoming request, so the page shows the host the caller actually reached
// (api.stormkit.io on Cloud, the custom host on self-hosted).
func mcpDocsBaseURL(req *shttp.RequestContext) string {
	scheme := "https"

	if proto := req.Headers().Get("X-Forwarded-Proto"); proto != "" {
		scheme = proto
	} else if req.Request != nil && req.Request.TLS == nil {
		scheme = "http"
	}

	host := "api.stormkit.io"

	if req.Request != nil && req.Request.Host != "" {
		host = req.Request.Host
	}

	return scheme + "://" + host
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

type mcpDocsParam struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

type mcpDocsTool struct {
	Name        string
	Description string
	Params      []mcpDocsParam
}

type mcpDocsView struct {
	BaseURL         string
	Endpoint        string
	ProtocolVersion string
	Version         string
	Tools           []mcpDocsTool
}

type mcpDocsRenderer struct {
	baseURL string
}

func (r *mcpDocsRenderer) render() (string, error) {
	tmpl, err := template.New("mcp-docs").Parse(mcpDocsTemplate)

	if err != nil {
		return "", err
	}

	var buf bytes.Buffer

	if err := tmpl.Execute(&buf, r.view()); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func (r *mcpDocsRenderer) view() mcpDocsView {
	tools := make([]mcpDocsTool, 0, len(mcpAllTools()))

	for _, def := range mcpAllTools() {
		tools = append(tools, mcpDocsTool{
			Name:        def.Name,
			Description: def.Description,
			Params:      r.params(def.InputSchema),
		})
	}

	return mcpDocsView{
		BaseURL:         r.baseURL,
		Endpoint:        r.baseURL + "/v1/mcp",
		ProtocolVersion: mcpProtocolVersion,
		Version:         config.Get().Version.Tag,
		Tools:           tools,
	}
}

// params flattens a tool's JSON input schema into a display-ordered list:
// required parameters first (in declared order), then the rest alphabetically.
func (r *mcpDocsRenderer) params(schema map[string]any) []mcpDocsParam {
	properties, _ := schema["properties"].(map[string]any)

	if len(properties) == 0 {
		return nil
	}

	required := r.requiredSet(schema["required"])

	params := make([]mcpDocsParam, 0, len(properties))

	for name, raw := range properties {
		prop, _ := raw.(map[string]any)
		typ, _ := prop["type"].(string)
		desc, _ := prop["description"].(string)

		params = append(params, mcpDocsParam{
			Name:        name,
			Type:        typ,
			Description: desc,
			Required:    required[name],
		})
	}

	sort.Slice(params, func(i, j int) bool {
		if params[i].Required != params[j].Required {
			return params[i].Required
		}

		return params[i].Name < params[j].Name
	})

	return params
}

func (r *mcpDocsRenderer) requiredSet(raw any) map[string]bool {
	set := map[string]bool{}

	switch list := raw.(type) {
	case []string:
		for _, name := range list {
			set[name] = true
		}
	case []any:
		for _, name := range list {
			if s, ok := name.(string); ok {
				set[s] = true
			}
		}
	}

	return set
}

const mcpDocsTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Stormkit MCP Server</title>
<style>
:root { color-scheme: light dark; }
body { font: 16px/1.6 -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; max-width: 860px; margin: 0 auto; padding: 2rem 1.25rem 4rem; color: #1a1a1a; }
@media (prefers-color-scheme: dark) { body { color: #e6e6e6; background: #111; } }
h1 { font-size: 1.9rem; margin-bottom: .25rem; }
h2 { font-size: 1.3rem; margin-top: 2.5rem; border-bottom: 1px solid #8884; padding-bottom: .3rem; }
h3 { font-size: 1.05rem; margin-top: 1.75rem; }
code, pre { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: .9em; }
pre { background: #8881; padding: .9rem 1rem; border-radius: 8px; overflow-x: auto; }
code { background: #8882; padding: .12em .4em; border-radius: 4px; }
pre code { background: none; padding: 0; }
.lead { color: #8a8a8a; margin-top: 0; }
.tool { margin-top: 1.5rem; }
.tool h3 { margin-bottom: .3rem; }
.tool p { margin-top: 0; }
table { border-collapse: collapse; width: 100%; margin-top: .5rem; font-size: .9rem; }
th, td { text-align: left; padding: .4rem .6rem; border-bottom: 1px solid #8883; vertical-align: top; }
th { font-weight: 600; }
.req { color: #c0392b; font-weight: 600; }
@media (prefers-color-scheme: dark) { .req { color: #ff7a6b; } }
.muted { color: #8a8a8a; }
</style>
</head>
<body>
<h1>Stormkit MCP Server</h1>
<p class="lead">Deploy and manage Stormkit from any Model Context Protocol client.</p>

<h2>Connecting</h2>
<p><strong>Endpoint:</strong> <code>POST {{ .Endpoint }}</code> (Streamable HTTP, protocol <code>{{ .ProtocolVersion }}</code>). Server version <code>{{ .Version }}</code>.</p>
<p><strong>Authentication:</strong> a Stormkit API key as a Bearer token. Create one under <em>User Settings → API Keys</em>.</p>
<pre><code>Authorization: Bearer SK_xxxxxxxx</code></pre>

<h3>Claude Code plugin</h3>
<pre><code>/plugin marketplace add stormkit-io/stormkit-io
/plugin install stormkit@stormkit</code></pre>
<p>Then export your credentials before launching Claude Code:</p>
<pre><code>export STORMKIT_HOST="{{ .BaseURL }}"
export STORMKIT_API_KEY="SK_xxxxxxxx"</code></pre>

<h3>Manual configuration</h3>
<pre><code>{
  "mcpServers": {
    "stormkit": {
      "type": "http",
      "url": "{{ .Endpoint }}",
      "headers": { "Authorization": "Bearer SK_xxxxxxxx" }
    }
  }
}</code></pre>

<h2>Tools <span class="muted">({{ len .Tools }})</span></h2>
{{ range .Tools }}
<div class="tool">
<h3><code>{{ .Name }}</code></h3>
<p>{{ .Description }}</p>
{{ if .Params }}
<table>
<thead><tr><th>Parameter</th><th>Type</th><th>Description</th></tr></thead>
<tbody>
{{ range .Params }}
<tr>
<td><code>{{ .Name }}</code>{{ if .Required }} <span class="req">*</span>{{ end }}</td>
<td>{{ .Type }}</td>
<td>{{ .Description }}</td>
</tr>
{{ end }}
</tbody>
</table>
{{ else }}
<p class="muted">No parameters.</p>
{{ end }}
</div>
{{ end }}
<p class="muted" style="margin-top:2rem">Parameters marked <span class="req">*</span> are required.</p>
</body>
</html>`
