package adminhandlers

import (
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/nixstore"
)

type systemSettingsUpdateRequest struct {
	ArtifactRetentionDays int `json:"artifactRetentionDays"`
	NixRetentionDays      int `json:"nixRetentionDays"`
}

func handlerSystemSettingsUpdate(req *user.RequestContext) *shttp.Response {
	data := systemSettingsUpdateRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if data.ArtifactRetentionDays <= 0 {
		return shttp.BadRequest(map[string]any{
			"error": "artifactRetentionDays must be a positive number",
		})
	}

	// Omitted by older clients that predate the setting; keep what is stored.
	if data.NixRetentionDays < 0 {
		return shttp.BadRequest(map[string]any{
			"error": "nixRetentionDays must be a positive number",
		})
	}

	vc, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	if vc.SystemConfig == nil {
		vc.SystemConfig = &admin.SystemConfig{}
	}

	vc.SystemConfig.ArtifactRetentionDays = data.ArtifactRetentionDays

	if data.NixRetentionDays > 0 {
		vc.SystemConfig.NixRetentionDays = data.NixRetentionDays
	} else if vc.SystemConfig.NixRetentionDays <= 0 {
		vc.SystemConfig.NixRetentionDays = nixstore.DefaultRetentionDays
	}

	if err := admin.Store().UpsertConfig(req.Context(), vc); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"artifactRetentionDays": vc.SystemConfig.ArtifactRetentionDays,
			"nixRetentionDays":      vc.SystemConfig.NixRetentionDays,
		},
	}
}
