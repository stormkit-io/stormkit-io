package functiontrigger

import (
	"context"
	"io"
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// maxLoggedBodySize bounds how much of a target's response is kept in the log
// row. The target is an arbitrary user-supplied host and an unavailable one
// produces several rows per tick, so an uncapped body is a storage amplifier.
const maxLoggedBodySize = 64 * 1024

// loggedBody reads at most maxLoggedBodySize of the response body, marking the
// value when the target sent more than that.
func loggedBody(res *shttp.HTTPResponse) string {
	body, _ := io.ReadAll(io.LimitReader(res.Body, maxLoggedBodySize+1))

	if len(body) > maxLoggedBodySize {
		return string(body[:maxLoggedBodySize]) + "... (truncated)"
	}

	return string(body)
}

type RunParams struct {
	Context   context.Context
	TriggerID types.ID
	// EnvID is the trigger's environment. Requests to the environment's own
	// hosts carry a token that lets them through its Auth Wall.
	EnvID   types.ID
	Method  string
	URL     string
	Headers shttp.Headers
	Payload []byte
}

// Run executes the trigger's HTTP request and returns a log describing the
// request and its response. It does not persist anything.
func Run(p RunParams) (TriggerLog, error) {
	headers := p.Headers.Make()
	req := shttp.NewRequestV2(utils.GetString(p.Method, shttp.MethodGet), p.URL)

	// A redirect would carry the token to wherever it points, so a run that
	// holds one stops at the first response.
	if token := (wallPass{p}).token(); token != "" {
		headers.Set(authwall.HeaderTriggerToken, token)
		req = req.FollowRedirects(false)
	}

	res, err := req.
		Headers(headers).
		Payload(p.Payload).
		Do()

	request := map[string]any{
		"url":     p.URL,
		"method":  p.Method,
		"headers": p.Headers,
		"payload": string(p.Payload),
	}

	var response map[string]any

	if res != nil {
		response = map[string]any{
			"code": res.StatusCode,
			"body": loggedBody(res),
		}
	} else if err != nil {
		response = map[string]any{
			"error": err.Error(),
		}
	}

	// The row's created_at is filled in by the database on insert, but callers
	// that return the log straight to a client need it populated here too.
	return TriggerLog{
		TriggerID: p.TriggerID,
		Request:   request,
		Response:  response,
		CreatedAt: utils.NewUnix(),
	}, err
}

// wallPass issues the token that lets a run through its environment's Auth
// Wall. The token is left out of the run's log.
type wallPass struct {
	p RunParams
}

// token returns the token for the run, or an empty string when the run does
// not need one: the environment has no wall, or the target is not one of the
// environment's own hosts over HTTPS. A token handed to any other host would
// let whoever runs it through the wall.
func (w wallPass) token() string {
	if w.p.EnvID == 0 || w.p.TriggerID == 0 {
		return ""
	}

	target, err := url.Parse(w.p.URL)

	if err != nil || target.Scheme != "https" || target.Hostname() == "" {
		return ""
	}

	ctx := w.p.Context

	if ctx == nil {
		ctx = context.Background()
	}

	cnf, err := authwall.Store().AuthWallConfig(ctx, w.p.EnvID)

	if err != nil {
		slog.Errorf("cannot read the auth wall of trigger %s: %v", w.p.TriggerID, err)
		return ""
	}

	if cnf == nil || cnf.Status == authwall.StatusDisabled {
		return ""
	}

	owned, err := appconf.NewStore().BelongsToEnv(ctx, w.p.EnvID, appconf.ParseHost(strings.ToLower(target.Hostname())))

	if err != nil {
		slog.Errorf("cannot check whether trigger %s targets its environment: %v", w.p.TriggerID, err)
		return ""
	}

	if !owned {
		return ""
	}

	token, err := authwall.Token{EnvID: w.p.EnvID}.Trigger(w.p.TriggerID)

	if err != nil {
		slog.Errorf("cannot issue the auth wall token of trigger %s: %v", w.p.TriggerID, err)
		return ""
	}

	return token
}
