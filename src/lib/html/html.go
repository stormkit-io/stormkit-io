package html

import (
	"bytes"
	"fmt"
	"html/template"

	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

const htmlTemplate = `<!DOCTYPE html>
<html>
	<head>
		<meta charset="utf-8" />
		<meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no" />
		<meta name="robots" content="noindex">
		<title>{{ .page_title }}</title>
		<link rel="stylesheet" href="https://fonts.googleapis.com/css?family=Roboto:400,700">
		<style>
			*, *::before, *::after {
				box-sizing: border-box;
			}
			body {
				background: rgb(15, 9, 43);
				color: white;
				font-family: "Roboto", sans-serif;
				min-height: 100vh;
				display: flex;
				align-items: center;
				justify-content: center;
				line-height: 1.25;
			}
			.wrapper {
				margin: 0 auto;
				max-width: 800px;
				width: 100%;
			}
			.container {
				background-color: rgba(0, 0, 0, 0.3);
				border-radius: 1rem;
				padding: 4rem;
			}
			h1 {
				margin: 0;
				margin-bottom: 0.5rem;
				color: white;
			}
			h3 {
				color: white;
				margin: 0;
				margin-bottom: 3rem;
				font-weight: normal;
				font-size: 0.9rem;
				color: #9a9a9a;
			}
			h1, h3 {
				text-align: center;
			}
			.mb-1 {
				margin-bottom: 1rem;
			}
			.mb-3 {
				margin-bottom: 3rem;
			}
			label {
				display: block;
				margin-bottom: 0.5rem;
				color: #9a9a9a;
				font-size: 0.77rem;
			}
			.text-left {
				text-align: left;
			}
			.text-center {
				text-align: center;
			}
			.text-input {
				flex: 1;
				background: rgba(255, 255, 255, 0.1);
				padding: 0.75rem;
				border: 1px solid black;
				color: white;
			}
			.form-group {
				display: flex;
				flex-direction: column;
				margin-bottom: 1.5rem;
			}
			.form-group.error {
				margin-top: 1.5rem;
				background-color: #460e0e;
				padding: 1rem;
				border-radius: 0.25rem;
			}
			.form-submit {
				text-align: center;
			}
			.submit-button {
				background-color: #78193b;
				color: white;
				padding: 0.75rem 2rem;
				border-radius: 0.5rem;
				cursor: pointer;
				transition: filter 0.2s;
				border: 1px solid black;
			}
			.submit-button:hover {
				filter: brightness(1.25);
			}
			.code {
				padding: 1rem;
				margin-bottom: 1rem;
				font-family: Monospace;
				border-top: 1px solid #9a9a9a;
				background: rgba(255, 255, 255, 0.1);
			}
			@media only screen and (max-width: 450px) {
				.container {
					padding: 1.25rem;
				}
				h1 {
					font-size: 1.3rem;
				}
				h3 {
					font-size: 0.9rem;
				}
			}
			a.secondary {
				color: #f6005c;
			}
			footer {
				margin-top: 2rem;
				text-align: center;
			}
		</style>
		{{ template "page_head" . }}
	</head>
	<body>
		<div class="wrapper">
			{{ template "page_content" . }}
		</div>
	</body>
</html>`

type RenderArgs struct {
	PageTitle   string
	PageHead    string
	PageContent string
	ContentData map[string]any
}

// MustRender renders the page content and returns an error string if it fails.
func MustRender(args RenderArgs) []byte {
	content, err := Render(args)

	if err != nil {
		return []byte(fmt.Sprintf("Error while rendering page: %s", err.Error()))
	}

	return content
}

func Render(args RenderArgs) ([]byte, error) {
	var buf bytes.Buffer

	// Define templates in the main template
	mainTmpl := template.Must(template.New("html").Parse(htmlTemplate))
	template.Must(mainTmpl.New("page_content").Parse(args.PageContent))
	template.Must(mainTmpl.New("page_head").Parse(args.PageHead))

	data := map[string]any{
		"page_title": utils.GetString(args.PageTitle, "Stormkit - Application"),
	}

	// Merge ContentData into main data
	for k, v := range args.ContentData {
		data[k] = v
	}

	if err := mainTmpl.Execute(&buf, data); err != nil {
		return []byte("Error while parsing template"), err
	}

	return buf.Bytes(), nil
}

var Templates = map[string]string{
	"error": `
	<h1>Whoops! We got something wrong.</h1>
	<h3>This usually occurs when your application has an error and nothing catches it.<br />We did our best to show you a trace to help you out.</h3>
	<div class="container wrapper">
		<div>
			<h3 class="text-left mb-1">Error message</h3>
			<div class="code mb-3">{{ .error_msg }}</div>
			<div class="mb-3"></div>
			<h3 class="text-left mb-1">Error stack</h3>
			<div class="code mb-3">{{ if .stack_trace }}{{ .stack_trace }}{{ else }}No stack trace available{{ end }}</div>
		</div>
	</div>
	<footer>
		{{ if .runtime_logs_url }}
		You can view additional logs under your <a href="{{ .runtime_logs_url }}" class="secondary">runtime logs</a>.
		{{ else }}
		You can view additional logs under your application runtime logs.
		{{ end }}
	</footer>`,

	"login": `
	<form method="POST" action="{{ .api_host }}/auth-wall/login" class="container">
		<input type="hidden" name="token" value="{{ .token }}" />
		<input type="hidden" name="envId" value="{{ .env_id }}" />
		<h1>{{ .title }}</h1>
		<h3>Login with your credentials to access the deployment</h3>
		<div>
			<div class="form-group">
				<label for="email">Email</label>
				<input class="text-input" type="email" id="email" name="email" autofocus required>
			</div>
			<div class="form-group">
				<label for="password">Password</label>
				<input class="text-input" type="password" id="password" name="password" required>
			</div>
			<div class="form-submit">
				<button class="submit-button" type="submit">Login</button>
			</div>
			{{ if not .token }}
			<div class="form-group">Token generation failed. Please retry and contact us if the problem persists.</div>
			{{ end }}
			{{ if .error }}
			<div class="form-group error">{{ .error }}</div>
			{{ end }}
		</div>
	</form>`,

	"404": `
	<div class="container">
		<h1>4 oh 4</h1>
		<h3>Whoops! We've got nothing under this link.<br/>This usually occurs when the deployment is removed, or it never existed before.</h3>
		<footer>
			<a href="{{ .app_url }}" class="secondary">Click here</a> to go back to the application.
		</footer>
	</div>`,
}
