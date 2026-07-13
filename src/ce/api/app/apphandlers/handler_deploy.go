package apphandlers

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ee/api/team"

	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/github"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// for authorized user it clones given "template" github repository
// i.e http://localhost:8080/deploy?template=https%3A%2F%2Fgithub.com%2Fstormkit-io%2Fmonorepo-template-react
func handlerOneClickDeploy(req *shttp.RequestContext) *shttp.Response {
	usr, err := user.FromContext(req)
	repoName := req.Request.URL.Query().Get("template")

	if usr == nil {
		url := admin.MustConfig().AppURL(fmt.Sprintf("/auth?template=%s", repoName))
		req.Redirect(url, http.StatusFound)
		return nil
	}

	if err != nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
		}
	}

	client, err := github.NewClient(usr.ID)

	if err != nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]interface{}{
				"message": "Can't start OAuth App make sure to give permissions",
			},
		}
	}

	templateOwner, templateRepo, err := getInfoFromParameter(repoName)

	if err != nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]interface{}{
				"message": err.Error(),
			},
		}
	}

	params := &github.TemplateRepoRequest{
		Name:    utils.Ptr(templateRepo),
		Owner:   utils.Ptr(usr.DisplayName),
		Private: utils.Ptr(true),
	}

	repo, response, err := client.CreateFromTemplate(req.Context(), templateOwner, templateRepo, params)

	if err != nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string]interface{}{
				"message": err.Error(),
			},
		}
	}

	if response.StatusCode == http.StatusCreated {
		myApp := app.New(usr.ID)
		myApp.Repo = fmt.Sprintf("github/%s", *repo.FullName)
		myApp.TeamID, err = team.NewStore().DefaultTeamID(req.Context(), usr.ID)

		if err != nil {
			return shttp.Error(err)
		}

		if _, err := app.NewStore().InsertApp(req.Context(), myApp); err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Status: http.StatusOK,
	}
}

func getInfoFromParameter(template string) (owner string, repo string, err error) {
	const baseUrl = "https://github.com/"

	decoded, err := url.QueryUnescape(template)

	if err != nil {
		return "", "", err
	}

	if decoded == "" || !strings.HasPrefix(decoded, baseUrl) {
		return "", "", errors.New("template parameter is in wrong format")
	}

	decoded = strings.ReplaceAll(decoded, baseUrl, "")
	info := strings.Split(decoded, "/")

	if len(info) < 2 {
		return "", "", errors.New("template parameter is in wrong format")
	}

	return info[0], info[1], nil
}
