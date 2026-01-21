package jobs

import (
	"context"
	"strings"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
)

type KeyContextNumberOfDeploymentsToDelete struct{}

// RemoveDeploymentArtifactsManually removes the artifacts of expired deployments.
// An expired deployment is a deployment that has not been used for more than 30 days.
// Overwrite the numberOfDays to set a custom expiration time.
func RemoveDeploymentArtifactsManually(ctx context.Context, numberOfDays int) ([]string, error) {
	limit, _ := ctx.Value(KeyContextNumberOfDeploymentsToDelete{}).(int)

	if limit <= 0 {
		limit = 100
	}

	if numberOfDays == 0 {
		numberOfDays = 30
	}

	store := NewStore()
	deployments, err := store.DeploymentsOlderThan30Days(ctx, numberOfDays, limit)

	if err != nil {
		return nil, err
	}

	if len(deployments) == 0 {
		return nil, nil
	}

	client := integrations.Client()
	idsToBeMarked := []types.ID{}
	idsToBeMarkedStr := []string{}

	for _, d := range deployments {
		if d.UploadResult != nil {
			args := integrations.DeleteArtifactsArgs{
				FunctionLocation: d.UploadResult.ServerLocation,
				APILocation:      d.UploadResult.ServerlessLocation,
				StorageLocation:  d.UploadResult.ClientLocation,
			}

			if err := client.DeleteArtifacts(ctx, args); err != nil {
				slog.Errorf("error while deleting artifact: %s", err.Error())
				continue
			}
		}

		idsToBeMarked = append(idsToBeMarked, d.ID)
		idsToBeMarkedStr = append(idsToBeMarkedStr, d.ID.String())
	}

	if err = store.MarkDeploymentArtifactsDeleted(ctx, idsToBeMarked); err != nil {
		slog.Errorf("error while marking artifacts deleted: %s", strings.Join(idsToBeMarkedStr, ", "))
		return nil, err
	}

	return idsToBeMarkedStr, nil
}

// RemoveDeploymentArtifacts is a job to remove the artifacts of expired deployments.
func RemoveDeploymentArtifacts(ctx context.Context) error {
	idsToBeMarked, err := RemoveDeploymentArtifactsManually(ctx, 30)

	if err != nil {
		return err
	}

	if !config.IsTest() {
		slog.Infof("artifacts deleted: %s", strings.Join(idsToBeMarked, ", "))
	}

	return nil
}

// TimedOutDeployments is a job to time out deployments that have been running for too long.
func TimedOutDeployments(ctx context.Context) error {
	dids, err := NewStore().TimedOutDeployments(ctx)

	if err != nil {
		return err
	}

	for _, did := range dids {
		slog.Infof("timing out deployment: %s", did.String())

		if err := deploy.NewStore().TimeoutDeployment(ctx, did); err != nil {
			slog.Errorf("error while timing out deployment %s: %s", did.String(), err.Error())
		}

		// Sleep for a second to avoid overwhelming the database
		SleepOneSecond()
	}

	return nil
}

var SleepOneSecond = func() {
	time.Sleep(1 * time.Second)
}
