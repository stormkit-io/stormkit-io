package deployservice

import (
	"fmt"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/bitbucket"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/github"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/gitlab"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

// CloudMaxRepoSize is the largest repository Stormkit Cloud will build. Self
// hosted instances own their own disk and are left uncapped.
const CloudMaxRepoSize int64 = 1 << 30 // 1GB

// errRepoTooLarge builds the error a deployment is rejected with. The message
// is shown to the customer, so it names both numbers and what to do about it.
func errRepoTooLarge(size, limit int64) error {
	return shttperr.New(
		http.StatusUnprocessableEntity,
		fmt.Sprintf(
			"Repository is larger than allowed. It is %s while the limit is %s. "+
				"Reduce the size of the repository, or move large files out of git.",
			humanBytes(size), humanBytes(limit),
		),
		"repo-too-large",
	)
}

// humanBytes formats a byte count for a customer-facing message.
func humanBytes(b int64) string {
	if b >= 1<<30 {
		return fmt.Sprintf("%.1fGB", float64(b)/(1<<30))
	}

	return fmt.Sprintf("%dMB", b/(1<<20))
}

// repoSizeChecker rejects a deployment before a build host is scheduled, using
// the size the git provider reports for the repository.
//
// The number is advisory: every provider reports the packed, full-history size
// of the whole repository, while a build only ever shallow-clones one branch.
// It is therefore a fast first gate rather than a precise one, and a provider
// that cannot answer must never block a deployment.
type repoSizeChecker struct {
	app *app.App
}

// limit returns the cap in bytes, or 0 when no cap applies.
func (c repoSizeChecker) limit() int64 {
	if !config.IsStormkitCloud() {
		return 0
	}

	// The override exists so an oversized customer can be unblocked by
	// reconfiguring the instance rather than by cutting a release.
	if mb := config.MaxRepoSizeMB(); mb > 0 {
		return mb << 20
	}

	return CloudMaxRepoSize
}

// MockRepoSize replaces the provider lookup in tests.
var MockRepoSize func(*app.App) (int64, error)

// size asks the provider how large the repository is, in bytes.
func (c repoSizeChecker) size() (int64, error) {
	if MockRepoSize != nil {
		return MockRepoSize(c.app)
	}

	switch {
	case c.app.IsGithub():
		return github.RepoSize(c.app.Repo)

	case c.app.IsGitlab():
		client, err := gitlab.NewClient(c.app.UserID)

		if err != nil || client == nil {
			return 0, err
		}

		return client.RepoSize(c.app.Repo)

	case c.app.IsBitbucket():
		client, err := bitbucket.NewClient(c.app.UserID)

		if err != nil || client == nil {
			return 0, err
		}

		return client.RepoSize(c.app.Repo)
	}

	return 0, nil
}

// check returns an error when the repository is too large to build.
func (c repoSizeChecker) check() error {
	limit := c.limit()

	if limit <= 0 {
		return nil
	}

	size, err := c.size()

	// A provider outage, a missing scope, or a token that cannot read
	// statistics must not take deployments down with it. The build host is
	// still protected by its own disk; this check only saves it the trip.
	if err != nil {
		slog.Errorf("could not read the size of %s: %v", c.app.Repo, err)
		return nil
	}

	if size <= limit {
		return nil
	}

	slog.Errorf("rejecting %s: %s repository, over the %s cap", c.app.Repo, humanBytes(size), humanBytes(limit))

	return errRepoTooLarge(size, limit)
}
