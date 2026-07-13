package deployhooks

import (
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"gopkg.in/guregu/null.v3"
)

// AppDetails represents the application details for hooks.
type AppDetails struct {
	// Repo is the application's repository path.
	Repo string

	// DisplayName is the application's display name. It is used to reconstruct the development url.
	DisplayName string

	// AppID is the application's ID.
	AppID types.ID

	// UserID represents the owner of the repository. It's used for
	// bitbucket authentication.
	UserID types.ID

	// IsAutoDeploy specifies whether the deployment was an auto deploy or not.
	IsAutoDeploy bool

	// PullRequestNumber is the unique identifier number for the pull request.
	PullRequestNumber int64

	// AutoPublish specifies if the auto publish is turned on for environment.
	AutoPublish null.Bool
}
