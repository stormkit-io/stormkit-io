package adminhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/rediscache"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

// handlerDiskCleanup asks every container to garbage collect its own Nix
// store now, rather than waiting for the daily run. Each container holds a
// separate /nix volume, so this is a broadcast and not a local call.
func handlerDiskCleanup(req *user.RequestContext) *shttp.Response {
	if err := rediscache.Broadcast(rediscache.EventDiskCleanup); err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
