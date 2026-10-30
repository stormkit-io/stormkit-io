package authwallhandlers_test

import (
	"context"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/authwall/authwallhandlers"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stretchr/testify/suite"
)

type HandlerAuthMagicSuite struct {
	suite.Suite
	*factory.Factory
	conn    databasetest.TestDB
	app     *factory.MockApp
	envID   types.ID
	envName string
	aw      *authwall.AuthWall
	sent    []authwall.SendMagicLinkParams
	restore func()
}

func (s *HandlerAuthMagicSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.app = s.MockApp(s.MockUser())
	env := s.MockEnv(s.app)
	s.envID = env.ID
	s.envName = env.Name
	s.aw = &authwall.AuthWall{
		TeamID:     s.app.TeamID,
		LoginEmail: "client@example.org",
		Metadata:   `{"role":"client"}`,
		MagicLink:  true,
		AppIDs:     []types.ID{s.app.ID},
	}

	s.Require().NoError(authwall.Store().CreateLogin(context.Background(), s.aw))
	s.stubEmail(true)
}

func (s *HandlerAuthMagicSuite) AfterTest(_, _ string) {
	s.restore()
	s.conn.CloseTx()
}

// stubEmail captures sign-in links instead of sending them.
func (s *HandlerAuthMagicSuite) stubEmail(available bool) {
	if s.restore != nil {
		s.restore()
	}

	s.sent = nil
	s.restore = authwallhandlers.StubMagicLinkEmail(available, func(p authwall.SendMagicLinkParams) {
		s.sent = append(s.sent, p)
	})
}

func (s *HandlerAuthMagicSuite) handler() http.Handler {
	return shttp.NewRouter().RegisterService(authwallhandlers.Services).Router().Handler()
}

// post submits form fields to path.
func (s *HandlerAuthMagicSuite) post(path string, fields map[string]string) shttptest.Response {
	values := map[string][]byte{}

	for k, v := range fields {
		values[k] = []byte(v)
	}

	body, contentType, err := shttptest.MultipartForm(values, nil)
	s.Require().NoError(err)

	return shttptest.RequestWithHeaders(s.handler(), shttp.MethodPost, path, body, map[string]string{
		"Content-Type": contentType,
		"Referer":      attackerPage,
	})
}

// requestLink submits email on the login form of the suite's environment.
func (s *HandlerAuthMagicSuite) requestLink(email string) shttptest.Response {
	token, err := authwall.Token{EnvID: s.envID}.Form(protectedPage)
	s.Require().NoError(err)

	return s.post("/auth-wall/magic", map[string]string{
		"email": email,
		"envId": s.envID.String(),
		"token": token,
	})
}

// link returns the query of the only sign-in link sent.
func (s *HandlerAuthMagicSuite) link() url.Values {
	s.Require().Len(s.sent, 1)

	u, err := url.Parse(s.sent[0].URL)
	s.Require().NoError(err)
	s.Equal("/auth-wall/magic", u.Path)
	s.Equal([]string{"id"}, slices.Collect(maps.Keys(u.Query())), "the link carries nothing but its ID")

	return u.Query()
}

func (s *HandlerAuthMagicSuite) verify(query url.Values) shttptest.Response {
	return s.post("/auth-wall/magic/verify", map[string]string{
		"id": query.Get("id"),
	})
}

// Test_SignsIn verifies the whole flow: the link is sent to the login, opening
// it does not use it up, and continuing signs the visitor in once.
func (s *HandlerAuthMagicSuite) Test_SignsIn() {
	response := s.requestLink("CLIENT@example.org")

	s.Equal(http.StatusOK, response.Code)
	s.Contains(response.String(), "Check your inbox")
	s.Equal("client@example.org", s.sent[0].To)
	s.Equal(s.app.DisplayName+" / "+s.envName, s.sent[0].Name)

	query := s.link()

	for range 2 {
		page := shttptest.Request(s.handler(), shttp.MethodGet, "/auth-wall/magic?"+query.Encode(), nil)

		body := page.String()

		s.Equal(http.StatusOK, page.Code)
		s.Contains(body, "Continue")
		s.Contains(body, "client@example.org")
		s.Equal("no-store", page.Header().Get("Cache-Control"))
	}

	response = s.verify(query)
	s.Equal(http.StatusFound, response.Code)

	location, err := url.Parse(response.Header().Get("Location"))
	s.Require().NoError(err)
	s.Equal("site.example.org", location.Host)
	s.Equal("/page", location.Path)

	session := authwall.Token{EnvID: s.envID}.ParseSession(location.Query().Get("stormkit_success"))
	s.Require().NotNil(session)
	s.Equal("client@example.org", session.Email)
	s.Equal(`{"role":"client"}`, session.Metadata)

	used := s.verify(query)
	s.Equal(http.StatusBadRequest, used.Code)
	s.Contains(used.String(), "This link no longer works")
}

// Test_SamePage_NoLink verifies that emails without a login, logins without a
// magic link and logins of other apps get the same page, and no link.
func (s *HandlerAuthMagicSuite) Test_SamePage_NoLink() {
	passwordOnly := &authwall.AuthWall{
		TeamID:        s.app.TeamID,
		LoginEmail:    "password@example.org",
		LoginPassword: "password123",
		AppIDs:        []types.ID{s.app.ID},
	}

	otherApp := &authwall.AuthWall{
		TeamID:     s.app.TeamID,
		LoginEmail: "other@example.org",
		MagicLink:  true,
		AppIDs:     []types.ID{s.MockApp(nil, map[string]any{"TeamID": s.app.TeamID}).ID},
	}

	s.Require().NoError(authwall.Store().CreateLogin(context.Background(), passwordOnly))
	s.Require().NoError(authwall.Store().CreateLogin(context.Background(), otherApp))

	for _, email := range []string{"nobody@example.org", passwordOnly.LoginEmail, otherApp.LoginEmail} {
		response := s.requestLink(email)

		s.Equal(http.StatusOK, response.Code, email)
		s.Contains(response.String(), "Check your inbox", email)
	}

	s.Empty(s.sent)
}

func (s *HandlerAuthMagicSuite) Test_InvalidEmail() {
	response := s.requestLink("not-an-email")

	s.Contains(response.Header().Get("Location"), "stormkit_error=invalid_email")
	s.Empty(s.sent)
}

func (s *HandlerAuthMagicSuite) Test_Unavailable() {
	s.stubEmail(false)

	response := s.requestLink(s.aw.LoginEmail)

	s.Contains(response.Header().Get("Location"), "stormkit_error=magic_link_unavailable")
	s.Empty(s.sent)
}

// Test_Revoked verifies that a link stops working once its login can no longer
// use magic links.
func (s *HandlerAuthMagicSuite) Test_Revoked() {
	s.requestLink(s.aw.LoginEmail)
	query := s.link()

	s.aw.MagicLink = false
	_, err := authwall.Store().UpdateLogin(context.Background(), authwall.UpdateLoginParams{Login: s.aw, Password: "password123"})
	s.Require().NoError(err)

	s.Equal(http.StatusBadRequest, s.verify(query).Code)
}

// Test_UnknownLink verifies that only links Stormkit sent sign visitors in.
func (s *HandlerAuthMagicSuite) Test_UnknownLink() {
	query := url.Values{"id": []string{"made-up"}}

	s.Equal(http.StatusBadRequest, s.verify(query).Code)
	s.Equal(http.StatusBadRequest, s.verify(url.Values{}).Code)

	page := shttptest.Request(s.handler(), shttp.MethodGet, "/auth-wall/magic?"+query.Encode(), nil)
	s.Equal(http.StatusBadRequest, page.Code)
}

// Test_NoPassword verifies that a login without a password cannot log in with
// any password, including an empty one.
func (s *HandlerAuthMagicSuite) Test_NoPassword() {
	token, err := authwall.Token{EnvID: s.envID}.Form(protectedPage)
	s.Require().NoError(err)

	for _, password := range []string{"", " ", "anything"} {
		response := s.post("/auth-wall/login", map[string]string{
			"email":    s.aw.LoginEmail,
			"password": password,
			"envId":    s.envID.String(),
			"token":    token,
		})

		location := response.Header().Get("Location")
		s.True(strings.Contains(location, "stormkit_error=invalid_credentials"), password)
	}
}

func TestHandlerAuthMagicSuite(t *testing.T) {
	suite.Run(t, &HandlerAuthMagicSuite{})
}
