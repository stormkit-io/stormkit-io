package authhandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

const (
	// TrustedLoginURLEnv is the identity provider endpoint this instance asks
	// to redeem one-time login codes. Unset disables trusted login.
	TrustedLoginURLEnv = "STORMKIT_TRUSTED_LOGIN_URL"

	trustedLoginTimeout      = 5 * time.Second
	trustedLoginMaxCodeLen   = 512
	trustedLoginMaxReplySize = 64 << 10
)

var (
	// errTrustedLoginRejected means the provider refused the code: unknown,
	// already used, expired or issued for another instance.
	errTrustedLoginRejected = errors.New("login code was rejected")

	// errTrustedLoginUnavailable means the provider could not be asked, so
	// the code may still be valid.
	errTrustedLoginUnavailable = errors.New("identity provider is unavailable")
)

// trustedLoginURLWarning makes sure a malformed provider URL is logged once,
// not on every unauthenticated request.
var trustedLoginURLWarning sync.Once

// TrustedLoginRequest is the body of a trusted login.
type TrustedLoginRequest struct {
	Code string `json:"code"`
}

// TrustedLoginRedeemRequest is what the instance sends to the provider.
type TrustedLoginRedeemRequest struct {
	Code string `json:"code"`

	// Instance is this instance's API URL, so the provider can refuse codes
	// it issued for another instance.
	Instance string `json:"instance"`
}

// TrustedLoginRedeemResponse is what the provider answers for a valid code.
type TrustedLoginRedeemResponse struct {
	Email string `json:"email"`
}

// handlerAuthTrustedLogin signs in an existing user on behalf of an external
// identity provider, such as Stormkit Cloud signing a customer into their
// dedicated instance. The provider hands over a one-time code, and the
// instance asks the provider over HTTPS whom it was issued for. No signing
// keys live on the instance, so there is nothing to leak or rotate here, and
// the provider enforces single use and expiry.
func handlerAuthTrustedLogin(req *shttp.RequestContext) *shttp.Response {
	tl := newTrustedLogin()
	endpoint, err := tl.endpoint()

	if err != nil {
		trustedLoginURLWarning.Do(func() {
			slog.Errorf("trusted login is disabled: %s", err.Error())
		})

		return &shttp.Response{
			Status: http.StatusInternalServerError,
			Data:   map[string]any{"error": "Trusted login is misconfigured on this instance"},
		}
	}

	if endpoint == "" {
		return shttp.NotFound()
	}

	data := TrustedLoginRequest{}

	if err := req.Post(&data); err != nil {
		return shttp.BadRequest(map[string]any{"error": "Invalid request body"})
	}

	if data.Code == "" || len(data.Code) > trustedLoginMaxCodeLen {
		return shttp.BadRequest(map[string]any{"error": "Invalid login code"})
	}

	// Without a known API URL the provider cannot tell which instance is
	// asking, and a code issued for any instance could be redeemed here.
	cfg, err := admin.Store().Config(req.Context())

	if err != nil {
		return shttp.Error(err)
	}

	instance := tl.normalizeURL(cfg.ApiURL(""))

	if instance == "" {
		slog.Errorf("trusted login rejected: instance API URL is unknown")
		return shttp.NotAllowed()
	}

	email, err := tl.redeem(redeemTrustedLoginParams{
		ctx:      req.Context(),
		endpoint: endpoint,
		body:     TrustedLoginRedeemRequest{Code: data.Code, Instance: instance},
	})

	// A client that went away cancels the request; that says nothing about
	// the provider, so it must not show up as a provider outage.
	if err != nil && req.Context().Err() != nil {
		slog.Infof("trusted login abandoned by the client: %s", req.Context().Err().Error())
		return &shttp.Response{Status: http.StatusServiceUnavailable}
	}

	if errors.Is(err, errTrustedLoginUnavailable) {
		slog.Errorf("trusted login: %s", err.Error())
		return &shttp.Response{Status: http.StatusServiceUnavailable}
	}

	if err != nil {
		slog.Infof("trusted login rejected: %s", err.Error())
		return shttp.NotAllowed()
	}

	store := user.NewStore()
	users, err := store.UsersByVerifiedEmail(req.Context(), email)

	if err != nil {
		return shttp.Error(err)
	}

	if len(users) != 1 {
		slog.Infof("trusted login rejected: %d users own the verified email", len(users))
		return shttp.NotAllowed()
	}

	usr := users[0]

	// The provider vouches for who the user is, not for their access: a
	// pending or rejected sign-up stays locked out, as it is for OAuth.
	if !usr.IsAuthorizedToLogin() {
		slog.Infof("trusted login rejected: user %s is not approved", usr.ID.String())
		return shttp.NotAllowed()
	}

	session, err := user.JWT(user.JWTParams{Purpose: user.PurposeSession, Claims: jwt.MapClaims{
		"uid": usr.ID.String(),
	}})

	if err != nil {
		return shttp.Error(err)
	}

	if err := store.UpdateLastLogin(req.Context(), usr.ID); err != nil {
		return shttp.Error(err)
	}

	slog.Infof("trusted login: signed in user %s", usr.ID.String())

	return &shttp.Response{
		Status: http.StatusOK,
		Data: map[string]any{
			"user":         usr.JSON(),
			"sessionToken": session,
		},
	}
}

// trustedLoginClient is shared by all requests so connections to the
// provider are reused.
var trustedLoginClient = &http.Client{
	Timeout: trustedLoginTimeout,

	// A redirect would send the code to wherever the response points, so
	// redirects are answered as a rejection instead.
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

type trustedLogin struct {
	client *http.Client
}

func newTrustedLogin() trustedLogin {
	return trustedLogin{client: trustedLoginClient}
}

// endpoint returns the provider's redeem URL, or "" when trusted login is off.
// Only HTTPS is accepted, except on loopback for local development: the
// provider's TLS certificate is what proves its answers are genuine.
func (trustedLogin) endpoint() (string, error) {
	raw := strings.TrimSpace(os.Getenv(TrustedLoginURLEnv))

	if raw == "" {
		return "", nil
	}

	u, err := url.Parse(raw)

	if err != nil || u.Host == "" {
		return "", fmt.Errorf("%s must be an absolute URL", TrustedLoginURLEnv)
	}

	if u.Scheme == "https" {
		return raw, nil
	}

	if ip := net.ParseIP(u.Hostname()); u.Scheme == "http" && (u.Hostname() == "localhost" || (ip != nil && ip.IsLoopback())) {
		return raw, nil
	}

	return "", fmt.Errorf("%s must use https", TrustedLoginURLEnv)
}

type redeemTrustedLoginParams struct {
	ctx      context.Context
	endpoint string
	body     TrustedLoginRedeemRequest
}

// redeem asks the provider whom the code was issued for and returns the
// user's email, lowercased. It returns errTrustedLoginRejected when the
// provider refuses the code and errTrustedLoginUnavailable when it cannot
// give an answer.
func (tl trustedLogin) redeem(p redeemTrustedLoginParams) (string, error) {
	payload, err := json.Marshal(p.body)

	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(p.ctx, http.MethodPost, p.endpoint, bytes.NewReader(payload))

	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := tl.client.Do(req)

	if err != nil {
		// url.Error repeats the endpoint, which may carry credentials in its
		// userinfo or query, so only the underlying cause is kept.
		var urlErr *url.Error

		if errors.As(err, &urlErr) {
			err = urlErr.Err
		}

		return "", fmt.Errorf("%w: %s", errTrustedLoginUnavailable, err.Error())
	}

	defer res.Body.Close()

	// Server errors and the 4xx statuses that mean "try again later" leave
	// the code untouched, so they must not read as a refusal.
	switch {
	case res.StatusCode >= 500,
		res.StatusCode == http.StatusRequestTimeout,
		res.StatusCode == http.StatusTooEarly,
		res.StatusCode == http.StatusTooManyRequests:
		return "", fmt.Errorf("%w: HTTP %d", errTrustedLoginUnavailable, res.StatusCode)
	}

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: HTTP %d", errTrustedLoginRejected, res.StatusCode)
	}

	reply := TrustedLoginRedeemResponse{}

	if err := json.NewDecoder(io.LimitReader(res.Body, trustedLoginMaxReplySize)).Decode(&reply); err != nil {
		return "", fmt.Errorf("%w: unreadable reply: %s", errTrustedLoginUnavailable, err.Error())
	}

	email := strings.ToLower(strings.TrimSpace(reply.Email))

	if email == "" {
		return "", fmt.Errorf("%w: reply has no email", errTrustedLoginRejected)
	}

	return email, nil
}

// normalizeURL lowercases the scheme and host and drops a default port and a
// trailing slash, so the provider sees the same instance URL however it was
// configured here. Anything that is not an absolute URL normalizes to "".
func (trustedLogin) normalizeURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))

	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}

	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())

	if port := u.Port(); port != "" && !(scheme == "https" && port == "443") && !(scheme == "http" && port == "80") {
		host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}

	return scheme + "://" + host + strings.TrimRight(u.EscapedPath(), "/")
}
