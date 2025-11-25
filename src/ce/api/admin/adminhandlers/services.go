package adminhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// Services sets the Handlers for this service.
func Services(r *shttp.Router) *shttp.Service {
	s := r.NewService()

	s.NewEndpoint("/admin/jobs").
		Handler(shttp.MethodPost, "/sync-analytics", user.WithAdmin(handlerJobsSyncAnalytics)).
		Handler(shttp.MethodPost, "/remove-old-artifacts", user.WithAdmin(handlerJobsRemoveOldArtifacts))

	s.NewEndpoint("/admin/system").
		Handler(shttp.MethodGet, "/runtimes", user.WithAdmin(handlerRuntimes)).
		Handler(shttp.MethodPost, "/runtimes", user.WithAdmin(handlerRuntimesInstall)).
		Handler(shttp.MethodGet, "/mise", user.WithAdmin(handlerMise)).
		Handler(shttp.MethodPost, "/mise", user.WithAdmin(handlerMiseUpdate)).
		Handler(shttp.MethodGet, "/proxies", user.WithAdmin(handlerProxies)).
		Handler(shttp.MethodPut, "/proxies", user.WithAdmin(handlerProxiesUpdate))

	s.NewEndpoint("/admin/license").
		Handler(shttp.MethodPost, "", user.WithAdmin(handlerLicenseSet))

	s.NewEndpoint("/admin/git").
		Handler(shttp.MethodGet, "/details", user.WithAdmin(handlerGitDetails)).
		Handler(shttp.MethodPost, "/configure", user.WithAdmin(handlerGitConfigure)).
		Handler(shttp.MethodPost, "/github/manifest", user.WithAdmin(handlerGitHubGenerateManifest)).
		Handler(shttp.MethodGet, "/github/callback", handlerGitHubManifestCallback)

	s.NewEndpoint("/admin/domains").
		Handler(shttp.MethodGet, "", user.WithAdmin(handlerAdminDomainsGet)).
		Handler(shttp.MethodPost, "", user.WithAdmin(handlerAdminDomainsSet))

	s.NewEndpoint("/admin/users").
		Handler(shttp.MethodGet, "/sign-up-mode", user.WithAdmin(handlerUserManagementGet)).
		Handler(shttp.MethodPost, "/sign-up-mode", user.WithAdmin(handlerUserManagementSet)).
		Handler(shttp.MethodGet, "/pending", user.WithAdmin(handlerUsersPending))

	if config.IsStormkitCloud() || config.IsDevelopment() {
		s.NewEndpoint("/admin/cloud").
			Handler(shttp.MethodPost, "/impersonate", user.WithAdmin(handlerImpersonate)).
			Handler(shttp.MethodGet, "/app", user.WithAdmin(handlerAdminAppGet)).
			Handler(shttp.MethodDelete, "/app", user.WithAdmin(handlerAdminAppDelete)).
			Handler(shttp.MethodPost, "/license", user.WithAdmin(handlerLicenseGenerate))
	}

	return s
}
