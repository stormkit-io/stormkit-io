package authhandlers

import (
	"net/http"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/limiter"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

func hasAnyProviderEnabled() *shttp.Response {
	if admin.MustConfig().IsAuthEnabled() {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]string{
				"error": "OAuth is already enabled. You cannot login with basic auth.",
			},
		}
	}

	return nil
}

// Services installs the user services.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	if config.IsDevelopment() {
		token, _ := user.JWT(jwt.MapClaims{
			"uid": "1",
		})

		slog.Info("open console and type:")
		slog.Info("localStorage.setItem('skit_provider', JSON.stringify('github'))")
		slog.Infof("localStorage.setItem('skit_token', JSON.stringify('%s'))", token)
	}

	authEp := s.NewEndpoint("/auth")

	if !config.IsStormkitCloud() {
		authEp.
			Handler(shttp.MethodPost, "/admin/register", handlerAdminRegister).
			Handler(shttp.MethodPost, "/admin/login", handlerAdminLogin)
	}

	authEp.
		Handler(shttp.MethodGet, "/{provider:github|gitlab|bitbucket}", shttp.WithRateLimit(
			handlerAuthLogin,
			&limiter.Options{Limit: 50, Burst: 3, Duration: time.Minute},
		)).
		Handler(shttp.MethodGet, "/{provider:github|gitlab|bitbucket}/callback", shttp.WithRateLimit(
			handlerAuthCallback,
			&limiter.Options{Limit: 50, Burst: 3, Duration: time.Minute},
		)).
		Handler(shttp.MethodGet, "/github/installation", shttp.WithRateLimit(
			handlerAuthGithubInstallationCallback,
			&limiter.Options{Limit: 20, Burst: 2, Duration: time.Minute},
		)).
		Handler(shttp.MethodGet, "/providers", shttp.WithRateLimit(
			handlerAuthProviders,
			&limiter.Options{Limit: 50, Burst: 3, Duration: time.Minute},
		))

	return s
}
