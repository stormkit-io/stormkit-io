package publicapiv1

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth/skauthhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

// databaseIntegrationEnabled mirrors the REST-side gating in services.go —
// the database integration is available only on dev and self-hosted builds.
func databaseIntegrationEnabled() bool {
	return config.IsDevelopment() || config.IsSelfHosted()
}

// ---------------------------------------------------------------------------
// Tool manifest
// ---------------------------------------------------------------------------

type mcpToolDef struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"inputSchema"`
}

func mcpAllTools() []mcpToolDef {
	tools := []mcpToolDef{
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
			Name:        "get_runtime_logs",
			Description: "Return runtime logs (server side rendering and API function output) produced by a deployment. Paginate with afterId/beforeId using the id of the last log entry returned.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"deploymentId": map[string]any{"type": "string", "description": "Deployment to read runtime logs for."},
					"envId":        map[string]any{"type": "string", "description": "Environment the deployment belongs to."},
					"sort":         map[string]any{"type": "string", "enum": []string{"asc", "desc"}, "description": "Sort order by log id. Defaults to 'asc' (oldest first)."},
					"afterId":      map[string]any{"type": "string", "description": "Return logs older than this log id. Use with sort 'desc'."},
					"beforeId":     map[string]any{"type": "string", "description": "Return logs newer than this log id. Use with sort 'asc'."},
				},
				"required":             []string{"deploymentId", "envId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "get_access_logs",
			Description: "Return raw HTTP access logs (one entry per request served) for an environment, newest first. Defaults to the last 24 hours when 'from' is omitted. Paginate by passing pagination.cursor from the previous response.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"envId":    map[string]any{"type": "string", "description": "Environment to read access logs for."},
					"from":     map[string]any{"type": "string", "description": "Only return requests at or after this unix timestamp, in seconds. Defaults to 24 hours ago."},
					"to":       map[string]any{"type": "string", "description": "Only return requests at or before this unix timestamp, in seconds."},
					"domainId": map[string]any{"type": "string", "description": "Only return requests served for this domain."},
					"hostName": map[string]any{"type": "string", "description": "Only return requests for this host name."},
					"clientIp": map[string]any{"type": "string", "description": "Only return requests from this client IP."},
					"method":   map[string]any{"type": "string", "description": "Only return requests with this HTTP method, e.g. GET."},
					"path":     map[string]any{"type": "string", "description": "Only return requests whose path starts with this value."},
					"status":   map[string]any{"type": "string", "description": "Only return requests answered with this HTTP status code."},
					"isBot":    map[string]any{"type": "boolean", "description": "Filter bot traffic in or out. Omit to return both."},
					"cursor":   map[string]any{"type": "string", "description": "Opaque pagination cursor. Pass pagination.cursor from the previous response back verbatim to fetch the next page."},
					"limit":    map[string]any{"type": "number", "description": "How many entries to return per page. Defaults to 100, maximum 1000."},

					"minDurationMs": map[string]any{"type": "number", "description": "Only return requests that took at least this many milliseconds to serve. Use to inspect the latency tail, e.g. 500. Requests logged before durations were recorded are excluded."},
				},
				"required":             []string{"envId"},
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
			Name:        "prioritize_deployment",
			Description: "Move a queued deployment to the front of the build queue. Only deployments that are still waiting to be picked up by a worker can be reordered.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"deploymentId": map[string]any{"type": "string", "description": "Deployment ID to prioritize."},
					"envId":        map[string]any{"type": "string", "description": "Environment the deployment belongs to."},
				},
				"required":             []string{"deploymentId", "envId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "stop_deployment",
			Description: "Stop a running deployment.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"deploymentId": map[string]any{"type": "string", "description": "Deployment ID to stop."},
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
			Name:        "create_app",
			Description: "Create a new application linked to a source-control repository.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"teamId": map[string]any{"type": "string", "description": "Team ID to create the app under. Required."},
					"repo": map[string]any{
						"type":        "string",
						"description": "Repository reference. Accepted formats: full URL (https://github.com/org/repo), Stormkit style (github/org/repo), or bare owner/repo.",
					},
					"displayName": map[string]any{"type": "string", "description": "Human-readable name for the app. Auto-generated if omitted."},
				},
				"required":             []string{"teamId"},
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
					"workDir":            map[string]any{"type": "string", "description": "Working directory relative to the repository root where install/build commands run. Defaults to the repo root."},
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
					"cacheDirs": map[string]any{
						"type":        "array",
						"description": "Directories (relative to the build working directory) restored before install and snapshotted after a successful build. Best for compiler caches like .next/cache or .turbo. Requires a premium or ultimate subscription on Stormkit Cloud; always enabled on self-hosted.",
						"items":       map[string]any{"type": "string"},
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
					"workDir":            map[string]any{"type": "string", "description": "Working directory relative to the repository root where install/build commands run. Defaults to the repo root."},
					"apiFolder":          map[string]any{"type": "string", "description": "Path to the API / serverless functions folder."},
					"apiPathPrefix":      map[string]any{"type": "string", "description": "URL prefix used to route requests to API functions (default: /api)."},
					"serverCmd":          map[string]any{"type": "string", "description": "Command to start the server process (self-hosted only)."},
					"errorFile":          map[string]any{"type": "string", "description": "Custom error page file served instead of 404.html."},
					"headers":            map[string]any{"type": "string", "description": "Custom response headers in Netlify / Caddy format."},
					"headersFile":        map[string]any{"type": "string", "description": "Path to a headers file (relative to repo root)."},
					"redirectsFile":      map[string]any{"type": "string", "description": "Path to a redirects file (relative to repo root)."},
					"previewLinks":       map[string]any{"type": "boolean", "description": "Generate preview links for each deployment."},
					"priorityPattern":    map[string]any{"type": "string", "description": "Regex matched against the commit message of auto-deploys; matching deployments are automatically routed to the priority queue. Leave empty to disable."},
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
					"cacheDirs": map[string]any{
						"type":        "array",
						"description": "Directories (relative to the build working directory) restored before install and snapshotted after a successful build. Best for compiler caches like .next/cache or .turbo. Pass an empty array to disable caching. Requires a premium or ultimate subscription on Stormkit Cloud; always enabled on self-hosted.",
						"items":       map[string]any{"type": "string"},
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
		{
			Name:        "create_domain",
			Description: "Attach a custom domain to an environment. Returns the domain ID and the verification token to use for the DNS TXT record.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"envId":  map[string]any{"type": "string", "description": "Environment ID to attach the domain to."},
					"domain": map[string]any{"type": "string", "description": "Domain name to attach, e.g. 'app.example.com'."},
				},
				"required":             []string{"envId", "domain"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "delete_domain",
			Description: "Remove a custom domain from an environment.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"envId":    map[string]any{"type": "string", "description": "Environment the domain belongs to."},
					"domainId": map[string]any{"type": "string", "description": "ID of the domain to remove (from list_domains or create_domain)."},
				},
				"required":             []string{"envId", "domainId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "list_triggers",
			Description: "Return all periodic triggers configured for an environment.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"envId": map[string]any{"type": "string", "description": "Environment ID."},
				},
				"required":             []string{"envId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "create_trigger",
			Description: "Create a periodic trigger that sends an HTTP request to a URL on a cron schedule (evaluated in UTC). Returns the created trigger including its ID.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"envId":         map[string]any{"type": "string", "description": "Environment ID to attach the trigger to."},
					"cron":          map[string]any{"type": "string", "description": "Cron expression, evaluated in UTC, e.g. '*/5 * * * *'."},
					"status":        map[string]any{"type": "boolean", "description": "Whether the trigger is active. Inactive triggers are not scheduled."},
					"description":   map[string]any{"type": "string", "description": "One-line summary of what the trigger does, e.g. 'Autofill weekly newsletter'. Shown next to the trigger in listings. Max 200 characters, single line."},
					"documentation": map[string]any{"type": "string", "description": "Markdown notes describing what the trigger is for. Displayed in the UI only; never affects execution."},
					"method":        map[string]any{"type": "string", "description": "HTTP method: GET, POST, HEAD, PATCH or DELETE. Defaults to GET."},
					"url":           map[string]any{"type": "string", "description": "http/https URL to call."},
					"payload":       map[string]any{"type": "string", "description": "Request body, sent for non-GET methods."},
					"headers":       map[string]any{"type": "object", "description": "Request headers as a key/value map.", "additionalProperties": map[string]any{"type": "string"}},
				},
				"required":             []string{"envId", "cron", "url"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "update_trigger",
			Description: "Update an existing periodic trigger. The trigger must belong to the given environment. This is a partial update: only the fields you pass are changed and everything else keeps its current value, so pass just what you want to change. To clear a field, pass its empty value explicitly (e.g. an empty string for documentation or payload).",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"envId":         map[string]any{"type": "string", "description": "Environment the trigger belongs to."},
					"triggerId":     map[string]any{"type": "string", "description": "ID of the trigger to update (from list_triggers or create_trigger)."},
					"cron":          map[string]any{"type": "string", "description": "Cron expression, evaluated in UTC."},
					"status":        map[string]any{"type": "boolean", "description": "Whether the trigger is active."},
					"description":   map[string]any{"type": "string", "description": "One-line summary of what the trigger does, e.g. 'Autofill weekly newsletter'. Shown next to the trigger in listings. Max 200 characters, single line."},
					"documentation": map[string]any{"type": "string", "description": "Markdown notes describing what the trigger is for. Displayed in the UI only; never affects execution."},
					"method":        map[string]any{"type": "string", "description": "HTTP method: GET, POST, HEAD, PATCH or DELETE."},
					"url":           map[string]any{"type": "string", "description": "http/https URL to call."},
					"payload":       map[string]any{"type": "string", "description": "Request body, sent for non-GET methods."},
					"headers":       map[string]any{"type": "object", "description": "Request headers as a key/value map.", "additionalProperties": map[string]any{"type": "string"}},
				},
				"required":             []string{"envId", "triggerId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "delete_trigger",
			Description: "Delete a periodic trigger from an environment.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"envId":     map[string]any{"type": "string", "description": "Environment the trigger belongs to."},
					"triggerId": map[string]any{"type": "string", "description": "ID of the trigger to delete."},
				},
				"required":             []string{"envId", "triggerId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "invoke_trigger",
			Description: "Run a periodic trigger immediately, regardless of its schedule or status. Executes synchronously and returns the execution log, which is also stored alongside the scheduled runs.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"envId":     map[string]any{"type": "string", "description": "Environment the trigger belongs to."},
					"triggerId": map[string]any{"type": "string", "description": "ID of the trigger to run."},
				},
				"required":             []string{"envId", "triggerId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "get_trigger_logs",
			Description: "Return the last 25 executions (scheduled or manual) of a periodic trigger, most recent first.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"envId":     map[string]any{"type": "string", "description": "Environment the trigger belongs to."},
					"triggerId": map[string]any{"type": "string", "description": "ID of the trigger to read logs for."},
				},
				"required":             []string{"envId", "triggerId"},
				"additionalProperties": false,
			},
		},
		{
			Name:        "list_teams",
			Description: "Return all teams the authenticated user belongs to.",
			InputSchema: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"additionalProperties": false,
			},
		},
		{
			Name:        "create_team",
			Description: "Create a new team owned by the authenticated user. Enterprise license required. Returns the created team.",
			InputSchema: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string", "description": "Display name for the team."},
				},
				"required":             []string{"name"},
				"additionalProperties": false,
			},
		},
	}

	if authConfigEnabled() {
		tools = append(tools,
			mcpToolDef{
				Name:        "get_auth_config",
				Description: "Return the Stormkit Auth configuration for an environment (session, allowed origins, and OAuth-server settings). Self-hosted only. Secrets are never included.",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"envId": map[string]any{"type": "string", "description": "Environment ID to read the auth configuration for."},
					},
					"required":             []string{"envId"},
					"additionalProperties": false,
				},
			},
			mcpToolDef{
				Name:        "configure_auth",
				Description: "Update the Stormkit Auth configuration for an environment. Self-hosted only. Only the fields you provide are changed; omitted fields keep their current value. Does not manage individual providers or users. Returns the updated configuration.",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"envId":              map[string]any{"type": "string", "description": "Environment ID."},
						"status":             map[string]any{"type": "boolean", "description": "Enable or disable Stormkit Auth for the environment."},
						"successUrl":         map[string]any{"type": "string", "description": "Relative URL the browser is redirected to after a successful sign-in (e.g. /auth/success)."},
						"tokenTtl":           map[string]any{"type": "integer", "description": "Session token lifetime in minutes."},
						"allowedOrigins":     map[string]any{"type": "array", "description": "Origins (scheme + host, no path) allowed to initiate cross-origin sign-in. Replaces the existing list.", "items": map[string]any{"type": "string"}},
						"cookieDomain":       map[string]any{"type": "string", "description": "Parent domain to scope the session cookie to (e.g. .example.com) for sharing across subdomains. Empty for a host-only cookie."},
						"loginUrl":           map[string]any{"type": "string", "description": "App-owned login page the OAuth server redirects unauthenticated users to. Relative path or absolute URL."},
						"oauthServerEnabled": map[string]any{"type": "boolean", "description": "Turn the OAuth 2.1 authorization server (for MCP connectors) on or off. Requires Stormkit Auth enabled."},
						"oauthResourcePath":  map[string]any{"type": "string", "description": "Path the environment serves its MCP endpoint on (e.g. /mcp). Must equal the path in the connector URL."},
						"oauthAllowLoopback": map[string]any{"type": "boolean", "description": "Allow RFC 8252 loopback (localhost) redirects for native/CLI OAuth clients."},
					},
					"required":             []string{"envId"},
					"additionalProperties": false,
				},
			},
		)
	}

	if databaseIntegrationEnabled() {
		tools = append(tools,
			mcpToolDef{
				Name:        "enable_database_integration",
				Description: "Provision a Postgres schema for the environment and store its credentials on the build config. Self-hosted only. Returns the schema name. Use configure_database_integration afterwards to toggle migrations or env-var injection.",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"envId": map[string]any{"type": "string", "description": "Environment ID to enable the database integration for."},
					},
					"required":             []string{"envId"},
					"additionalProperties": false,
				},
			},
			mcpToolDef{
				Name:        "configure_database_integration",
				Description: "Update the database integration flags (migrations, env-var injection) for an environment that already has a schema provisioned. Omitted fields keep their current value.",
				InputSchema: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"envId":             map[string]any{"type": "string", "description": "Environment ID."},
						"injectEnvVars":     map[string]any{"type": "boolean", "description": "Inject DATABASE_URL and friends into the build/runtime env vars."},
						"migrationsEnabled": map[string]any{"type": "boolean", "description": "Run migrations from migrationsFolder on every successful deployment."},
						"migrationsFolder":  map[string]any{"type": "string", "description": "Repository path containing SQL migration files (relative to repo root)."},
					},
					"required":             []string{"envId"},
					"additionalProperties": false,
				},
			},
		)
	}

	return tools
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
	case string:
		// A model that quotes a numeric argument still means the number.
		n, _ := strconv.Atoi(v)
		return n
	}
	return 0
}

// stringPtrArg returns a pointer to the string value at key, or nil when the
// key is absent. Used by patch-style tools that must distinguish "leave
// unchanged" (absent) from "set to empty" (present, "").
func stringPtrArg(args map[string]any, key string) *string {
	if v, ok := args[key].(string); ok {
		return &v
	}

	return nil
}

func boolPtrArg(args map[string]any, key string) *bool {
	if v, ok := args[key].(bool); ok {
		return &v
	}

	return nil
}

func intPtrArg(args map[string]any, key string) *int {
	switch v := args[key].(type) {
	case float64:
		n := int(v)
		return &n
	case int:
		return &v
	}

	return nil
}

// stringSliceArg converts a JSON array argument to []string, or returns nil
// when the key is absent so a patch tool leaves the stored list untouched.
func stringSliceArg(args map[string]any, key string) []string {
	raw, ok := args[key].([]any)

	if !ok {
		return nil
	}

	out := make([]string, 0, len(raw))

	for _, v := range raw {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}

	return out
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

func mcpGetRuntimeLogs(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	if resp := req.withDeploymentID(args); resp != nil {
		return resp
	}

	req.setQuery(map[string]string{
		"sort":     stringArg(args, "sort"),
		"afterId":  stringArg(args, "afterId"),
		"beforeId": stringArg(args, "beforeId"),
	})

	return handlerDeploymentRuntimeLogsGet(req.RequestContext)
}

func mcpGetAccessLogs(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	query := map[string]string{
		"from":     stringArg(args, "from"),
		"to":       stringArg(args, "to"),
		"domainId": stringArg(args, "domainId"),
		"hostName": stringArg(args, "hostName"),
		"clientIp": stringArg(args, "clientIp"),
		"method":   stringArg(args, "method"),
		"path":     stringArg(args, "path"),
		"status":   stringArg(args, "status"),
		"cursor":   stringArg(args, "cursor"),
	}

	if isBot := boolPtrArg(args, "isBot"); isBot != nil {
		query["isBot"] = strconv.FormatBool(*isBot)
	}

	if ms := intArg(args, "minDurationMs"); ms > 0 {
		query["minDurationMs"] = strconv.Itoa(ms)
	}

	if limit := intArg(args, "limit"); limit > 0 {
		query["limit"] = strconv.Itoa(limit)
	}

	req.setQuery(query)

	return handlerAccessLogsGet(req.RequestContext)
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

func mcpStopDeployment(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	if resp := req.withDeploymentID(args); resp != nil {
		return resp
	}

	return handlerDeploymentStop(req.RequestContext)
}

func mcpPrioritizeDeployment(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	if resp := req.withDeploymentID(args); resp != nil {
		return resp
	}

	return handlerDeploymentPrioritize(req.RequestContext)
}

func mcpCreateApp(req *RequestContextMCP, id any, args map[string]any) *shttp.Response {
	if resp := req.withTeamID(args); resp != nil {
		return resp
	}

	provider, ownerSlug := utils.ParseRepoWithProvider(stringArg(args, "repo"))

	body := appCreatePost{
		Repo:        ownerSlug,
		Provider:    provider,
		DisplayName: stringArg(args, "displayName"),
	}

	if resp := req.setBody(id, body); resp != nil {
		return resp
	}

	return handlerAppCreate(req.RequestContext)
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

	// Env var values are masked by Env.JSON/MarshalJSON, so the list response
	// already hides them — no MCP-specific masking needed.
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
		WorkDir:       stringArg(args, "workDir"),
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
	body.CacheDirs, _ = stringArrayArg(args, "cacheDirs")

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

// stringArrayArg reads a JSON string array argument. The second return value
// reports whether the key was present as an array, so callers can tell an
// omitted argument (leave unchanged) from an empty one (clear the list).
func stringArrayArg(args map[string]any, key string) ([]string, bool) {
	raw, ok := args[key].([]any)

	if !ok {
		return nil, false
	}

	out := make([]string, 0, len(raw))

	for _, item := range raw {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}

	return out, true
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
	setString("workDir", &update.WorkDir)
	setString("apiFolder", &update.APIFolder)
	setString("apiPathPrefix", &update.APIPathPrefix)
	setString("serverCmd", &update.ServerCmd)
	setString("errorFile", &update.ErrorFile)
	setString("headers", &update.Headers)
	setString("headersFile", &update.HeadersFile)
	setString("redirectsFile", &update.RedirectsFile)
	setString("priorityPattern", &update.PriorityPattern)
	setBool("autoDeploy", &update.AutoDeploy)
	setBool("autoPublish", &update.AutoPublish)
	setBool("previewLinks", &update.PreviewLinks)

	if m := stringMapArg(args, "envVars"); m != nil {
		update.EnvVars = m
	}

	if r := parseRedirectsArg(args); r != nil {
		update.Redirects = &r
	}

	update.StatusChecks = parseStatusChecksArg(args)

	if dirs, ok := stringArrayArg(args, "cacheDirs"); ok {
		update.CacheDirs = &dirs
	}

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

	// Query the store directly to return a flat list, without the pagination
	// envelope HandlerDomainsList wraps its REST response in.
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

func mcpCreateDomain(req *RequestContextMCP, id any, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	// HandlerDomainAdd reads {"domain": ...} from the request body and validates
	// the format itself, returning a helpful error for empty/invalid values.
	if resp := req.setBody(id, map[string]any{"domain": stringArg(args, "domain")}); resp != nil {
		return resp
	}

	return HandlerDomainAdd(req.RequestContext)
}

func mcpDeleteDomain(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	domainID := stringArg(args, "domainId")

	if utils.StringToID(domainID) == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"domainId must be a numeric ID"}})
	}

	req.setQuery(map[string]string{"domainId": domainID})

	return HandlerDomainDelete(req.RequestContext)
}

func mcpListTriggers(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	return handlerFunctionTriggersGet(req.RequestContext)
}

// triggerBodyFromArgs builds the request body shared by create_trigger and
// update_trigger from the tool arguments. The headers map is forwarded as-is so
// that shttp.Headers unmarshals it the same way the REST handler does.
//
// Only the arguments the caller actually supplied are forwarded: an update is a
// partial one, so filling in a zero value for an absent argument here would
// clear the stored field instead of leaving it alone.
func triggerBodyFromArgs(args map[string]any) map[string]any {
	options := map[string]any{}

	for _, key := range []string{"method", "url", "payload", "headers"} {
		if v, ok := args[key]; ok {
			options[key] = v
		}
	}

	body := map[string]any{}

	for _, key := range []string{"cron", "status", "description", "documentation"} {
		if v, ok := args[key]; ok {
			body[key] = v
		}
	}

	if len(options) > 0 {
		body["options"] = options
	}

	return body
}

func mcpCreateTrigger(req *RequestContextMCP, id any, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	if resp := req.setBody(id, triggerBodyFromArgs(args)); resp != nil {
		return resp
	}

	return handlerFunctionTriggerCreate(req.RequestContext)
}

func mcpUpdateTrigger(req *RequestContextMCP, id any, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	triggerID := stringArg(args, "triggerId")

	if utils.StringToID(triggerID) == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"triggerId must be a numeric ID"}})
	}

	body := triggerBodyFromArgs(args)
	body["id"] = triggerID

	if resp := req.setBody(id, body); resp != nil {
		return resp
	}

	return handlerFunctionTriggerUpdate(req.RequestContext)
}

func mcpDeleteTrigger(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	triggerID := stringArg(args, "triggerId")

	if utils.StringToID(triggerID) == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"triggerId must be a numeric ID"}})
	}

	req.setQuery(map[string]string{"triggerId": triggerID})

	return handlerFunctionTriggerDelete(req.RequestContext)
}

func mcpInvokeTrigger(req *RequestContextMCP, id any, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	triggerID := stringArg(args, "triggerId")

	if utils.StringToID(triggerID) == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"triggerId must be a numeric ID"}})
	}

	if resp := req.setBody(id, map[string]any{"id": triggerID}); resp != nil {
		return resp
	}

	return handlerFunctionTriggerInvoke(req.RequestContext)
}

func mcpGetTriggerLogs(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	triggerID := stringArg(args, "triggerId")

	if utils.StringToID(triggerID) == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"triggerId must be a numeric ID"}})
	}

	req.setQuery(map[string]string{"triggerId": triggerID})

	return handlerFunctionTriggerLogsGet(req.RequestContext)
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

func mcpListTeams(req *RequestContextMCP) *shttp.Response {
	return handlerTeamList(req.RequestContext)
}

func mcpCreateTeam(req *RequestContextMCP, id any, args map[string]any) *shttp.Response {
	if resp := req.setBody(id, TeamCreateRequest{Name: stringArg(args, "name")}); resp != nil {
		return resp
	}

	return handlerTeamCreate(req.RequestContext)
}

func mcpEnableDatabaseIntegration(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	return handlerSchemaSet(req.asAppContext())
}

func mcpConfigureDatabaseIntegration(req *RequestContextMCP, id any, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	// Ship args through as the JSON body — handlerSchemaConfigure already
	// treats missing keys as "leave the stored value untouched" thanks to the
	// pointer fields on SchemaConfigureRequest, so no per-field plumbing is
	// needed here.
	if resp := req.setBody(id, args); resp != nil {
		return resp
	}

	return handlerSchemaConfigure(req.asAppContext())
}

func mcpGetAuthConfig(req *RequestContextMCP, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	return handlerAuthConfigGet(req.RequestContext)
}

func mcpConfigureAuth(req *RequestContextMCP, id any, args map[string]any) *shttp.Response {
	if resp := req.withEnv(args); resp != nil {
		return resp
	}

	// Build a patch from only the keys the caller supplied so unspecified
	// settings keep their stored value (see AuthConfigUpdateRequest).
	body := skauthhandlers.AuthConfigUpdateRequest{
		SuccessURL:         stringPtrArg(args, "successUrl"),
		TTL:                intPtrArg(args, "tokenTtl"),
		Status:             boolPtrArg(args, "status"),
		AllowedOrigins:     stringSliceArg(args, "allowedOrigins"),
		CookieDomain:       stringPtrArg(args, "cookieDomain"),
		LoginURL:           stringPtrArg(args, "loginUrl"),
		OAuthServerEnabled: boolPtrArg(args, "oauthServerEnabled"),
		OAuthResourcePath:  stringPtrArg(args, "oauthResourcePath"),
		OAuthAllowLoopback: boolPtrArg(args, "oauthAllowLoopback"),
	}

	if resp := req.setBody(id, body); resp != nil {
		return resp
	}

	return handlerAuthConfigSet(req.RequestContext)
}
