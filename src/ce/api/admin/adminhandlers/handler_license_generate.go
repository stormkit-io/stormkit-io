package adminhandlers

import (
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

type LicenseGenerateRequest struct {
	Seats       int    `json:"seats"`
	Description string `json:"description"`
	IsUltimate  bool   `json:"isUltimate"`
}

func handlerLicenseGenerate(req *user.RequestContext) *shttp.Response {
	data := LicenseGenerateRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if data.Seats > 100 {
		return shttp.BadRequest(map[string]any{
			"error": "maximum allowed seats is 100",
		})
	}

	packageName := config.PackagePremium

	if data.IsUltimate {
		packageName = config.PackageUltimate
	}

	license, err := user.NewStore().GenerateSelfHostedLicense(
		req.Context(),
		data.Seats,
		0,
		packageName,
		map[string]any{
			"description": data.Description,
		})

	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Data: map[string]any{
			"key": license.Key,
		},
	}
}
