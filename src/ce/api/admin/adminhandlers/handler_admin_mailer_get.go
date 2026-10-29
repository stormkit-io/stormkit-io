package adminhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerAdminMailerGet returns the instance's default mailer, or null when
// none is configured. The password is never included.
func handlerAdminMailerGet(req *user.RequestContext) *shttp.Response {
	return &shttp.Response{
		Data: map[string]any{
			"mailer": admin.MustConfig().MailerConfig.JSON(),
		},
	}
}
