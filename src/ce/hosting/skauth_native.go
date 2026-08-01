package hosting

import (
	"errors"
	"net/http"
	"regexp"
	"time"

	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

// nativeCodeTTL bounds how long a deep-link code stays redeemable.
const nativeCodeTTL = time.Minute

// nativeCodeStore holds the short-lived codes that hand a session token to a
// native app's custom-scheme deep link. It reuses the same single-use Redis
// store as the OAuth authorization codes; the prefix keeps the keyspaces apart.
//
// The TTL is deliberately much shorter than the OAuth code's: the app redeems
// it the moment the OS hands over the deep link, so a minute is generous, and
// every second beyond that is a window an interceptor could race.
var nativeCodeStore = oneTimeCode{prefix: "skauth:native:", ttl: nativeCodeTTL}

// nativeSessionCode is the Redis-stashed payload behind a deep-link code.
type nativeSessionCode struct {
	Token         string `json:"token"`
	CodeChallenge string `json:"codeChallenge"`
}

// pkceChallengePattern matches an S256 code_challenge: base64url(sha256(...))
// unpadded, which is always exactly 43 characters from the unreserved set.
var pkceChallengePattern = regexp.MustCompile(`^[A-Za-z0-9\-._~]{43}$`)

// validateNativeChallenge gates custom-scheme delivery on PKCE.
//
// A private-use URI scheme is not exclusively claimable — on Android any app may
// declare the same scheme, and on iOS a duplicate registration has an undefined
// winner — so a redirect to one must be assumed interceptable. RFC 8252 §8.1
// answers that by carrying only a code bound to a proof the interceptor does not
// have. Custom-scheme sign-in is therefore refused outright without a challenge,
// rather than falling back to putting the session token in the URL.
//
// http(s) targets are browsers using the session cookie and are unaffected; any
// challenge they send is ignored.
func validateNativeChallenge(origin, challenge, method string) *shttp.Response {
	if !isNativeSchemeOrigin(origin) {
		return nil
	}

	if method != "" && method != "S256" {
		return shttp.BadRequest(map[string]any{
			"errors": []string{"code_challenge_method must be S256"},
		})
	}

	if !pkceChallengePattern.MatchString(challenge) {
		return shttp.BadRequest(map[string]any{
			"errors": []string{"a PKCE code_challenge using S256 is required for a custom-scheme redirect"},
		})
	}

	return nil
}

// handleNativeToken serves POST /_stormkit/auth/token: the redemption half of
// the deep-link handover. The app posts the code it received on its deep link
// together with the code_verifier whose challenge it sent at sign-in, and gets
// the session token back over TLS instead of through the OS.
//
// The code is single-use (GetDel) and short-lived, so an interceptor that also
// sees it wins nothing without the verifier, and a replay after the real app has
// redeemed finds nothing at all.
func (m *skAuthMiddleware) handleNativeToken() (*shttp.Response, error) {
	req := m.req.RequestContext

	body := &struct {
		Code     string `json:"code"`
		Verifier string `json:"code_verifier"`
	}{}

	_ = req.Post(body)

	code := utils.GetString(body.Code, req.FormValue("code"))
	verifier := utils.GetString(body.Verifier, req.FormValue("code_verifier"))

	if code == "" || verifier == "" {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data:   map[string]any{"errors": []string{"code and code_verifier are required"}},
		}, nil
	}

	var payload nativeSessionCode

	if err := nativeCodeStore.redeem(req.Context(), code, &payload); err != nil {
		// Unknown, expired and already-redeemed codes are indistinguishable to the
		// caller on purpose, so the endpoint cannot be used to probe code validity.
		if !errors.Is(err, errCodeNotFound) {
			slog.Errorf("native token exchange: failed to redeem code: %s", err.Error())
		}

		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data:   map[string]any{"errors": []string{"code is invalid or expired"}},
		}, nil
	}

	if !verifyPKCE(verifier, payload.CodeChallenge) {
		return &shttp.Response{
			Status: http.StatusBadRequest,
			Data:   map[string]any{"errors": []string{"PKCE verification failed"}},
		}, nil
	}

	return &shttp.Response{
		Status: http.StatusOK,
		Headers: shttp.HeadersFromMap(map[string]string{
			"Cache-Control": "no-store",
		}),
		Data: map[string]any{"token": payload.Token},
	}, nil
}
