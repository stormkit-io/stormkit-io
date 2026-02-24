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

	// api endpoints
	s.NewEndpoint("/skauth").
		Handler(shttp.MethodPost, "", app.WithApp(handlerAuthUpsert, &app.Opts{Env: true})).
		Handler(shttp.MethodGet, "/providers", app.WithApp(handlerAuths, &app.Opts{Env: true}))

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
