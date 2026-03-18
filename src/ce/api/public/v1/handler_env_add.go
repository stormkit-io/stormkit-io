package publicapiv1

import (
	"net/http"
	"regexp"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"gopkg.in/guregu/null.v3"
)

type EnvAddRequest struct {
	APIFolder          string                  `json:"apiFolder,omitempty"`
	AutoDeploy         bool                    `json:"autoDeploy"`
	AutoDeployBranches null.String             `json:"autoDeployBranches,omitempty"`
	AutoDeployCommits  null.String             `json:"autoDeployCommits,omitempty"`
	AutoPublish        bool                    `json:"autoPublish"`
	Branch             string                  `json:"branch"`
	BuildCmd           string                  `json:"buildCmd,omitempty"`
	DistFolder         string                  `json:"distFolder,omitempty"`
	EnvVars            map[string]string       `json:"envVars,omitempty"`
	ErrorFile          string                  `json:"errorFile,omitempty"`
	HeadersFile        string                  `json:"headersFile,omitempty"`
	Name               string                  `json:"name"`
	PreviewLinks       null.Bool               `json:"previewLinks,omitempty"`
	Redirects          []redirects.Redirect    `json:"redirects,omitempty"`
	RedirectsFile      string                  `json:"redirectsFile,omitempty"`
	ServerCmd          string                  `json:"serverCmd,omitempty"`
	StatusChecks       []buildconf.StatusCheck `json:"statusChecks,omitempty"`
}

func validateEnv(env *buildconf.Env) []string {
	errors := []string{}

	if env.Branch == "" {
		errors = append(errors, "Branch is a required field")
	} else if match, _ := regexp.MatchString(`^[a-zA-Z0-9-/+=\.]+$`, env.Branch); !match {
		// See https://wincent.com/wiki/Legal_Git_branch_names for more details.
		errors = append(errors, "Branch name can only contain following characters: alphanumeric, -, +, /, ., and =")
	}

	if env.Name == "" {
		errors = append(errors, "Name is a required field")
	} else if match, _ := regexp.MatchString("^[a-zA-Z-0-9]+$", env.Name); !match {
		errors = append(errors, "Environment can only contain alphanumeric characters and hypens.")
	}

	if match, _ := regexp.MatchString("--", env.Name); match {
		errors = append(errors, "Double hypens (--) are not allowed as they are reserved for Stormkit.")
	}

	if len(errors) == 0 {
		return nil
	}

	return errors
}

func handlerEnvAdd(req *RequestContext) *shttp.Response {
	data := &EnvAddRequest{}

	if err := req.Post(data); err != nil {
		return shttp.Error(err)
	}

	cnf := &buildconf.Env{
		Data: &buildconf.BuildConf{
			APIFolder:     data.APIFolder,
			BuildCmd:      data.BuildCmd,
			DistFolder:    data.DistFolder,
			ErrorFile:     data.ErrorFile,
			HeadersFile:   data.HeadersFile,
			PreviewLinks:  data.PreviewLinks,
			ServerCmd:     data.ServerCmd,
			Redirects:     data.Redirects,
			RedirectsFile: data.RedirectsFile,
			Vars:          data.EnvVars,
		},
		Name:        data.Name,
		AppID:       req.App.ID,
		Branch:      data.Branch,
		AutoPublish: data.AutoPublish,
		AutoDeploy:  data.AutoDeploy,
	}

	if data.AutoDeployBranches.Valid {
		cnf.AutoDeployBranches = data.AutoDeployBranches
	} else if data.AutoDeployCommits.Valid {
		cnf.AutoDeployCommits = data.AutoDeployCommits
	}

	cnf.AutoDeploy = cnf.AutoDeploy || data.AutoDeployBranches.Valid || data.AutoDeployCommits.Valid

	if err := validateEnv(cnf); err != nil {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data: map[string][]string{
				"errors": err,
			},
		}
	}

	if err := buildconf.NewStore().Insert(req.Context(), cnf); err != nil {
		if database.IsDuplicate(err) {
			return &shttp.Response{
				Status: http.StatusConflict,
				Data: map[string][]string{
					"errors": {
						"Environment name already exists for this application.",
					},
				},
			}
		}

		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		err := audit.FromRequestContext(req).
			WithAction(audit.CreateAction, audit.TypeEnv).
			WithDiff(&audit.Diff{New: audit.DiffFields{EnvName: cnf.Name, EnvID: cnf.ID.String()}}).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	return &shttp.Response{
		Status: http.StatusCreated,
		Data: map[string]any{
			"envId": cnf.ID.String(),
		},
	}
}
