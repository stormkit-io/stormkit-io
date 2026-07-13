package providerhandlers

import (
	"net/http"
	"strings"

	gh "github.com/stormkit-io/stormkit-io/src/ce/api/oauth/github"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type Repo struct {
	Name     string `json:"name"`
	FullName string `json:"fullName"`
}

const maxPerPage = 100

func handlerRepoList(req *user.RequestContext) *shttp.Response {
	provider := req.Vars()["provider"]

	if provider != gh.ProviderName && provider != "gitlab" && provider != "bitbucket" {
		return shttp.BadRequest()
	}

	query := req.Query()
	search := strings.ToLower(query.Get("search"))
	page := utils.StringToInt(query.Get("page"))
	perPage := 10
	installationID := utils.StringToInt64(query.Get("installationId"))

	if page == 0 {
		page = 1
	}

	if search != "" {
		perPage = maxPerPage
	}

	var err error

	repos := []Repo{}
	hasNextPage := false

	switch provider {
	case gh.ProviderName:
		repos, hasNextPage, err = githubListRepos(GithubListReposRequest{
			req:            req,
			installationID: installationID,
			page:           page,
			perPage:        perPage,
			search:         search})
	}

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"repos":       repos,
			"hasNextPage": hasNextPage,
		},
	}
}

type GithubListReposRequest struct {
	req            *user.RequestContext
	installationID int64
	page           int
	perPage        int
	search         string
}

func githubListRepos(opts GithubListReposRequest) ([]Repo, bool, error) {
	client, err := gh.NewAppClient(opts.installationID)

	if err != nil {
		return nil, false, err
	}

	repos := []Repo{}
	hasNextPage := false
	page := opts.page
	perPage := opts.perPage

	for {
		ghRepos, _, err := client.ListRepos(opts.req.Context(), &gh.ListOptions{
			Page:    page,
			PerPage: perPage,
		})

		if err != nil {
			return nil, false, err
		}

		if ghRepos == nil {
			break
		}

		for _, repo := range ghRepos.Repositories {
			if opts.search == "" || (strings.Contains(strings.ToLower(repo.GetFullName()), opts.search)) {
				repos = append(repos, Repo{
					Name:     repo.GetName(),
					FullName: repo.GetFullName(),
				})

				continue
			}
		}

		hasNextPage = ghRepos.GetTotalCount() > page*perPage

		if opts.search == "" || !hasNextPage {
			break
		}

		page = page + 1
	}

	return repos, hasNextPage, nil
}
