package adminhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerAdminMailerDelete removes the instance's default mailer. Environments
// that used it keep their choice, but record emails without sending them until
// a mailer is configured again.
func handlerAdminMailerDelete(req *user.RequestContext) *shttp.Response {
	cnf := admin.MustConfig().Clone()
	cnf.MailerConfig = nil

	if err := admin.Store().UpsertConfig(req.Context(), cnf); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
