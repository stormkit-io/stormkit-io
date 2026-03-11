package publicapiv1

import (
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type RequestContext struct {
	*shttp.RequestContext
	Token *apikey.Token
}

type Opts struct {
	MinimumScope string // apikey.SCOPE_*
}

func getOpts(opts ...*Opts) *Opts {
	o := &Opts{}

	if len(opts) == 1 && opts[0] != nil {
		o = opts[0]
	}

	return o
}

func WithAPIKey(handler func(*RequestContext) *shttp.Response, opts ...*Opts) shttp.RequestFunc {
	options := getOpts(opts...)

	return func(req *shttp.RequestContext) *shttp.Response {
		token := strings.Replace(req.Headers().Get("Authorization"), "Bearer ", "", 1)
		request := &RequestContext{
			RequestContext: req,
		}

		if !strings.Contains(token, "SK_") {
			return shttp.Forbidden()
		}

		key, err := apikey.NewStore().APIKey(req.Context(), token)

		if err != nil {
			return shttp.Error(err)
		}

		// This is an invalid key. A key needs to have either an app ID, user ID or team ID.
		if key == nil || (key.UserID == 0 && key.AppID == 0 && key.TeamID == 0) {
			return shttp.Forbidden()
		}

		request.Token = key

		switch options.MinimumScope {
		case apikey.SCOPE_USER:
			if key.UserID == 0 {
				return shttp.Forbidden()
			}
		case apikey.SCOPE_TEAM:
			if key.TeamID == 0 && key.UserID == 0 {
				return shttp.Forbidden()
			}
		case apikey.SCOPE_APP:
			if key.AppID == 0 && key.TeamID == 0 && key.UserID == 0 {
				return shttp.Forbidden()
			}
		case apikey.SCOPE_ENV:
			if (key.EnvID == 0 && key.AppID == 0) && key.TeamID == 0 && key.UserID == 0 {
				return shttp.Forbidden()
			}
		}

		return handler(request)
	}
}
