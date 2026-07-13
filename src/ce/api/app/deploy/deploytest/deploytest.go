package deploytest

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	null "gopkg.in/guregu/null.v3"
)

// MockDeploy returns a mock deployment.
func MockDeploy(appID types.ID) *deploy.Deployment {
	return &deploy.Deployment{
		ID:        types.ID(1535),
		AppID:     appID,
		CreatedAt: utils.NewUnix(),
		ExitCode:  null.NewInt(0, true),
		Logs: null.NewString(`[
			{
				"title":"checkout master",
				"status":true,
				"message":"Succesfully checked out branch",
				"payload":{
					"branch":"master",
					"commit":{
						"author":"Johnny Depp",
						"sha":"abxv141",
						"message":"Test application layer"
					}
				}
			},
			{
				"title":"yarn run build",
				"status":true,
				"message":"Succesfully built yarn project"
			}
		]`, true),
	}
}
