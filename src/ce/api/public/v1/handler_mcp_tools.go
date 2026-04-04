package publicapiv1

import (
	"fmt"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"gopkg.in/guregu/null.v3"
)

// ---------------------------------------------------------------------------
// Tool manifest
// ---------------------------------------------------------------------------

type mcpToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func mcpAllTools() []mcpToolDef {
	return []mcpToolDef{
		{
			Name:        "deploy",
			Description: "Trigger a new deployment for the given environment. Returns the deployment object including its ID.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"envId":   map[string]any{"type": "string", "description": "ID of the environment to deploy."},
					"branch":  map[string]any{"type": "string", "description": "Git branch to deploy. Defaults to the environment's configured branch."},
					"publish": map[string]any{"type": "boolean", "description": "Publish the deployment immediately after a successful build."},
				},
				"required":             []string{"envId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "get_deployment",
			Description: "Return metadata and status for a deployment. Poll until status is 'success' or 'failed'.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"deploymentId": map[string]any{"type": "string", "description": "Deployment ID returned by deploy."},
					"envId":        map[string]any{"type": "string", "description": "Environment the deployment belongs to."},
					"logs":         map[string]any{"type": "boolean", "description": "Include build logs in the response."},
				},
				"required":             []string{"deploymentId", "envId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "publish_deployment",
			Description: "Publish a successfully built deployment, making it live.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"deploymentId": map[string]any{"type": "string", "description": "Deployment ID to publish."},
					"envId":        map[string]any{"type": "string", "description": "Environment the deployment belongs to."},
				},
				"required":             []string{"deploymentId", "envId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "delete_deployment",
			Description: "Delete a deployment and its associated artifacts.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"deploymentId": map[string]any{"type": "string", "description": "Deployment ID to delete."},
					"envId":        map[string]any{"type": "string", "description": "Environment the deployment belongs to."},
				},
				"required":             []string{"deploymentId", "envId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "restart_deployment",
			Description: "Restart a failed deployment. Only deployments with status 'failed' can be restarted.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"deploymentId": map[string]any{"type": "string", "description": "Deployment ID to restart."},
					"envId":        map[string]any{"type": "string", "description": "Environment the deployment belongs to."},
				},
				"required":             []string{"deploymentId", "envId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "list_deployments",
			Description: "Return a paginated list of deployments for the given environment. Use hasNextPage and increment 'from' to paginate.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"envId":  map[string]any{"type": "string", "description": "ID of the environment to list deployments for."},
					"from":   map[string]any{"type": "integer", "description": "Pagination offset (default 0)."},
					"branch": map[string]any{"type": "string", "description": "Filter by branch name."},
				},
				"required":             []string{"envId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "list_apps",
			Description: "Return a paginated list of applications scoped to a team. Use hasNextPage and increment 'from' to paginate.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"teamId":      map[string]any{"type": "string", "description": "Team ID to scope the listing. Required."},
					"from":        map[string]any{"type": "integer", "description": "Pagination offset (default 0)."},
					"repo":        map[string]any{"type": "string", "description": "Exact match on repository path, e.g. 'github/org/repo'."},
					"displayName": map[string]any{"type": "string", "description": "Exact match on display name."},
				},
				"required":             []string{"teamId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "list_environments",
			Description: "Return all environments configured for an application. Returns up to 50 environments.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"appId": map[string]any{"type": "string", "description": "Application ID."},
				},
				"required":             []string{"appId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "create_environment",
			Description: "Create a new environment for an application.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"appId":              map[string]any{"type": "string", "description": "Application ID."},
					"name":               map[string]any{"type": "string", "description": "Environment name, e.g. 'staging'."},
					"branch":             map[string]any{"type": "string", "description": "Default git branch."},
					"buildCmd":           map[string]any{"type": "string", "description": "Build command, e.g. 'npm run build'."},
					"installCmd":         map[string]any{"type": "string", "description": "Install command, e.g. 'npm install'."},
					"distFolder":         map[string]any{"type": "string", "description": "Client output directory, e.g. 'dist'."},
					"serverFolder":       map[string]any{"type": "string", "description": "Server output directory for self-hosted deployments."},
					"serverCmd":          map[string]any{"type": "string", "description": "Command to start the server process (self-hosted only)."},
					"apiFolder":          map[string]any{"type": "string", "description": "Path to the API / serverless functions folder."},
					"apiPathPrefix":      map[string]any{"type": "string", "description": "URL prefix used to route requests to API functions (default: /api)."},
					"errorFile":          map[string]any{"type": "string", "description": "Custom error page file served instead of 404.html."},
					"headers":            map[string]any{"type": "string", "description": "Custom response headers in Netlify / Caddy format."},
					"headersFile":        map[string]any{"type": "string", "description": "Path to a headers file (relative to repo root)."},
					"redirectsFile":      map[string]any{"type": "string", "description": "Path to a redirects file (relative to repo root)."},
					"autoDeploy":         map[string]any{"type": "boolean", "description": "Automatically deploy on every push to the configured branch."},
					"autoDeployBranches": map[string]any{"type": "string", "description": "Comma-separated branch patterns that trigger auto-deploy."},
					"autoDeployCommits":  map[string]any{"type": "string", "description": "Regex pattern for commit messages that trigger auto-deploy."},
					"autoPublish":        map[string]any{"type": "boolean", "description": "Automatically publish every successful deployment."},
					"previewLinks":       map[string]any{"type": "boolean", "description": "Generate preview links for each deployment."},
					"envVars":            map[string]any{"type": "object", "description": "Environment variables injected at build and runtime.", "additionalProperties": map[string]any{"type": "string"}},
					"redirects": map[string]any{
						"type": "array", "description": "Redirect / rewrite rules.",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"from":    map[string]any{"type": "string"},
								"to":      map[string]any{"type": "string"},
								"status":  map[string]any{"type": "integer"},
								"assets":  map[string]any{"type": "boolean"},
								"hosts":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
								"headers": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
							},
							"required": []string{"from", "to"},
						},
					},
					"statusChecks": map[string]any{
						"type": "array", "description": "Commands executed after a successful deployment to verify it.",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"name":        map[string]any{"type": "string"},
								"cmd":         map[string]any{"type": "string"},
								"description": map[string]any{"type": "string"},
							},
							"required": []string{"name", "cmd"},
						},
					},
				},
				"required":             []string{"appId", "name", "branch"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "update_environment",
			Description: "Update configuration or environment variables for an existing environment.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"envId":              map[string]any{"type": "string", "description": "Environment ID to update."},
					"name":               map[string]any{"type": "string", "description": "New environment name."},
					"branch":             map[string]any{"type": "string", "description": "Default git branch."},
					"autoDeploy":         map[string]any{"type": "boolean", "description": "Automatically deploy on every push to the configured branch."},
					"autoDeployBranches": map[string]any{"type": "string", "description": "Comma-separated branch patterns that trigger auto-deploy."},
					"autoDeployCommits":  map[string]any{"type": "string", "description": "Regex pattern for commit messages that trigger auto-deploy."},
					"autoPublish":        map[string]any{"type": "boolean", "description": "Automatically publish every successful deployment."},
					"buildCmd":           map[string]any{"type": "string", "description": "Build command, e.g. 'npm run build'."},
					"installCmd":         map[string]any{"type": "string", "description": "Install command, e.g. 'npm install'."},
					"distFolder":         map[string]any{"type": "string", "description": "Client output directory, e.g. 'dist'."},
					"apiFolder":          map[string]any{"type": "string", "description": "Path to the API / serverless functions folder."},
					"apiPathPrefix":      map[string]any{"type": "string", "description": "URL prefix used to route requests to API functions (default: /api)."},
					"serverCmd":          map[string]any{"type": "string", "description": "Command to start the server process (self-hosted only)."},
					"serverFolder":       map[string]any{"type": "string", "description": "Server output directory for self-hosted deployments."},
					"errorFile":          map[string]any{"type": "string", "description": "Custom error page file served instead of 404.html."},
					"headers":            map[string]any{"type": "string", "description": "Custom response headers in Netlify / Caddy format."},
					"headersFile":        map[string]any{"type": "string", "description": "Path to a headers file (relative to repo root)."},
					"redirectsFile":      map[string]any{"type": "string", "description": "Path to a redirects file (relative to repo root)."},
					"previewLinks":       map[string]any{"type": "boolean", "description": "Generate preview links for each deployment."},
					"envVars":            map[string]any{"type": "object", "description": "Environment variables to set or update.", "additionalProperties": map[string]any{"type": "string"}},
					"redirects": map[string]any{
						"type":        "array",
						"description": "Redirect / rewrite rules.",
						"items": map[string]any{
							"type":     "object",
							"required": []string{"from", "to"},
							"properties": map[string]any{
								"from":    map[string]any{"type": "string"},
								"to":      map[string]any{"type": "string"},
								"status":  map[string]any{"type": "integer"},
								"assets":  map[string]any{"type": "boolean"},
								"hosts":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
								"headers": map[string]any{"type": "object", "additionalProperties": map[string]any{"type": "string"}},
							},
						},
					},
					"statusChecks": map[string]any{
						"type":        "array",
						"description": "Commands executed after a successful deployment to verify it.",
						"items": map[string]any{
							"type":     "object",
							"required": []string{"name", "cmd"},
							"properties": map[string]any{
								"name":        map[string]any{"type": "string"},
								"cmd":         map[string]any{"type": "string"},
								"description": map[string]any{"type": "string"},
							},
						},
					},
				},
				"required":             []string{"envId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "list_domains",
			Description: "Return all custom domains configured for an environment.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"envId": map[string]any{"type": "string", "description": "Environment ID."},
				},
				"required":             []string{"envId"},
				"additionalProperties": false,
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Argument helpers
// ---------------------------------------------------------------------------

func stringArg(args map[string]any, key string) string {
	v, _ := args[key].(string)
	return v
}

func boolArg(args map[string]any, key string) bool {
	v, _ := args[key].(bool)
	return v
}

func intArg(args map[string]any, key string) int {
	switch v := args[key].(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0
}

func stringMapArg(args map[string]any, key string) map[string]string {
	raw, ok := args[key].(map[string]any)

	if !ok {
		return nil
	}

	out := make(map[string]string, len(raw))

	for k, v := range raw {
		if s, ok := v.(string); ok {
			out[k] = s
		}
	}

	return out
}

// ---------------------------------------------------------------------------
// Tool wrappers — each sets up req and delegates to an existing handler
// ---------------------------------------------------------------------------

func mcpDeploy(req *RequestContextMCP, id any, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	resp := req.setBody(id, DeploymentCreateRequest{
		Branch:  stringArg(args, "branch"),
		Publish: boolArg(args, "publish"),
	})

	if resp != nil {
		return resp
	}

	return handlerDeploymentCreate(req.RequestContext)
}

func mcpGetDeployment(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	if resp := req.withDeploymentID(args); resp != nil {
		return resp
	}

	if boolArg(args, "logs") {
		req.setQuery(map[string]string{"logs": "true"})
	}

	return handlerDeploymentGet(req.RequestContext)
}

func mcpPublishDeployment(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	if resp := req.withDeploymentID(args); resp != nil {
		return resp
	}

	return handlerDeploymentPublish(req.RequestContext)
}

func mcpDeleteDeployment(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	if resp := req.withDeploymentID(args); resp != nil {
		return resp
	}

	return handlerDeploymentDelete(req.RequestContext)
}

func mcpRestartDeployment(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	if resp := req.withDeploymentID(args); resp != nil {
		return resp
	}

	return handlerDeploymentRestart(req.RequestContext)
}

func mcpListApps(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withTeamID(args); resp != nil {
		return resp
	}

	req.setQuery(map[string]string{
		"from":        fmt.Sprintf("%d", intArg(args, "from")),
		"repo":        stringArg(args, "repo"),
		"displayName": stringArg(args, "displayName"),
	})

	return handlerAppList(req.RequestContext)
}

func mcpListEnvironments(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withApp(args); resp != nil {
		return resp
	}

	return handlerEnvList(req.RequestContext)
}

func mcpCreateEnvironment(req *RequestContextMCP, id any, args map[string]any) *shttp.Response {
	if resp := req.withApp(args); resp != nil {
		return resp
	}

	body := EnvAddRequest{
		Name:          stringArg(args, "name"),
		Branch:        stringArg(args, "branch"),
		BuildCmd:      stringArg(args, "buildCmd"),
		InstallCmd:    stringArg(args, "installCmd"),
		DistFolder:    stringArg(args, "distFolder"),
		ServerFolder:  stringArg(args, "serverFolder"),
		ServerCmd:     stringArg(args, "serverCmd"),
		APIFolder:     stringArg(args, "apiFolder"),
		APIPathPrefix: stringArg(args, "apiPathPrefix"),
		ErrorFile:     stringArg(args, "errorFile"),
		Headers:       stringArg(args, "headers"),
		HeadersFile:   stringArg(args, "headersFile"),
		RedirectsFile: stringArg(args, "redirectsFile"),
		AutoDeploy:    boolArg(args, "autoDeploy"),
		AutoPublish:   boolArg(args, "autoPublish"),
		EnvVars:       stringMapArg(args, "envVars"),
	}

	if v := stringArg(args, "autoDeployBranches"); v != "" {
		body.AutoDeployBranches = null.StringFrom(v)
	}

	if v := stringArg(args, "autoDeployCommits"); v != "" {
		body.AutoDeployCommits = null.StringFrom(v)
	}

	if raw, ok := args["previewLinks"].(bool); ok {
		body.PreviewLinks = null.BoolFrom(raw)
	}

	if body.Redirects = parseRedirectsArg(args); body.Redirects == nil {
		body.Redirects = []redirects.Redirect{}
	}

	body.StatusChecks = parseStatusChecksArg(args)

	resp := req.setBody(id, body)

	if resp != nil {
		return resp
	}

	return handlerEnvAdd(req.RequestContext)
}

func parseRedirectsArg(args map[string]any) []redirects.Redirect {
	raw, ok := args["redirects"].([]any)

	if !ok {
		return nil
	}

	out := make([]redirects.Redirect, 0, len(raw))

	for _, item := range raw {
		m, ok := item.(map[string]any)

		if !ok {
			continue
		}

		r := redirects.Redirect{
			From:   stringArgMap(m, "from"),
			To:     stringArgMap(m, "to"),
			Assets: boolArgMap(m, "assets"),
		}

		if s, ok := m["status"].(float64); ok {
			r.Status = int(s)
		}

		if hosts, ok := m["hosts"].([]any); ok {
			for _, h := range hosts {
				if hs, ok := h.(string); ok {
					r.Hosts = append(r.Hosts, hs)
				}
			}
		}

		if hdrs, ok := m["headers"].(map[string]any); ok {
			r.Headers = make(map[string]string, len(hdrs))
			for k, v := range hdrs {
				if vs, ok := v.(string); ok {
					r.Headers[k] = vs
				}
			}
		}

		out = append(out, r)
	}

	return out
}

func parseStatusChecksArg(args map[string]any) []buildconf.StatusCheck {
	raw, ok := args["statusChecks"].([]any)

	if !ok {
		return nil
	}

	out := make([]buildconf.StatusCheck, 0, len(raw))

	for _, item := range raw {
		m, ok := item.(map[string]any)

		if !ok {
			continue
		}

		out = append(out, buildconf.StatusCheck{
			Name:        stringArgMap(m, "name"),
			Cmd:         stringArgMap(m, "cmd"),
			Description: stringArgMap(m, "description"),
		})
	}

	return out
}

// stringArgMap and boolArgMap are like stringArg/boolArg but operate on a
// plain map[string]any instead of the top-level args map.
func stringArgMap(m map[string]any, key string) string {
	v, _ := m[key].(string)
	return v
}

func boolArgMap(m map[string]any, key string) bool {
	v, _ := m[key].(bool)
	return v
}

func mcpUpdateEnvironment(req *RequestContextMCP, id any, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	update := EnvUpdateRequest{}

	setString := func(key string, dst **string) {
		if _, ok := args[key]; ok {
			v := stringArg(args, key)
			*dst = &v
		}
	}
	setBool := func(key string, dst **bool) {
		if v, ok := args[key].(bool); ok {
			*dst = &v
		}
	}

	setString("name", &update.Name)
	setString("branch", &update.Branch)
	setString("autoDeployBranches", &update.AutoDeployBranches)
	setString("autoDeployCommits", &update.AutoDeployCommits)
	setString("buildCmd", &update.BuildCmd)
	setString("installCmd", &update.InstallCmd)
	setString("distFolder", &update.DistFolder)
	setString("apiFolder", &update.APIFolder)
	setString("apiPathPrefix", &update.APIPathPrefix)
	setString("serverCmd", &update.ServerCmd)
	setString("serverFolder", &update.ServerFolder)
	setString("errorFile", &update.ErrorFile)
	setString("headers", &update.Headers)
	setString("headersFile", &update.HeadersFile)
	setString("redirectsFile", &update.RedirectsFile)
	setBool("autoDeploy", &update.AutoDeploy)
	setBool("autoPublish", &update.AutoPublish)
	setBool("previewLinks", &update.PreviewLinks)

	if m := stringMapArg(args, "envVars"); m != nil {
		update.EnvVars = m
	}

	update.Redirects = parseRedirectsArg(args)
	update.StatusChecks = parseStatusChecksArg(args)

	resp := req.setBody(id, update)

	if resp != nil {
		return resp
	}

	return handlerEnvUpdate(req.RequestContext)
}

func mcpListDomains(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	// domainhandlers.HandlerDomainsList uses a different RequestContext type
	// (app.RequestContext from app.WithAPIKey), so we call the store directly.
	domains, err := buildconf.DomainStore().Domains(req.Context(), buildconf.DomainFilters{
		EnvID: req.Env.ID,
	})

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data:   map[string]any{"domains": domains},
	}
}

func mcpListDeployments(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	req.setQuery(map[string]string{
		"from":   fmt.Sprintf("%d", intArg(args, "from")),
		"branch": stringArg(args, "branch"),
	})

	return handlerDeploymentList(req.RequestContext)
}
