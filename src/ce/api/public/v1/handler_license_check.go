package publicapiv1

import (
	"encoding/json"
	"net/http"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func handlerLicenseCheck(req *shttp.RequestContext) *shttp.Response {
	token := req.Query().Get("token")
	license, _ := licenseFromContent(token)
	var fromDB *admin.License
	var err error

	if license != nil {
		fromDB, err = user.NewStore().LicenseByUserID(req.Context(), license.UserID)
	} else {
		fromDB, err = user.NewStore().LicenseByToken(req.Context(), token)
	}

	if err != nil {
		if err.Error() == "invalid-token" {
			return shttp.BadRequest()
		}

		return shttp.Error(err)
	}

	if fromDB == nil {
		return shttp.NotAllowed()
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"license": map[string]any{
				"premium":  fromDB.Premium,
				"ultimate": fromDB.Ultimate,
				"version":  fromDB.Version,
				"seats":    fromDB.Seats,
			},
		},
	}
}

// Backwards compatibility:
// licenseFromContent takes an encrypted and encoded license as an argument
// and returns a License object from it.
func licenseFromContent(content string) (*admin.License, error) {
	token, err := utils.DecodeString(content)

	if err != nil {
		return nil, err
	}

	license := &admin.License{}

	if err := json.Unmarshal(token, license); err != nil {
		return nil, err
	}

	if license.UserID.String() == "" {
		return nil, nil
	}

	return license, nil
}
