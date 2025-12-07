package router

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin/adminhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey/apikeyhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apphandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall/authwallhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/buildconfhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/domainhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/schemahandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/snippetshandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy/deployhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/functiontrigger/functiontriggerhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/mailer/mailerhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/providerhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects/redirectshandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/volumes/volumeshandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/applog/apploghandlers"
	publicapiv1 "github.com/stormkit-io/stormkit-io/src/ce/api/public/v1"
	"github.com/stormkit-io/stormkit-io/src/ce/api/status"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/authhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/instancehandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/subscriptionhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/userhandlers"
	"github.com/stormkit-io/stormkit-io/src/ee/api/analytics/analyticshandlers"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit/audithandlers"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team/teamhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func Get() *shttp.Router {
	r := shttp.NewRouter()
	r.RegisterMiddleware(WithCors)
	r.RegisterMiddleware(WithTimeout)

	// This simply checks for the license and kills the process if the license is not
	// available or expired.
	license := admin.CurrentLicense()
	license.Debug()

	// Enable cors
	_ = Cors()

	r.RegisterService(apphandlers.Services)
	r.RegisterService(deployhandlers.Services)
	r.RegisterService(buildconfhandlers.Services)
	r.RegisterService(userhandlers.Services)
	r.RegisterService(status.Services)
	r.RegisterService(publicapiv1.Services)
	r.RegisterService(apploghandlers.Services)
	r.RegisterService(apikeyhandlers.Services)
	r.RegisterService(authhandlers.Services)
	r.RegisterService(domainhandlers.Services)
	r.RegisterService(redirectshandlers.Services)
	r.RegisterService(instancehandlers.Services)
	r.RegisterService(mailerhandlers.Services)
	r.RegisterService(adminhandlers.Services)
	r.RegisterService(providerhandlers.Services)
	r.RegisterService(snippetshandlers.Services)
	r.RegisterService(volumeshandlers.Services)
	r.RegisterService(functiontriggerhandlers.Services)

	// Enterprise handlers
	r.RegisterService(authwallhandlers.Services)
	r.RegisterService(analyticshandlers.Services)
	r.RegisterService(audithandlers.Services)
	r.RegisterService(teamhandlers.Services)

	if config.IsStormkitCloud() || config.IsDevelopment() {
		r.RegisterService(subscriptionhandlers.Services)
	}

	// This is currently only available in development mode
	if config.IsDevelopment() {
		r.RegisterService(schemahandlers.Services)
	}

	return r
}
