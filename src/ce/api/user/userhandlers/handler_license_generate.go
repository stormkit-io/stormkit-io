package userhandlers

import (
	"errors"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
)

func handlerLicenseGenerate(req *user.RequestContext) *shttp.Response {
	if req.User.Metadata.PackageName != config.PackagePremium &&
		req.User.Metadata.PackageName != config.PackageUltimate {
		return shttp.BadRequest(map[string]any{
			"error": "User has not purchased any seats",
		})
	}

	license, err := user.NewStore().GenerateSelfHostedLicense(
		req.Context(),
		req.User.Metadata.SeatsPurchased,
		req.User.ID,
		req.User.Metadata.PackageName,
		nil,
	)

	if err != nil {
		return shttp.Error(err)
	}

	if license == nil {
		return shttp.Error(errors.New("failed to generate license"))
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"key": license.Token(),
		},
	}
}
