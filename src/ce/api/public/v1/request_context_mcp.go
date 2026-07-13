package publicapiv1

import (
	"bytes"
	"encoding/json"
	"io"
	"net/url"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// RequestContextMCP extends RequestContext with helpers used exclusively by
// MCP tool wrappers. Tool wrappers receive a *RequestContextMCP; when
// delegating to existing REST handlers they pass req.RequestContext directly.
type RequestContextMCP struct {
	*RequestContext
}

// withEnv loads and authorises access to the environment identified by
// args["envId"], mirroring the SCOPE_ENV check in WithAPIKey.
// On success it sets req.Env, req.App, and req.TeamID; on failure it returns
// a non-nil response that should be returned immediately by the caller.
func (req *RequestContextMCP) withEnv(args map[string]any) *shttp.Response {
	envID := utils.StringToID(stringArg(args, "envId"))

	if envID == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"envId is required"}})
	}

	env, err := buildconf.NewStore().EnvironmentByID(req.Context(), envID)

	if err != nil {
		return shttp.Error(err)
	}

	if env == nil {
		return shttp.NotFound()
	}

	myApp, err := app.NewStore().AppByID(req.Context(), env.AppID)

	if err != nil {
		return shttp.Error(err)
	}

	if myApp == nil {
		return shttp.NotFound()
	}

	if !buildconf.NewStore().IsMember(req.Context(), env.ID, req.Token.UserID) {
		return shttp.Forbidden()
	}

	req.Env = env
	req.App = myApp
	req.TeamID = myApp.TeamID

	return nil
}

// withApp loads and authorises access to the application identified by
// args["appId"], mirroring the SCOPE_APP check in WithAPIKey.
// On success it sets req.App and req.TeamID.
func (req *RequestContextMCP) withApp(args map[string]any) *shttp.Response {
	appID := utils.StringToID(stringArg(args, "appId"))

	if appID == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"appId is required"}})
	}

	myApp, err := app.NewStore().AppByID(req.Context(), appID)

	if err != nil {
		return shttp.Error(err)
	}

	if myApp == nil {
		return shttp.NotFound()
	}

	if !team.NewStore().IsMember(req.Context(), req.Token.UserID, myApp.TeamID) {
		return shttp.Forbidden()
	}

	req.App = myApp
	req.TeamID = myApp.TeamID

	return nil
}

// withDeploymentID extracts "deploymentId" from args and sets it as the "id"
// path variable so that handlers using req.Vars()["id"] pick it up correctly.
// Returns a non-nil response on validation failure.
func (req *RequestContextMCP) withDeploymentID(args map[string]any) *shttp.Response {
	id := stringArg(args, "deploymentId")

	if id == "" {
		return shttp.BadRequest(map[string]any{"errors": []string{"deploymentId is required"}})
	}

	if utils.StringToID(id) == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"deploymentId must be a numeric ID"}})
	}

	req.vars = map[string]string{"id": id}
	return nil
}

// withTeamID resolves and authorises the team identified by args["teamId"].
// teamId is required; returns a 400 when it is absent or cannot be parsed.
// Sets req.TeamID on success.
func (req *RequestContextMCP) withTeamID(args map[string]any) *shttp.Response {
	teamID := utils.StringToID(stringArg(args, "teamId"))

	if teamID == 0 {
		return shttp.BadRequest(map[string]any{"errors": []string{"teamId is required"}})
	}

	if !team.NewStore().IsMember(req.Context(), req.Token.UserID, teamID) {
		return shttp.Forbidden()
	}

	req.TeamID = teamID
	return nil
}

// setBody JSON-encodes v and replaces the request body so that handlers that
// call req.Post() consume the tool arguments instead of the MCP envelope.
// id is the JSON-RPC request ID echoed in the error response when marshaling
// fails, per the JSON-RPC 2.0 spec.
func (req *RequestContextMCP) setBody(id any, v any) *shttp.Response {
	b, err := json.Marshal(v)
	if err != nil {
		return jsonRPCError(id, rpcErrInternal, "internal error: failed to marshal tool arguments")
	}

	req.Request.Body = io.NopCloser(bytes.NewReader(b))

	return nil
}

// setQuery sets URL query parameters from a string map and resets the
// parsedURL cache in the embedded RequestContext so that the next call to
// Query() re-parses the updated RawQuery.
func (req *RequestContextMCP) setQuery(params map[string]string) {
	q := url.Values{}

	for k, v := range params {
		if v != "" {
			q.Set(k, v)
		}
	}

	req.Request.URL.RawQuery = q.Encode()
	req.ResetQuery()
}
