package status

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"text/template"

	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

const page = `<!DOCTYPE html>
<html>
	<head>
		<title>Stormkit API</title>
		<meta name="viewport" content="width=device-width, initial-scale=1" />
		<style>
			html, body {
				font-family: Verdana, Geneva, sans-serif;
				background-color: #0f092b;
				color: white;
				display: flex;
				min-height: 100vh;
				align-items: center;
				justify-content: center;
			}

			.container {
				position: relative;
			}

			.logo-container {
				text-align: center;
				margin-top: 1.5rem;
				opacity: 0.5;
			}

			.logo {
				width: 5rem;
				height: 5rem;
			}

			h1 {
				text-align: center;
			}

			table {
				border-collapse: collapse;
				border-radius: 0.5rem;
				overflow: hidden;
				border: 1px solid rgba(255, 255, 255, 0.1);
			}

			table td {
				padding: 2rem 1rem;
				min-width: 20rem;
				border: 1px solid rgba(255, 255, 255, 0.1);
				background-color: rgba(0, 0, 0, 0.3);
			}

			table td:last-of-type {
				text-align: right;
			}

			@media screen and (max-width: 600px) {
				table td {
					min-width: auto;
				}
			}
		</style>
	</head>
	<body>
		<div class="container">
			<h1>Stormkit API</h1>
			<table>
				<tbody>
					<tr>
						<td>API Status</td>
						<td style="color: green;">OK</td>
					</tr>
					<tr>
						<td>Commit</td>
						<td>{{ .hash }}</td>
					</tr>
					<tr>
						<td>Version</td>
						<td>{{ .version }}</td>
					</tr>
					<tr>
						<td>Self Hosted</td>
						<td>{{ .isSelfHosted }}</td>
					</tr>
				</tbody>
			</table>
			<div class="logo-container">
				<img src="https://www.stormkit.io/stormkit-logo.png" alt="Stormkit Logo" class="logo" />
			</div>
		</div>
	</body>
</html>`

// handlerAPIStatus returns the status for the api.
func handlerAPIStatus(req *shttp.RequestContext) *shttp.Response {
	if req.Method == shttp.MethodHead {
		return &shttp.Response{
			Status: http.StatusOK,
		}
	}

	cfg := config.Get()

	headers := http.Header{}
	headers.Add("Content-Type", "text/html")

	tmpl, err := template.New("page").Parse(page)

	if err != nil {
		return shttp.Error(err)
	}

	var qb strings.Builder

	hash := cfg.Version.Hash

	if len(hash) > 7 {
		hash = hash[:7]
	}

	data := map[string]any{
		"hash":         hash,
		"version":      cfg.Version.Tag,
		"env":          config.Env(),
		"isSelfHosted": fmt.Sprintf("%v", config.IsSelfHosted()),
	}

	if err := tmpl.Execute(&qb, data); err != nil {
		return shttp.Error(err)
	}

	re := regexp.MustCompile(`\s{2,}`)

	return &shttp.Response{
		Status:  http.StatusOK,
		Headers: headers,
		Data:    re.ReplaceAll([]byte(qb.String()), []byte(" ")),
	}
}
