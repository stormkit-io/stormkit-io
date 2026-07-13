package providerhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/github"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type Account struct {
	ID     string `json:"id"`
	Login  string `json:"login"`
	Avatar string `json:"avatar"`
}

func handlerAccountList(req *user.RequestContext) *shttp.Response {
	provider := req.Vars()["provider"]

	if provider != github.ProviderName && provider != "gitlab" && provider != "bitbucket" {
		return shttp.BadRequest()
	}

	var err error
	accounts := []Account{}

	switch provider {
	case github.ProviderName:
		accounts, err = githubListAccounts(req)
	}

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"accounts": accounts,
		},
	}
}

func githubListAccounts(req *user.RequestContext) ([]Account, error) {
	client, err := github.NewClient(req.User.ID)

	if err != nil {
		return nil, err
	}

	ghInstallations, _, err := client.ListUserInstallations(req.Context(), &github.ListOptions{
		PerPage: maxPerPage,
	})

	if err != nil || ghInstallations == nil {
		return nil, err
	}

	accounts := []Account{}

	for _, inst := range ghInstallations {
		acc := inst.GetAccount()

		accounts = append(accounts, Account{
			ID:     utils.Int64ToString(inst.GetID()),
			Login:  acc.GetLogin(),
			Avatar: acc.GetAvatarURL(),
		})
	}

	return accounts, nil
}
