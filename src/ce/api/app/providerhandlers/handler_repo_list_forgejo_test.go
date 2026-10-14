package providerhandlers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/providerhandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/forgejo"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
)

type HandlerRepoListForgejoSuite struct {
	suite.Suite
	*factory.Factory

	conn   databasetest.TestDB
	server *httptest.Server
	pages  map[string][]map[string]any

	// lastPage is the last page the server links to.
	lastPage int
}

func (s *HandlerRepoListForgejoSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.pages = map[string][]map[string]any{}

	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/user/repos" || r.Header.Get("Authorization") != "Bearer forgejo-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))

		if page < s.lastPage {
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/user/repos?page=%d>; rel="next"`, s.server.URL, page+1))
		} else {
			w.Header().Set("Link", fmt.Sprintf(`<%s/api/v1/user/repos?page=1>; rel="first"`, s.server.URL))
		}

		_ = json.NewEncoder(w).Encode(s.pages[r.URL.Query().Get("page")])
	}))

	cnf := admin.MustConfig().Clone()
	cnf.AuthConfig = &admin.AuthConfig{
		Forgejo: admin.ForgejoConfig{
			BaseURL:      s.server.URL,
			ClientID:     "client-id",
			ClientSecret: "client-secret",
		},
	}

	admin.SetConfig(&cnf)
}

func (s *HandlerRepoListForgejoSuite) AfterTest(_, _ string) {
	s.server.Close()
	s.conn.CloseTx()
	admin.ResetCache(context.Background())
}

// connectedUser creates a user with a Forgejo token.
func (s *HandlerRepoListForgejoSuite) connectedUser() *factory.MockUser {
	usr := s.MockUser(nil)

	err := oauth.NewStore().UpsertToken(usr.ID, &oauth.User{
		ProviderName: forgejo.ProviderName,
		DisplayName:  "jdoe",
		Token: &oauth2.Token{
			AccessToken: "forgejo-token",
			TokenType:   "bearer",
			Expiry:      time.Now().Add(time.Hour),
		},
	})

	s.Require().NoError(err)
	return usr
}

func (s *HandlerRepoListForgejoSuite) get(usr *factory.MockUser, query string) shttptest.Response {
	return shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(providerhandlers.Services).Router().Handler(),
		shttp.MethodGet,
		"/provider/forgejo/repos"+query,
		nil,
		map[string]string{"Authorization": usertest.Authorization(usr.ID)},
	)
}

func (s *HandlerRepoListForgejoSuite) Test_Success() {
	s.lastPage = 6
	s.pages["1"] = []map[string]any{
		{"name": "app", "full_name": "stormkit/app"},
		{"name": "www", "full_name": "stormkit/www"},
	}

	response := s.get(s.connectedUser(), "")

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{
		"hasNextPage": true,
		"repos": [
			{ "name": "app", "fullName": "stormkit/app" },
			{ "name": "www", "fullName": "stormkit/www" }
		]
	}`, response.String())
}

// Test_Search verifies that a search walks every page, since the API
// cannot filter the user's repositories by name.
func (s *HandlerRepoListForgejoSuite) Test_Search() {
	s.lastPage = 2
	s.pages["1"] = make([]map[string]any, 50)

	for i := range s.pages["1"] {
		s.pages["1"][i] = map[string]any{"name": "other", "full_name": "stormkit/other"}
	}

	s.pages["2"] = []map[string]any{{"name": "app", "full_name": "stormkit/app"}}

	response := s.get(s.connectedUser(), "?search=App")

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(`{
		"hasNextPage": false,
		"repos": [{ "name": "app", "fullName": "stormkit/app" }]
	}`, response.String())
}

func (s *HandlerRepoListForgejoSuite) Test_NotConnected() {
	response := s.get(s.MockUser(nil), "")

	s.Equal(http.StatusUnauthorized, response.Code)
}

func TestHandlerRepoListForgejoSuite(t *testing.T) {
	suite.Run(t, new(HandlerRepoListForgejoSuite))
}
