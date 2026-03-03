package skauthhandlers

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/skauth"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

// Services sets the Handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	withEnv := &app.Opts{Env: true}

	// api endpoints
	s.NewEndpoint("/skauth").
		Handler(shttp.MethodPost, "", app.WithApp(handlerAuthUpsert, withEnv)).
		Handler(shttp.MethodPost, "/config", app.WithApp(handlerAuthConfigUpdate, withEnv)).
		Handler(shttp.MethodGet, "/providers", app.WithApp(handlerAuths, withEnv))

	return s
}

// AuthUser retrieves the authenticated user from the environment's schema store.
var AuthUser = func(ctx context.Context, env *buildconf.Env, authID types.ID) (*skauth.User, error) {
	store, err := env.SchemaConf.Store(buildconf.SchemaAccessTypeAppUser)

	if err != nil {
		return nil, err
	}

	return store.AuthUser(ctx, authID)
}
