package userhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerUserDelete deletes the user account. We don't do hard deletes,
// but rather turn on flags and later our garbage collector will remove
// all the artifacts.
func handlerUserDelete(req *user.RequestContext) *shttp.Response {
	if err := user.NewStore().MarkUserAsDeleted(req.Context(), req.User.ID); err != nil {
		return shttp.UnexpectedError(err)
	}
	return shttp.OK()
}
