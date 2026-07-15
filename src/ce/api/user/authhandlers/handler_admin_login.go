package authhandlers

import (
	"crypto/subtle"
	"net/http"
	"strings"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"golang.org/x/crypto/bcrypt"
)

// verifyAdminPassword reports whether provided matches the stored admin
// secret. New registrations store a bcrypt hash; instances created before the
// bcrypt migration stored an AES-encrypted plaintext. The legacy path is still
// accepted with a constant-time compare, and legacy reports true so the caller
// can upgrade the stored value to bcrypt.
func verifyAdminPassword(stored, provided string) (ok bool, legacy bool) {
	if strings.HasPrefix(stored, "$2") {
		return bcrypt.CompareHashAndPassword([]byte(stored), []byte(provided)) == nil, false
	}

	decrypted := utils.DecryptToString(stored)

	return subtle.ConstantTimeCompare([]byte(decrypted), []byte(provided)) == 1, true
}

func handlerAdminLogin(req *shttp.RequestContext) *shttp.Response {
	if res := hasAnyProviderEnabled(); res != nil {
		return res
	}

	data := AdminLoginRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.Error(err)
	}

	if res := validateAdminLoginRequest(data); res != nil {
		return res
	}

	store := user.NewStore()
	usr, err := store.UserByEmail(req.Context(), []string{data.Email})

	if err != nil {
		return shttp.Error(err)
	}

	if usr == nil {
		return shttp.NotAllowed()
	}

	cfg, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	if cfg.AdminUserConfig == nil {
		return shttp.NotAllowed()
	}

	if !strings.EqualFold(cfg.AdminUserConfig.Email, data.Email) {
		return shttp.NotAllowed()
	}

	ok, legacy := verifyAdminPassword(cfg.AdminUserConfig.Password, data.Password)

	if !ok {
		return shttp.NotAllowed()
	}

	// Transparently migrate a legacy AES-encrypted secret to a bcrypt hash on
	// the first successful login. Best-effort: a failed write must not block
	// sign-in, the next login simply retries.
	if legacy {
		if hash, err := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost); err == nil {
			cfg.AdminUserConfig.Password = string(hash)

			if err := admin.Store().UpsertConfig(req.Context(), cfg); err != nil {
				slog.Errorf("admin login: failed to upgrade password hash: %s", err.Error())
			}
		}
	}

	if err := user.NewStore().UpdateLastLogin(req.Context(), usr.ID); err != nil {
		return errorResponse(err, 0)
	}

	jwt, err := user.JWT(jwt.MapClaims{
		"uid": usr.ID.String(),
	})

	// Creating new token failed
	if err != nil {
		return shttp.Error(err)
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"user":         usr.JSON(),
			"sessionToken": jwt,
		},
	}
}
