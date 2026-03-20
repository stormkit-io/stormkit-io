package deploy

import (
	"context"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// PublishSettings are the settings for publishing a deployment.
type PublishSettings struct {
	DeploymentID types.ID
	EnvID        types.ID
	Percentage   float64
	NoCacheReset bool
}

// AutoPublish automatically publishes successful deployments if the
// auto publish feature is enabled.
func AutoPublishIfNecessary(ctx context.Context, d *Deployment) error {
	if !d.ShouldPublish || d.ExitCode.ValueOrZero() != 0 || d.Error.ValueOrZero() != "" {
		return nil
	}

	settings := []*PublishSettings{
		{
			EnvID:        d.EnvID,
			DeploymentID: d.ID,
			Percentage:   100,
		},
	}

	if err := Publish(ctx, settings); err != nil {
		return err
	}

	var license *admin.License

	if config.IsStormkitCloud() {
		usr, err := user.NewStore().AppOwner(ctx, d.AppID)

		if err != nil {
			return err
		}

		license = user.License(usr)
	} else {
		license = admin.CurrentLicense()
	}

	if license != nil && license.IsEnterprise() {
		apl, err := app.NewStore().AppByID(ctx, d.AppID)

		if err != nil {
			return err
		}

		if apl == nil {
			return nil
		}

		return audit.New(ctx).
			WithAction(audit.UpdateAction, audit.TypeDeployment).
			WithTeamID(apl.TeamID).
			WithAppID(d.AppID).
			WithEnvID(d.EnvID).
			WithDiff(&audit.Diff{
				New: audit.DiffFields{
					DeploymentID:  d.ID.String(),
					AutoPublished: utils.Ptr(true),
				},
			}).
			Insert()
	}

	return nil
}

// Publish publishes a new deployment.
func Publish(ctx context.Context, settings []*PublishSettings) error {
	if err := NewStore().Publish(ctx, settings...); err != nil {
		return err
	}

	envIDs := map[types.ID]bool{}
	store := buildconf.NewStore()

	for _, s := range settings {
		envID := s.EnvID

		if _, ok := envIDs[envID]; ok {
			continue
		}

		env, err := store.EnvironmentByID(ctx, envID)

		if err != nil {
			return err
		}

		appl, err := app.NewStore().AppByEnvID(ctx, envID)

		if err != nil {
			return err
		}

		if !s.NoCacheReset {
			if err := appcache.Service().Reset(env.ID); err != nil {
				return err
			}
		}

		whs := app.NewStore().OutboundWebhooks(ctx, env.AppID)
		cnf := admin.MustConfig()

		for _, wh := range whs {
			if wh.TriggerOnPublish() {
				wh.Dispatch(app.OutboundWebhookSettings{
					AppID:                  env.AppID,
					DeploymentID:           s.DeploymentID,
					DeploymentStatus:       "success",
					EnvironmentName:        env.Name,
					DeploymentEndpoint:     cnf.PreviewURL(appl.DisplayName, s.DeploymentID.String()),
					DeploymentLogsEndpoint: cnf.DeploymentLogsURL(env.AppID, s.DeploymentID),
				})
			}
		}

		envIDs[envID] = true
	}

	return nil
}
