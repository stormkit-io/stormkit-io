package publicapiv1

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"gopkg.in/guregu/null.v3"
)

type EnvUpdateRequest struct {
	Name               *string                 `json:"name,omitempty"`
	Branch             *string                 `json:"branch,omitempty"`
	AutoDeploy         *bool                   `json:"autoDeploy,omitempty"`
	AutoDeployBranches *string                 `json:"autoDeployBranches,omitempty"`
	AutoDeployCommits  *string                 `json:"autoDeployCommits,omitempty"`
	AutoPublish        *bool                   `json:"autoPublish,omitempty"`
	APIFolder          *string                 `json:"apiFolder,omitempty"`
	APIPathPrefix      *string                 `json:"apiPathPrefix,omitempty"`
	BuildCmd           *string                 `json:"buildCmd,omitempty"`
	DistFolder         *string                 `json:"distFolder,omitempty"`
	ErrorFile          *string                 `json:"errorFile,omitempty"`
	Headers            *string                 `json:"headers,omitempty"`
	HeadersFile        *string                 `json:"headersFile,omitempty"`
	InstallCmd         *string                 `json:"installCmd,omitempty"`
	PreviewLinks       *bool                   `json:"previewLinks,omitempty"`
	Redirects          []redirects.Redirect    `json:"redirects,omitempty"`
	RedirectsFile      *string                 `json:"redirectsFile,omitempty"`
	ServerCmd          *string                 `json:"serverCmd,omitempty"`
	ServerFolder       *string                 `json:"serverFolder,omitempty"`
	StatusChecks       []buildconf.StatusCheck `json:"statusChecks,omitempty"`
	EnvVars            map[string]string       `json:"envVars,omitempty"`
}

func handlerEnvUpdate(req *RequestContext) *shttp.Response {
	data := &EnvUpdateRequest{}

	if err := req.Post(data); err != nil {
		return shttp.Error(err)
	}

	env := req.Env

	// Snapshot the original state before applying changes.
	old := *env
	oldData := *env.Data

	if data.Name != nil {
		env.Name = *data.Name
		env.Env = *data.Name
	}

	if data.Branch != nil {
		env.Branch = *data.Branch
	}

	if data.AutoPublish != nil {
		env.AutoPublish = *data.AutoPublish
	}

	if data.AutoDeploy != nil {
		env.AutoDeploy = *data.AutoDeploy
	}

	if data.AutoDeployBranches != nil {
		env.AutoDeployBranches = null.StringFrom(*data.AutoDeployBranches)

		if *data.AutoDeployBranches != "" {
			env.AutoDeploy = true
		}
	}

	if data.AutoDeployCommits != nil {
		env.AutoDeployCommits = null.StringFrom(*data.AutoDeployCommits)

		if *data.AutoDeployCommits != "" {
			env.AutoDeploy = true
		}
	}

	if data.APIFolder != nil {
		env.Data.APIFolder = utils.TrimPath(*data.APIFolder)
	}

	if data.APIPathPrefix != nil {
		env.Data.APIPathPrefix = utils.TrimPath(*data.APIPathPrefix)
	}

	if data.BuildCmd != nil {
		env.Data.BuildCmd = *data.BuildCmd
	}

	if data.DistFolder != nil {
		env.Data.DistFolder = utils.TrimPath(*data.DistFolder)
	}

	if data.ErrorFile != nil {
		env.Data.ErrorFile = utils.TrimPath(*data.ErrorFile)
	}

	if data.Headers != nil {
		if _, err := deploy.ParseHeaders(*data.Headers); err != nil {
			return shttp.BadRequest(map[string]any{"error": err.Error()})
		}

		env.Data.Headers = *data.Headers
	}

	if data.HeadersFile != nil {
		env.Data.HeadersFile = utils.TrimPath(*data.HeadersFile)
	}

	if data.InstallCmd != nil {
		env.Data.InstallCmd = *data.InstallCmd
	}

	if data.PreviewLinks != nil {
		env.Data.PreviewLinks = null.BoolFrom(*data.PreviewLinks)
	}

	if data.RedirectsFile != nil {
		env.Data.RedirectsFile = utils.TrimPath(*data.RedirectsFile)
	}

	if data.ServerCmd != nil {
		env.Data.ServerCmd = *data.ServerCmd
	}

	if data.ServerFolder != nil {
		env.Data.ServerFolder = utils.TrimPath(*data.ServerFolder)
	}

	if data.Redirects != nil {
		env.Data.Redirects = data.Redirects

		if errs := redirects.Validate(env.Data.Redirects); len(errs) > 0 {
			return shttp.BadRequest(map[string]any{"errors": errs})
		}
	}

	if data.StatusChecks != nil {
		env.Data.StatusChecks = data.StatusChecks
	}

	if data.EnvVars != nil {
		env.Data.Vars = data.EnvVars
	}

	if errs := buildconf.Validate(env); len(errs) > 0 {
		return shttp.BadRequest(map[string]any{"errors": errs})
	}

	store := buildconf.NewStore()

	if err := store.Update(req.Context(), env); err != nil {
		if database.IsDuplicate(err) {
			return shttp.Error(buildconf.ErrDuplicateEnvName)
		}

		return shttp.Error(err)
	}

	if req.License().IsEnterprise() {
		diff := &audit.Diff{
			Old: audit.DiffFields{
				EnvName:               old.Name,
				EnvBranch:             old.Branch,
				EnvAutoPublish:        audit.Bool(old.AutoPublish),
				EnvAutoDeploy:         audit.Bool(old.AutoDeploy),
				EnvAutoDeployBranches: old.AutoDeployBranches.ValueOrZero(),
				EnvAutoDeployCommits:  old.AutoDeployCommits.ValueOrZero(),
				EnvBuildConfig:        &oldData,
			},
			New: audit.DiffFields{
				EnvName:               env.Name,
				EnvBranch:             env.Branch,
				EnvAutoPublish:        audit.Bool(env.AutoPublish),
				EnvAutoDeploy:         audit.Bool(env.AutoDeploy),
				EnvAutoDeployBranches: env.AutoDeployBranches.ValueOrZero(),
				EnvAutoDeployCommits:  env.AutoDeployCommits.ValueOrZero(),
				EnvBuildConfig:        env.Data,
			},
		}

		err := audit.FromRequestContext(req).
			WithAction(audit.UpdateAction, audit.TypeEnv).
			WithDiff(diff).
			WithEnvID(env.ID).
			Insert()

		if err != nil {
			return shttp.Error(err)
		}
	}

	if err := appcache.Service().Reset(env.ID); err != nil {
		return shttp.Error(err)
	}

	err := app.UpdateFunctionConfiguration(req.Context(), app.FunctionConfiguration{
		AppID: req.App.ID,
		EnvID: env.ID,
		Vars:  env.Data.Vars,
	})

	if err != nil {
		return shttp.Error(err)
	}

	return shttp.OK()
}
