package publicapiv1

import (
	"fmt"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type RequestContext struct {
	*shttp.RequestContext
	Token  *apikey.Token
	Env    *buildconf.Env
	App    *app.App
	TeamID types.ID
}

func (req *RequestContext) License() *admin.License {
	if config.IsSelfHosted() {
		return admin.CurrentLicense()
	}

	if req.Token == nil {
		return &admin.License{}
	}

	var usr *user.User
	var err error

	if req.Token.UserID != 0 {
		usr, err = user.NewStore().UserByID(req.Token.UserID)
	} else if req.Token.TeamID != 0 {
		usr, err = user.NewStore().TeamOwner(req.Context(), req.Token.TeamID)
	} else if req.Token.EnvID != 0 {
		usr, err = user.NewStore().EnvOwner(req.Context(), req.Token.EnvID)
	} else if req.Token.AppID != 0 {
		usr, err = user.NewStore().AppOwner(req.Context(), req.Token.AppID)
	}

	if err != nil {
		slog.Errorf("error fetching user for license check: %s", err.Error())
		return &admin.License{}
	}

	if usr == nil {
		return &admin.License{}
	}

	return user.License(usr)
}

// GetAuditData satisfies the audit.AuditContexter interface without importing
// the audit package (Go structural typing eliminates the import cycle).
func (req *RequestContext) GetAuditData() audit.AuditData {
	d := audit.AuditData{
		Ctx: req.Context(),
	}

	if req.Token != nil {
		d.TokenName = req.Token.Name
	}

	if req.App != nil {
		d.AppID = req.App.ID
		d.TeamID = req.App.TeamID
	}

	if req.TeamID != 0 {
		d.TeamID = req.TeamID
	}

	if req.Env != nil {
		d.EnvID = req.Env.ID
		d.AppID = req.Env.AppID
	}

	return d
}

// asAppContext converts this RequestContext into an *app.RequestContext so that
// internal handlers can be reused without duplicating logic.
func (req *RequestContext) asAppContext() *app.RequestContext {
	envID := types.ID(0)

	if req.Env != nil {
		envID = req.Env.ID
	}

	return &app.RequestContext{
		RequestContext: &user.RequestContext{
			RequestContext: req.RequestContext,
		},
		App:   req.App,
		EnvID: envID,
		Token: req.Token,
	}
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
			request.TeamID = key.TeamID

			if request.TeamID == 0 {
				request.TeamID = getTeamIDFromRequest(req)
			}

			// Validate membership if the key is not already tied to a team.
			if key.TeamID == 0 {
				isMember := team.NewStore().IsMember(req.Context(), key.UserID, request.TeamID)

				if !isMember {
					return shttp.Forbidden()
				}
			}
		case apikey.SCOPE_APP:
			appID := key.AppID

			if appID == 0 {
				appID = getAppIDFromRequest(req)
			}

			if appID == 0 {
				return shttp.NotFound()
			}

			app, err := app.NewStore().AppByID(req.Context(), appID)

			if err != nil {
				return shttp.Error(err)
			}

			if app == nil {
				return shttp.NotFound()
			}

			request.App = app
			request.TeamID = app.TeamID

			// Validate membership if the key is not already tied to an app.
			if key.AppID == 0 {
				if key.TeamID != 0 {
					if app.TeamID != key.TeamID {
						return shttp.Forbidden()
					}
				} else if key.UserID != 0 {
					isMember := team.NewStore().IsMember(req.Context(), key.UserID, app.TeamID)

					if !isMember {
						return shttp.Forbidden()
					}
				}
			}
		case apikey.SCOPE_ENV:
			envID := key.EnvID

			if envID == 0 {
				envID = getEnvIDFromRequest(req)
			}

			env, err := buildconf.NewStore().EnvironmentByID(req.Context(), envID)

			if err != nil {
				return shttp.Error(err, fmt.Sprintf("error while fetching environment in v1: %s", err.Error()))
			}

			if env == nil {
				return shttp.NotFound()
			}

			app, err := app.NewStore().AppByID(req.Context(), env.AppID)

			if err != nil {
				return shttp.Error(err, fmt.Sprintf("error while fetching app for environment in v1: %s", err.Error()))
			}

			if app == nil {
				return shttp.NotFound()
			}

			request.Env = env
			request.App = app
			request.TeamID = app.TeamID

			// Validate membership if the key is not already tied to an environment.
			if key.EnvID == 0 {
				if key.AppID != 0 && key.AppID != env.AppID {
					return shttp.Forbidden()
				} else if key.TeamID != 0 {
					if app.TeamID != key.TeamID {
						return shttp.Forbidden()
					}

					request.App = app
					request.TeamID = app.TeamID
				} else if key.UserID != 0 {
					isMember := buildconf.NewStore().IsMember(req.Context(), env.ID, key.UserID)

					if !isMember {
						return shttp.Forbidden()
					}
				}
			}
		}

		return handler(request)
	}
}

func isMultipart(req *shttp.RequestContext) bool {
	return strings.HasPrefix(req.Header.Get("content-type"), "multipart/form-data")
}

func getEnvIDFromRequest(req *shttp.RequestContext) types.ID {
	if req.Method == shttp.MethodGet || req.Method == shttp.MethodDelete {
		return utils.StringToID(req.Query().Get("envId"))
	}

	data := struct {
		EnvId types.ID `json:"envId,string"`
	}{}

	if req.Method != shttp.MethodGet {
		if isMultipart(req) {
			data.EnvId = utils.StringToID(req.FormValue("envId"))
		} else {
			_ = req.Post(&data)
		}
	}

	return data.EnvId
}

func getAppIDFromRequest(req *shttp.RequestContext) types.ID {
	if req.Method == shttp.MethodGet || req.Method == shttp.MethodDelete {
		return utils.StringToID(req.Query().Get("appId"))
	}

	data := struct {
		AppID types.ID `json:"appId,string"`
	}{}

	if req.Method != shttp.MethodGet {
		if isMultipart(req) {
			data.AppID = utils.StringToID(req.FormValue("appId"))
		} else {
			_ = req.Post(&data)
		}
	}

	return data.AppID
}

func getTeamIDFromRequest(req *shttp.RequestContext) types.ID {
	if req.Method == shttp.MethodGet || req.Method == shttp.MethodDelete {
		return utils.StringToID(req.Query().Get("teamId"))
	}

	data := struct {
		TeamID types.ID `json:"teamId,string"`
	}{}

	if isMultipart(req) {
		data.TeamID = utils.StringToID(req.FormValue("teamId"))
	} else {
		_ = req.Post(&data)
	}

	return data.TeamID
}
