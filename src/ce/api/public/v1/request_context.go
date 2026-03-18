package publicapiv1

import (
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type RequestContext struct {
	*shttp.RequestContext
	Token *apikey.Token
	EnvID types.ID
	AppID types.ID
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

		// Fill in the EnvID and AppID from the query parameters or request body
		// if they are not present in the token.
		if req.Method == shttp.MethodGet || req.Method == shttp.MethodDelete {
			query := req.Query()
			request.EnvID = getID(key.EnvID, utils.StringToID(query.Get("envId")))
			request.AppID = getID(key.AppID, utils.StringToID(query.Get("appId")))
		} else {
			data := struct {
				EnvID types.ID `json:"envId,string"`
				AppID types.ID `json:"appId,string"`
			}{}

			if req.Method != shttp.MethodGet {
				if isMultipart(req) {
					data.AppID = utils.StringToID(req.FormValue("appId"))
					data.EnvID = utils.StringToID(req.FormValue("envId"))
				} else {
					_ = req.Post(&data)
				}
			}

			request.EnvID = getID(key.EnvID, data.EnvID)
			request.AppID = getID(key.AppID, data.AppID)
		}

		return handler(request)
	}
}

// getID returns the first non-zero ID from the provided list of IDs. If all IDs are zero, it returns zero.
func getID(ids ...types.ID) types.ID {
	for _, id := range ids {
		if id != 0 {
			return id
		}
	}

	return 0
}

func isMultipart(req *shttp.RequestContext) bool {
	return strings.HasPrefix(req.Header.Get("content-type"), "multipart/form-data")
}
