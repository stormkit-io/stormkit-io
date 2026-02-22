package app

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// RequestContext is the argument that the AppHandler handler receives.
// Handlers using this endpoint are authenticated handlers.
//
//   - The App field will always be populated for successful requests
//   - The EnvID field will be made available if the request contains an envId or
//     the information was extracted from an API key.
type RequestContext struct {
	*user.RequestContext

	// App represents a pure app model.
	App *App

	// MyApp is an extended version of an App.
	// If you want the details to be fetched from the
	// database you need to pass the WithDetails options
	// in the request handler.
	MyApp *MyApp

	// EnvID represents the environment id. This is an optional field
	// which gets populated only under certain circumstances.
	EnvID types.ID

	// Token represents the API Token used in the request (if any).
	Token *apikey.Token
}

// Response returns a new app response.
type Response struct {
	App *MyApp `json:"app"`
}

// NewResponse returns a new App response.
func NewResponse(app *MyApp) *shttp.Response {
	return &shttp.Response{
		Status: http.StatusOK,
		Data: &Response{
			App: app,
		},
	}
}

type Opts struct {
	Env   bool // Require environment
	App   bool // Require app
	Admin bool // Require admin access
}

func getOpts(opts ...*Opts) *Opts {
	o := &Opts{}

	if len(opts) == 1 {
		o = opts[0]
	}

	return o
}

// WithAPIKey adds the app that is currently requested to context. This wrapper
// uses an API token to differentiate between apps.
func WithAPIKey(handler func(*RequestContext) *shttp.Response, opts ...*Opts) shttp.RequestFunc {
	options := getOpts(opts...)

	return func(req *shttp.RequestContext) *shttp.Response {
		token := strings.Replace(req.Headers().Get("Authorization"), "Bearer ", "", 1)
		request := &RequestContext{
			RequestContext: &user.RequestContext{
				RequestContext: req,
			},
		}

		if strings.Contains(token, "SK_") {
			key, err := apikey.NewStore().APIKey(req.Context(), token)

			if err != nil {
				return shttp.Error(err)
			}

			// This is an invalid key. A key needs to have either an app ID, user ID or team ID.
			if key == nil || (key.UserID == 0 && key.AppID == 0 && key.TeamID == 0) {
				return shttp.Forbidden()
			}

			request.Token = key
			isAdmin := false

			// Check if this is an admin scope. If yes, first thing
			// let's verify user is an admin.
			if options.Admin {
				usr, err := user.NewStore().UserByID(key.UserID)

				if err != nil {
					return shttp.Error(err)
				}

				if usr == nil || !usr.IsAdmin {
					return shttp.Forbidden()
				}

				isAdmin = true
			}

			var appID types.ID
			var envID types.ID

			if key.AppID != 0 {
				appID = key.AppID
			} else {
				appID = appIDFromContext(req)
			}

			if key.EnvID != 0 {
				envID = key.EnvID
				appID = key.AppID // In case of env level key, we can get the app id from the env id.
			} else {
				envID = envIDFromContext(req)
			}

			if options.Env && envID == 0 {
				return &shttp.Response{
					Status: http.StatusBadRequest,
					Data: map[string]string{
						"error": "This is an environment-level API endpoint. Make sure to provide the envId paramater or use an environment-level API key.",
					},
				}
			}

			if options.App && appID == 0 {
				return &shttp.Response{
					Status: http.StatusBadRequest,
					Data: map[string]string{
						"error": "This is an app-level API endpoint. Make sure to provide the appId.",
					},
				}
			}

			request.EnvID = envID

			// Then let's fetch the app
			if appID != 0 {
				request.App, err = NewStore().AppByID(req.Context(), appID)
			} else if envID != 0 {
				request.App, err = NewStore().AppByEnvID(req.Context(), envID)
			}

			if err != nil {
				return shttp.Error(err)
			}

			if request.App == nil {
				return shttp.Forbidden()
			}

			// If this is a team level key, let's check if the app belongs to the team.
			if key.TeamID != 0 && request.App.TeamID != key.TeamID {
				return shttp.Forbidden()
			}

			// If this is a user level key, let's check if the user has access to the app.
			if key.UserID != 0 && !isAdmin {
				if !team.NewStore().IsMember(req.Context(), key.UserID, request.App.TeamID) {
					return shttp.Forbidden()
				}
			}

			return handler(request)
		}

		// If the token does not start with SK_ use the traditional JWT approach.
		return WithApp(func(rc *RequestContext) *shttp.Response {
			return handler(rc)
		})(req)
	}
}

// WithApp adds the app that is currently requested to the context.
func WithApp(handler func(*RequestContext) *shttp.Response, opts ...*Opts) shttp.RequestFunc {
	options := getOpts(opts...)

	return user.WithAuth(func(req *user.RequestContext) *shttp.Response {
		appID := appIDFromContext(req.RequestContext)
		envID := envIDFromContext(req.RequestContext)

		store := NewStore()
		app := &MyApp{App: &App{
			user: req.User,
		}}

		var err error

		if appID != 0 {
			app.App, err = store.AppByID(req.Context(), appID)
		} else if envID != 0 {
			app.App, err = store.AppByEnvID(req.Context(), envID)
		} else {
			return shttp.Error(ErrMissingOrInvalidAppID)
		}

		if err != nil || app.App == nil {
			return shttp.NotFound().SetError(err)
		}

		if !req.User.IsAdmin {
			if is := team.NewStore().IsMember(req.Context(), req.User.ID, app.TeamID); !is {
				return shttp.NotFound().SetError(err)
			}
		}

		reqContext := &RequestContext{
			RequestContext: req,
			App:            app.App,
			MyApp:          app,
			EnvID:          envID,
		}

		if options.Env && reqContext.EnvID == 0 {
			return &shttp.Response{
				Status: http.StatusBadRequest,
				Data: map[string]string{
					"error": "Missing environment ID.",
				},
			}
		}

		return handler(reqContext)
	})
}

// WithAppNoAuth returns the application without a need for authentication.
func WithAppNoAuth(handler func(*RequestContext) *shttp.Response) shttp.RequestFunc {
	return func(req *shttp.RequestContext) *shttp.Response {
		did, ok := req.Vars()["did"]

		if !ok {
			return shttp.BadRequest(map[string]any{
				"error": "Missing app ID in path parameters.",
				"hint":  "Make sure to include the app ID in the path parameters. For example: /apps/{appId}",
			})
		}

		didInt, err := strconv.ParseInt(did, 10, 64)

		if err != nil {
			return shttp.BadRequest(map[string]any{
				"error": "Invalid app ID. App ID should be a number.",
				"hint":  "Make sure the app ID in the path parameters is a valid number. For example: /apps/123",
			})
		}

		appID := types.ID(didInt)
		store := NewStore()
		app := &MyApp{App: &App{}}
		app.App, err = store.AppByID(req.Context(), appID)

		if err != nil || app.App == nil {
			return shttp.NotFound().SetError(err)
		}

		return handler(&RequestContext{
			App: app.App,
			RequestContext: &user.RequestContext{
				RequestContext: req,
			},
		})
	}
}

// getID recursively iterates through the lookup names and search the parameter in
// query variables or query string.
func getID(req *shttp.RequestContext, lookupNames []string) types.ID {
	for _, lookupName := range lookupNames {
		id, ok := req.Vars()[lookupName]

		if ok && id != "" {
			return utils.StringToID(id)
		}

		if id = req.Query().Get(lookupName); id != "" {
			return utils.StringToID(id)
		}
	}

	return 0
}

// extracts the app id from the request context.
func appIDFromContext(req *shttp.RequestContext) types.ID {
	if req.Method == shttp.MethodGet || req.Method == shttp.MethodDelete {
		appID := getID(req, []string{"aid", "did", "appId"})

		// Allow continuing because DELETE requests may have body
		if appID != 0 {
			return appID
		}
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

// Extract the Env ID from the request context.
// This function does not expect an environment id.
// If it finds, it returns it, otherwise returns 0.
func envIDFromContext(req *shttp.RequestContext) types.ID {
	if req.Method == shttp.MethodGet || req.Method == shttp.MethodDelete {
		id := getID(req, []string{"eid", "envId"})

		if id != 0 {
			return id
		}
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

func isMultipart(req *shttp.RequestContext) bool {
	return strings.HasPrefix(req.Header.Get("content-type"), "multipart/form-data")
}
