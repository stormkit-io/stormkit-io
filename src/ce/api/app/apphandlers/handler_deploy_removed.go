package apphandlers

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

var deployRemovedTmpl = template.Must(template.New("deployRemoved").Parse(`<!DOCTYPE html>
<html>
	<head>
		<meta charset="utf-8">
		<title>Stormkit | Deploy button removed</title>
		<style>
			body { font-family: sans-serif; background: #0f092b; color: #262525; }
			.wrapper { max-width: 32rem; margin: 10vh auto; padding: 2rem; background: white; border-radius: 5px; line-height: 1.5; }
		</style>
	</head>
	<body>
		<div class="wrapper">
			<h1>Deploy button removed</h1>
			<p>"Deploy with Stormkit" buttons are no longer supported.</p>
			<p>
				Create an app from the <a href="{{.appURL}}">Stormkit dashboard</a>, or deploy
				with an AI agent through the <a href="https://www.stormkit.io/docs/api/mcp">Stormkit MCP server</a>.
			</p>
		</div>
	</body>
</html>
`))

// handlerDeployRemoved answers the removed "Deploy with Stormkit" button
// endpoint, which old badges in third-party READMEs still link to.
func handlerDeployRemoved(req *shttp.RequestContext) *shttp.Response {
	buf := &bytes.Buffer{}

	if err := deployRemovedTmpl.Execute(buf, map[string]string{"appURL": admin.MustConfig().AppURL("")}); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusGone,
		Data:   buf.String(),
		Headers: http.Header{
			"Content-Type": []string{"text/html; charset=utf-8"},
		},
	}
}
