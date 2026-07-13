package adminhandlers

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerAdminAppGet returns the app information, with the associated user.
func handlerAdminAppGet(req *user.RequestContext) *shttp.Response {
	addrs := req.Query().Get("url")
	store := app.NewStore()

	if !strings.HasPrefix(addrs, "http") && !strings.HasPrefix(addrs, "//") {
		addrs = fmt.Sprintf("https://%s", addrs)
	}

	addrs = strings.ReplaceAll(addrs, "[.]", ".") // AWS trust report format fix
	parsed, err := url.Parse(addrs)

	if err != nil {
		return shttp.BadRequest(map[string]any{
			"error": "Invalid URL format provided",
		})
	}

	hostname := parsed.Hostname()

	var appl *app.App

	if appconf.IsStormkitDev(addrs) {
		appl, err = store.AppByDisplayName(req.Context(), strings.Split(hostname, ".")[0])
	} else {
		appl, err = store.AppByDomainName(req.Context(), hostname)
	}

	if err != nil {
		return shttp.Error(err)
	}

	if appl == nil {
		return shttp.NoContent()
	}

	usr, err := user.NewStore().UserByID(appl.UserID)

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Data: map[string]any{
			"app":  appl.JSON(),
			"user": usr.JSON(),
		},
	}
}
