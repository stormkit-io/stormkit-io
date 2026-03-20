package instancehandlers

import (
	"context"
	"net/http"
	"time"

	"github.com/google/go-github/v71/github"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

var LatestCommit string
var LatestRelease string
var LastCacheTime time.Time
var GHClient ReleaseClient

type ReleaseClient interface {
	GetLatestRelease(context.Context, string, string) (*github.RepositoryRelease, *github.Response, error)
	GetBranch(context.Context, string, string, string, int) (*github.Branch, *github.Response, error)
}

func handlerInstanceDetails(req *shttp.RequestContext) *shttp.Response {
	conf := config.Get()

	if err := getLatestRelease(req); err != nil {
		slog.Errorf("error while getting latest release: %s", err.Error())
	}

	usr, _ := user.FromContext(req)
	license := user.License(usr)

	var totalUsers int64
	var err error

	if !license.IsEmpty() && config.IsSelfHosted() {
		totalUsers, err = user.NewStore().SelectTotalUsers(req.Context())
	} else if usr != nil {
		totalUsers, err = user.NewStore().SelectTotalUsersCloud(req.Context(), usr.ID)
	}

	if err != nil {
		return shttp.Error(err)
	}

	edition := "development"

	if config.IsStormkitCloud() {
		edition = "cloud"
	} else if config.IsSelfHosted() {
		edition = "self-hosted"
	}

	hash := conf.Version.Hash

	if len(hash) > 7 {
		hash = hash[:7]
	}

	data := map[string]any{
		"stormkit": map[string]any{
			"apiVersion": conf.Version.Tag,
			"apiCommit":  hash,
			"edition":    edition,
		},
		"latest": map[string]any{
			"apiVersion": LatestRelease,
		},
	}

	if !license.IsEmpty() {
		data["license"] = map[string]any{
			"seats":     license.Seats,
			"edition":   license.Edition(),
			"remaining": int64(license.Seats) - totalUsers,
		}
	}

	cnf := admin.MustConfig().AuthConfig

	if cnf.Github.Account != "" {
		data["auth"] = map[string]string{
			"github": cnf.Github.Account,
		}
	}

	return &shttp.Response{
		Data: data,
	}
}

// getLatestRelease returns the latest tag for self-hosted customers.
func getLatestRelease(req *shttp.RequestContext) error {
	if LatestRelease != "" && time.Since(LastCacheTime) < 6*time.Hour {
		return nil
	}

	if GHClient == nil {
		GHClient = github.NewClient(&http.Client{Timeout: time.Second * 30}).Repositories
	}

	release, _, err := GHClient.GetLatestRelease(req.Context(), "stormkit-io", "bin")

	if err != nil {
		return err
	}

	if release != nil && release.TagName != nil {
		LatestRelease = *release.TagName
		LastCacheTime = time.Now()
	}

	return nil
}
