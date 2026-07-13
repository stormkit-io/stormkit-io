package publicapiv1

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type LicenseCheckRequest struct {
	Token string `json:"token"`
}

func handlerLicenseCheck(req *shttp.RequestContext) *shttp.Response {
	data := LicenseCheckRequest{}

	if req.Method == http.MethodGet {
		data.Token = req.Query().Get("token")
	} else if req.Method == http.MethodPost {
		if err := req.Post(&data); err != nil {
			return shttp.Error(err)
		}
	}

	token := normalizeLicenseKey(data.Token)

	if token == "" {
		return shttp.BadRequest(map[string]any{"errors": []string{"token is required"}})
	}

	fromDB, err := user.NewStore().License(req.Context(), user.LicenseParams{Token: token})

	if err != nil {
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

// Older keys might have `userId:key` format, we need to normalize them before checking the database.
func normalizeLicenseKey(key string) string {
	if strings.Contains(key, ":") {
		pieces := strings.SplitN(key, ":", 2)
		return pieces[1]
	}

	// Backwards compatibility for older licenses that were generated with the old format, to be removed in the future.
	license := admin.License{}
	encodedKey, err := utils.DecodeString(key)

	if err != nil {
		return key
	}

	if err := json.Unmarshal([]byte(encodedKey), &license); err == nil {
		return license.Key
	}

	return key
}
