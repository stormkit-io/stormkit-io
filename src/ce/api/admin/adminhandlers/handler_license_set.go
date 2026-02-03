package adminhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type LicenseSetRequest struct {
	Key string `json:"key"`
}

func handlerLicenseSet(req *user.RequestContext) *shttp.Response {
	data := LicenseSetRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	license := admin.FreeLicense()

	// If the data.Key is empty, user wants to remove the license so we don't
	// need to validate it.
	if data.Key != "" {
		var err error
		license, err = admin.ValidateLicense(data.Key)

		if err != nil {
			return shttp.BadRequest(map[string]any{
				"error": err.Error(),
			})
		}
	}

	cnf, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	if cnf.LicenseConfig == nil {
		cnf.LicenseConfig = &admin.LicenseConfig{}
	}

	cnf.LicenseConfig.Key = data.Key

	if err := admin.Store().UpsertConfig(req.Context(), cnf); err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Data: map[string]any{
			"seats":    license.Seats,
			"premium":  license.Premium,
			"ultimate": license.Ultimate,
			"edition":  license.Edition(),
		},
	}
}
