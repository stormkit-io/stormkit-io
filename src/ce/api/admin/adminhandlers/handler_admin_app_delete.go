package adminhandlers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/discord"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerAdminAppDelete(req *user.RequestContext) *shttp.Response {
	appID := utils.StringToID(req.Query().Get("appId"))
	appl, err := app.NewStore().AppByID(req.Context(), appID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to fetch app with id %s, err: %s", req.Query().Get("appId"), err.Error()))
	}

	if appl == nil {
		return shttp.NotFound()
	}

	ustore := user.NewStore()
	usr, err := ustore.UserByID(appl.UserID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("cannot fetch user by id: %s, err: %s", appl.UserID, err.Error()))
	}

	err = ustore.MarkUserAsDeleted(req.Context(), appl.UserID)

	if err != nil {
		return shttp.Error(err, fmt.Sprintf("failed to mark user as deleted: %s, err: %s", appl.UserID, err.Error()))
	}

	envStore := buildconf.NewStore()
	envs, err := envStore.ListEnvironments(context.Background(), appl.ID)

	if err != nil {
		slog.Errorf("failed silently while fetching envs for cache invalidation: %v", err)
	}

	for _, env := range envs {
		if err := appcache.Service().Reset(env.ID); err != nil {
			slog.Errorf("failed silently while invalidating cache: %v", err)
		}
	}

	discord.Notify(config.Get().Reporting.DiscordProductionChannel, discord.Payload{
		Embeds: []discord.PayloadEmbed{
			{
				Title:     "User deleted",
				Timestamp: time.Now().Format(time.RFC3339),
				Fields: []discord.PayloadField{
					{Name: "ID", Value: strconv.FormatInt(int64(appl.UserID), 10)},
					{Name: "Name", Value: usr.FullName()},
					{Name: "DisplayName", Value: usr.Display()},
					{Name: "Email", Value: usr.PrimaryEmail()},
				}},
		},
	})

	return shttp.OK()
}
