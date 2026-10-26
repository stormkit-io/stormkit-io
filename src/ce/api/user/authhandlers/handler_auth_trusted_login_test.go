package authhandlers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/authhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

const trustedLoginInstance = "https://api.acme.stormkit.dev"

// fakeProvider is a minimal identity provider: it redeems one-time codes it
// issued for a given instance, each code only once.
type fakeProvider struct {
	mu sync.Mutex

	// codes maps a code to the email it was issued for. Redeeming removes it.
	codes map[string]string

	// instance is the instance URL every code was issued for.
	instance string

	// status, when set, is returned for every request instead.
	status int

	requests []authhandlers.TrustedLoginRedeemRequest
	calls    atomic.Int32
}

func (f *fakeProvider) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.calls.Add(1)

	body := authhandlers.TrustedLoginRedeemRequest{}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.requests = append(f.requests, body)

	if f.status != 0 {
		w.WriteHeader(f.status)
		return
	}

	email, ok := f.codes[body.Code]

	if !ok || body.Instance != f.instance {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	delete(f.codes, body.Code)

	_ = json.NewEncoder(w).Encode(authhandlers.TrustedLoginRedeemResponse{Email: email})
}

// issue registers a one-time code for email and returns it.
func (f *fakeProvider) issue(email string) string {
	f.mu.Lock()
	defer f.mu.Unlock()

	code := "code-" + strings.TrimSpace(email)
	f.codes[code] = email

	return code
}

type HandlerAuthTrustedLoginSuite struct {
	suite.Suite
	*factory.Factory

	conn     databasetest.TestDB
	provider *fakeProvider
	server   *httptest.Server
	usr      *factory.MockUser
	email    string
}

func (s *HandlerAuthTrustedLoginSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)

	s.provider = &fakeProvider{codes: map[string]string{}, instance: trustedLoginInstance}
	s.server = httptest.NewServer(s.provider)
	s.T().Setenv(authhandlers.TrustedLoginURLEnv, s.server.URL+"/redeem")

	s.Require().NoError(admin.Store().DeleteConfig(context.Background()))
	s.setInstanceConfig(admin.InstanceConfig{
		DomainConfig: &admin.DomainConfig{API: trustedLoginInstance},
	})

	s.usr = s.MockUser()
	s.email = s.usr.Emails[0].Address
}

func (s *HandlerAuthTrustedLoginSuite) AfterTest(_, _ string) {
	// The instance config is cached process-wide; rolling back the
	// transaction alone would leak it into the next suite.
	s.NoError(admin.Store().DeleteConfig(context.Background()))
	s.server.Close()
	s.conn.CloseTx()
}

func (s *HandlerAuthTrustedLoginSuite) setInstanceConfig(cfg admin.InstanceConfig) {
	s.Require().NoError(admin.Store().UpsertConfig(context.Background(), cfg))
}

// post sends a trusted login request with the given code.
func (s *HandlerAuthTrustedLoginSuite) post(code string) shttptest.Response {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(authhandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/auth/trusted",
		map[string]string{"code": code},
		nil,
	)
}

// signedInUser returns the user a successful response signed in.
func (s *HandlerAuthTrustedLoginSuite) signedInUser(response shttptest.Response) *user.User {
	body := struct {
		User         *user.User `json:"user"`
		SessionToken string     `json:"sessionToken"`
	}{}

	s.Require().Equal(http.StatusOK, response.Code)
	s.Require().NoError(json.Unmarshal(response.Byte(), &body))

	session := user.ParseJWT(&user.ParseJWTArgs{Bearer: body.SessionToken, Purposes: []user.Purpose{user.PurposeSession}})
	s.Require().NotNil(session)
	s.Equal(body.User.ID.String(), session["uid"])

	return body.User
}

func (s *HandlerAuthTrustedLoginSuite) Test_Success() {
	code := s.provider.issue(s.email)

	usr := s.signedInUser(s.post(code))

	s.Equal(s.usr.ID, usr.ID)
	s.Equal([]authhandlers.TrustedLoginRedeemRequest{{Code: code, Instance: trustedLoginInstance}}, s.provider.requests)
}

// Test_Disabled verifies that the endpoint does not exist on instances that
// trust no identity provider.
func (s *HandlerAuthTrustedLoginSuite) Test_Disabled() {
	s.T().Setenv(authhandlers.TrustedLoginURLEnv, "")

	s.Equal(http.StatusNotFound, s.post(s.provider.issue(s.email)).Code)
	s.Zero(s.provider.calls.Load())
}

// Test_RequiresHTTPS verifies that codes are never sent to the provider over
// plain HTTP, except on loopback.
func (s *HandlerAuthTrustedLoginSuite) Test_RequiresHTTPS() {
	for _, endpoint := range []string{"http://cloud.stormkit.io/redeem", "not a url", "/redeem"} {
		s.T().Setenv(authhandlers.TrustedLoginURLEnv, endpoint)

		response := s.post("code")

		s.Equal(http.StatusInternalServerError, response.Code, endpoint)
		s.Contains(response.String(), "misconfigured", endpoint)
	}

	s.Zero(s.provider.calls.Load())
}

// Test_CodeIsSingleUse verifies that the provider's single-use answer is
// honoured: the second redemption fails.
func (s *HandlerAuthTrustedLoginSuite) Test_CodeIsSingleUse() {
	code := s.provider.issue(s.email)

	s.Equal(http.StatusOK, s.post(code).Code)
	s.Equal(http.StatusUnauthorized, s.post(code).Code)
}

func (s *HandlerAuthTrustedLoginSuite) Test_UnknownCode() {
	s.Equal(http.StatusUnauthorized, s.post("made-up").Code)
}

// Test_CodeForAnotherInstance verifies that the instance identifies itself,
// so a code issued for another instance is refused by the provider.
func (s *HandlerAuthTrustedLoginSuite) Test_CodeForAnotherInstance() {
	code := s.provider.issue(s.email)
	s.provider.instance = "https://api.other.stormkit.dev"

	s.Equal(http.StatusUnauthorized, s.post(code).Code)
}

// Test_InstanceURLIsNormalized verifies that the instance reports the same
// URL to the provider however it was configured.
func (s *HandlerAuthTrustedLoginSuite) Test_InstanceURLIsNormalized() {
	for _, api := range []string{"HTTPS://API.Acme.Stormkit.dev/", "https://api.acme.stormkit.dev:443"} {
		s.setInstanceConfig(admin.InstanceConfig{
			DomainConfig: &admin.DomainConfig{API: api},
		})

		s.signedInUser(s.post(s.provider.issue(s.email)))
	}
}

// Test_UnknownInstanceURL verifies that an instance that does not know its own
// API URL refuses to redeem anything.
func (s *HandlerAuthTrustedLoginSuite) Test_UnknownInstanceURL() {
	s.setInstanceConfig(admin.InstanceConfig{DomainConfig: &admin.DomainConfig{}})

	s.Equal(http.StatusUnauthorized, s.post(s.provider.issue(s.email)).Code)
	s.Zero(s.provider.calls.Load())
}

func (s *HandlerAuthTrustedLoginSuite) Test_InvalidCode() {
	s.Equal(http.StatusBadRequest, s.post("").Code)
	s.Equal(http.StatusBadRequest, s.post(strings.Repeat("a", 513)).Code)
	s.Zero(s.provider.calls.Load())
}

// Test_ProviderDown verifies that an unreachable or failing provider, or one
// asking to retry later, is reported as unavailable, not as a rejected code.
func (s *HandlerAuthTrustedLoginSuite) Test_ProviderDown() {
	for _, status := range []int{http.StatusBadGateway, http.StatusTooManyRequests, http.StatusRequestTimeout} {
		s.provider.status = status
		s.Equal(http.StatusServiceUnavailable, s.post("code").Code, status)
	}

	s.server.Close()
	s.Equal(http.StatusServiceUnavailable, s.post("code").Code)
}

// Test_RedirectIsNotFollowed verifies that the code is never forwarded to
// wherever a redirect points.
func (s *HandlerAuthTrustedLoginSuite) Test_RedirectIsNotFollowed() {
	var followed atomic.Bool

	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		followed.Store(true)
	}))

	defer target.Close()

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	}))

	defer redirector.Close()

	s.T().Setenv(authhandlers.TrustedLoginURLEnv, redirector.URL)

	s.Equal(http.StatusUnauthorized, s.post("code").Code)
	s.False(followed.Load())
}

// Test_MixedCaseEmail verifies that the email is matched case-insensitively.
func (s *HandlerAuthTrustedLoginSuite) Test_MixedCaseEmail() {
	code := s.provider.issue(" " + strings.ToUpper(s.email) + " ")

	s.signedInUser(s.post(code))
}

// Test_UnverifiedEmailIgnored verifies that someone who attached the email to
// their own account without verifying it cannot be signed in for it, and
// that the verified owner is the one signed in.
func (s *HandlerAuthTrustedLoginSuite) Test_UnverifiedEmailIgnored() {
	intruder := s.MockUser(map[string]any{
		"Emails": []oauth.Email{
			{Address: "intruder@acme.com", IsVerified: true, IsPrimary: true},
			{Address: "victim@acme.com", IsVerified: false},
		},
	})

	s.Equal(http.StatusUnauthorized, s.post(s.provider.issue("victim@acme.com")).Code)

	victim := s.MockUser(map[string]any{
		"Emails": []oauth.Email{{Address: "victim@acme.com", IsVerified: true, IsPrimary: true}},
	})

	usr := s.signedInUser(s.post(s.provider.issue("victim@acme.com")))

	s.Equal(victim.ID, usr.ID)
	s.NotEqual(intruder.ID, usr.ID)
}

func (s *HandlerAuthTrustedLoginSuite) Test_UnknownUser() {
	s.Equal(http.StatusUnauthorized, s.post(s.provider.issue("nobody@acme.com")).Code)
}

// Test_PendingUser verifies that the provider vouching for an identity does
// not bypass sign-up approval.
func (s *HandlerAuthTrustedLoginSuite) Test_PendingUser() {
	s.setInstanceConfig(admin.InstanceConfig{
		DomainConfig: &admin.DomainConfig{API: trustedLoginInstance},
		AuthConfig: &admin.AuthConfig{
			UserManagement: admin.UserManagement{SignUpMode: admin.SIGNUP_MODE_OFF},
		},
	})

	pending := s.MockUser(map[string]any{"IsApproved": null.Bool{}})

	s.Equal(http.StatusUnauthorized, s.post(s.provider.issue(pending.Emails[0].Address)).Code)
}

func TestHandlerAuthTrustedLoginSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthTrustedLoginSuite{})
}
