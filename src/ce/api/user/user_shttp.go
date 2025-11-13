package user

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/apikey"
	"github.com/stormkit-io/stormkit-io/src/lib/config"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttperr"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// RequestContext is the argument that the AppHandler handler receives.
// Handlers using this endpoint are authenticated handlers.
type RequestContext struct {
	*shttp.RequestContext
	User *User
}

// JWT returns a new JWT token instance.
func (r *RequestContext) JWT(values jwt.MapClaims) (string, error) {
	return JWT(values)
}

// License returns the license associated with the user in the request context.
// If the user is nil, it returns a nil license.
// For self-hosted instances, it returns the instance license.
func (r *RequestContext) License() *admin.License {
	return License(r.User)
}

// License returns the license associated with the given user.
// For self-hosted instances, it returns the instance license.
func License(user *User) *admin.License {
	if config.IsSelfHosted() {
		return admin.CurrentLicense()
	}

	if user == nil {
		return nil
	}

	return &admin.License{
		Seats:      user.Metadata.SeatsPurchased,
		Enterprise: user.Metadata.PackageName == config.PackagePremium || user.Metadata.PackageName == config.PackageUltimate,
	}
}

var ErrEnterpriseOnly = shttperr.New(http.StatusPaymentRequired, "This is an enterprise-only feature. Please upgrade your subscription to continue.", "enterprise-only")

// WithEE is a guard for enterprise-only endpoints.
func WithEE(req *shttp.RequestContext) *shttp.Response {
	usr, _ := FromContext(req)
	license := License(usr)

	if license == nil || !license.Enterprise {
		return shttp.Error(ErrEnterpriseOnly)
	}

	return nil
}

// JWT returns a signed JWT token string.
func JWT(values jwt.MapClaims) (string, error) {
	claims := make(jwt.MapClaims)
	claims["issued"] = time.Now().Unix()

	for k, v := range values {
		claims[k] = v
	}

	token := jwt.New(jwt.GetSigningMethod("HS256"))
	token.Claims = claims

	return token.SignedString([]byte(config.AppSecret()))
}

// FromContext fetches the user object from the request.
func FromContext(req *shttp.RequestContext) (*User, error) {
	uid := uidFromRequest(req)

	if uid == 0 {
		return nil, nil
	}

	return NewStore().UserByID(uid)
}

func WithAPIKey(handler func(*RequestContext) *shttp.Response) shttp.RequestFunc {
	return func(rc *shttp.RequestContext) *shttp.Response {
		token := strings.Replace(rc.Headers().Get("Authorization"), "Bearer ", "", 1)
		key, err := apikey.NewStore().APIKey(rc.Context(), token)

		if err != nil {
			return shttp.Error(err)
		}

		if key == nil || (key.UserID == 0) || key.Scope != apikey.SCOPE_USER {
			return shttp.Forbidden()
		}

		user, err := NewStore().UserByID(key.UserID)

		if err != nil {
			return shttp.Error(err)
		}

		if user == nil {
			return shttp.Forbidden()
		}

		return handler(&RequestContext{
			RequestContext: rc,
			User:           user,
		})
	}
}

// WithAuth is a wrapper for authenticated handlers.
// Wrap the handler with this function like: WithAuth(myHandler)
func WithAuth(handler func(*RequestContext) *shttp.Response) shttp.RequestFunc {
	return func(req *shttp.RequestContext) *shttp.Response {
		usr, err := FromContext(req)

		if err != nil {
			return shttp.UnexpectedError(err)
		}

		if usr == nil {
			return shttp.NotAllowed()
		}

		if config.IsSelfHosted() && admin.MustConfig().SignUpMode() != admin.SIGNUP_MODE_ON {
			if !usr.IsApproved.Valid {
				return &shttp.Response{
					Status: http.StatusForbidden,
					Data: map[string]any{
						"error": "Your account is pending approval by an administrator.",
					},
				}
			}

			if !usr.IsApproved.ValueOrZero() {
				return &shttp.Response{
					Status: http.StatusForbidden,
					Data: map[string]any{
						"error": "Your account is not approved by an administrator. You cannot access this resource.",
					},
				}
			}
		}

		return handler(&RequestContext{
			RequestContext: req,
			User:           usr,
		})
	}
}

// WithAdmin is a wrapper for admin authenticated handlers.
// Wrap the handler with this function like: WithAdmin(myHandler)
func WithAdmin(handler func(*RequestContext) *shttp.Response) shttp.RequestFunc {
	return func(req *shttp.RequestContext) *shttp.Response {
		uid := uidFromRequest(req)

		if uid != 0 {
			user, err := NewStore().UserByID(uid)

			if err != nil {
				return shttp.UnexpectedError(err)
			}

			if user != nil && user.IsAdmin {
				return handler(&RequestContext{
					RequestContext: req,
					User:           user,
				})
			}
		}

		return shttp.NotAllowed()
	}
}

type ParseJWTArgs struct {
	Bearer  string
	Secret  string
	MaxMins int
}

// ParseJWT parses the given token and returns the claims.
func ParseJWT(args *ParseJWTArgs) jwt.MapClaims {
	token, err := jwt.Parse(args.Bearer, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}

		secret := args.Secret

		if secret == "" {
			secret = config.AppSecret()
		}

		return []byte(secret), nil
	})

	if err != nil {
		return nil
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		timestamp, _ := claims["issued"].(float64) // jwt stores as a float 64...
		issued := time.Unix(int64(timestamp), 0)

		// 1 day
		maxMins := float64(24 * 60)

		if args.MaxMins > 0 {
			maxMins = float64(args.MaxMins)
		}

		// Is expired?
		if time.Since(issued).Minutes() > maxMins {
			return nil
		}

		return claims
	}

	return nil
}

// ParseBearer parses the Authorization header and returns the token.
func ParseBearer(token string) string {
	split := strings.Split(token, "Bearer ")

	if len(split) > 1 {
		return split[1]
	}

	return ""
}

// uidFromRequest returns the user id from the given request context.
func uidFromRequest(req *shttp.RequestContext) types.ID {
	var bearer string
	auth := req.Headers().Get("Authorization")

	if auth == "" {
		bearer = req.Query().Get("auth")
	} else {
		bearer = ParseBearer(auth)
	}

	claims := ParseJWT(&ParseJWTArgs{Bearer: bearer})

	switch claims["uid"].(type) {
	case int64:
		return types.ID(claims["uid"].(int64))
	case string:
		return utils.StringToID(claims["uid"].(string))
	default:
		return 0
	}
}
