package forgejo

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
)

type ForgejoSuite struct {
	suite.Suite

	conn     databasetest.TestDB
	server   *httptest.Server
	mux      *http.ServeMux
	client   *Forgejo
	requests []*http.Request
	bodies   []map[string]any
}

func (s *ForgejoSuite) SetupSuite() {
	utils.SetAppKey([]byte("12345678901234567890123456789012"))
}

func (s *ForgejoSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.mux = http.NewServeMux()
	s.requests = nil
	s.bodies = nil

	s.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.requests = append(s.requests, r)

		if r.Body != nil {
			data, _ := io.ReadAll(r.Body)
			body := map[string]any{}
			_ = json.Unmarshal(data, &body)
			s.bodies = append(s.bodies, body)
		}

		s.mux.ServeHTTP(w, r)
	}))

	admin.SetConfig(&admin.InstanceConfig{
		DomainConfig: &admin.DomainConfig{
			API:      "https://api.example.org",
			Webhooks: "https://api.example.org",
		},
		AuthConfig: &admin.AuthConfig{
			Forgejo: admin.ForgejoConfig{
				BaseURL:      s.server.URL,
				ClientID:     "client-id",
				ClientSecret: "client-secret",
				RedirectURL:  "https://api.example.org/auth/forgejo/callback",
			},
		},
	})

	s.client = newClientWithToken(&oauth2.Token{AccessToken: "forgejo-token"})
}

func (s *ForgejoSuite) AfterTest(_, _ string) {
	s.server.Close()
	s.conn.CloseTx()
	admin.SetConfig(nil)
}

// reply registers a JSON handler for the given pattern.
func (s *ForgejoSuite) reply(pattern string, status int, body any, headers ...map[string]string) {
	s.mux.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {
		for _, h := range headers {
			for k, v := range h {
				w.Header().Set(k, v)
			}
		}

		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	})
}

func (s *ForgejoSuite) Test_AuthCodeURL() {
	address := AuthCodeURL("my-state")

	s.Contains(address, s.server.URL+"/login/oauth/authorize?")
	s.Contains(address, "client_id=client-id")
	s.Contains(address, "state=my-state")
}

func (s *ForgejoSuite) Test_CloneURL() {
	s.Equal(s.server.URL+"/stormkit/app.git", CloneURL("stormkit/app"))
}

func (s *ForgejoSuite) Test_UserProfile() {
	s.reply("GET /api/v1/user", http.StatusOK, map[string]any{
		"login":      "jdoe",
		"full_name":  "Jane Doe",
		"avatar_url": "https://git.example.org/avatar.png",
		"html_url":   "https://git.example.org/jdoe",
	})

	s.reply("GET /api/v1/user/emails", http.StatusOK, []map[string]any{
		{"email": "jane@example.org", "primary": true, "verified": true},
		{"email": "jane@work.example.org", "primary": false, "verified": false},
	})

	usr, err := s.client.UserProfile()

	s.NoError(err)
	s.Equal("jdoe", usr.DisplayName)
	s.Equal("Jane Doe", usr.FullName)
	s.Equal("https://git.example.org/jdoe", usr.AccountURI)
	s.Equal(ProviderName, usr.ProviderName)
	// Unconfirmed addresses must not sign in to the accounts they name.
	s.Equal([]oauth.Email{
		{Address: "jane@example.org", IsPrimary: true, IsVerified: true},
	}, usr.Emails)
	s.Equal("Bearer forgejo-token", s.requests[0].Header.Get("Authorization"))
}

func (s *ForgejoSuite) Test_ListRepos_HasNextPage() {
	s.reply("GET /api/v1/user/repos", http.StatusOK, []map[string]any{
		{"name": "app", "full_name": "stormkit/app"},
		{"name": "www", "full_name": "stormkit/www"},
	}, map[string]string{"Link": `<https://git.example.org/api/v1/user/repos?page=3>; rel="next",<https://git.example.org/api/v1/user/repos?page=1>; rel="first"`})

	repos, hasNextPage, err := s.client.ListRepos(ListReposParams{Page: 2, PerPage: 2})

	s.NoError(err)
	s.True(hasNextPage)
	s.Len(repos, 2)
	s.Equal("stormkit/app", repos[0].FullName)
	s.Equal("2", s.requests[0].URL.Query().Get("page"))
	s.Equal("2", s.requests[0].URL.Query().Get("limit"))
}

func (s *ForgejoSuite) Test_ListRepos_LastPage() {
	s.reply("GET /api/v1/user/repos", http.StatusOK, []map[string]any{
		{"name": "app", "full_name": "stormkit/app"},
	}, map[string]string{"Link": `<https://git.example.org/api/v1/user/repos?page=1>; rel="first"`})

	_, hasNextPage, err := s.client.ListRepos(ListReposParams{Page: 2, PerPage: 2})

	s.NoError(err)
	s.False(hasNextPage)
}

// Test_ListRepos_CappedPageSize verifies that an instance returning fewer
// repositories than requested per page still pages through all of them.
func (s *ForgejoSuite) Test_ListRepos_CappedPageSize() {
	s.reply("GET /api/v1/user/repos", http.StatusOK, []map[string]any{
		{"name": "app", "full_name": "stormkit/app"},
	}, map[string]string{"Link": `<https://git.example.org/api/v1/user/repos?page=3>; rel="next"`})

	_, hasNextPage, err := s.client.ListRepos(ListReposParams{Page: 2, PerPage: 50})

	s.NoError(err)
	s.True(hasNextPage)
}

func (s *ForgejoSuite) Test_InstallDeployKey_Creates() {
	s.reply("GET /api/v1/repos/stormkit/app/keys", http.StatusOK, []map[string]any{
		{"key": "ssh-rsa OTHER"},
	})

	s.reply("POST /api/v1/repos/stormkit/app/keys", http.StatusCreated, map[string]any{"id": 1})

	err := s.client.InstallDeployKey(InstallDeployKeyParams{
		Repo:      "forgejo/stormkit/app",
		AppID:     types.ID(15),
		PublicKey: "ssh-rsa AAAA\n",
	})

	s.NoError(err)
	s.Equal(map[string]any{
		"title":     "Stormkit Deploy Key - 15",
		"key":       "ssh-rsa AAAA",
		"read_only": true,
	}, s.bodies[1])
}

// Test_InstallDeployKey_AlreadyInstalled verifies that a key Forgejo stored
// with a trailing comment counts as installed.
func (s *ForgejoSuite) Test_InstallDeployKey_AlreadyInstalled() {
	s.reply("GET /api/v1/repos/stormkit/app/keys", http.StatusOK, []map[string]any{
		{"key": "ssh-rsa AAAA Stormkit Deploy Key - 15"},
	})

	err := s.client.InstallDeployKey(InstallDeployKeyParams{
		Repo:      "forgejo/stormkit/app",
		AppID:     types.ID(15),
		PublicKey: "ssh-rsa AAAA\n",
	})

	s.NoError(err)
	s.Len(s.requests, 1)
}

func (s *ForgejoSuite) Test_DefaultBranch_And_RepoSize() {
	s.reply("GET /api/v1/repos/stormkit/app", http.StatusOK, map[string]any{
		"name":           "app",
		"full_name":      "stormkit/app",
		"default_branch": "trunk",
		"size":           2,
	})

	branch, err := s.client.DefaultBranch("forgejo/stormkit/app")
	s.NoError(err)
	s.Equal("trunk", branch)

	size, err := s.client.RepoSize("forgejo/stormkit/app")
	s.NoError(err)
	s.Equal(int64(2048), size)
}

func (s *ForgejoSuite) Test_Repository_NotFound() {
	s.reply("GET /api/v1/repos/stormkit/missing", http.StatusNotFound, map[string]any{})

	_, err := s.client.Repository("forgejo/stormkit/missing")

	s.ErrorIs(err, oauth.ErrRepoNotFound)
}

func (s *ForgejoSuite) Test_CreatePullRequestComment() {
	s.reply("POST /api/v1/repos/stormkit/app/issues/7/comments", http.StatusCreated, map[string]any{"id": 1})

	err := s.client.CreatePullRequestComment(CreatePullRequestCommentParams{
		Repo:   "forgejo/stormkit/app",
		Number: 7,
		Body:   "Deployment completed",
	})

	s.NoError(err)
	s.Equal("Deployment completed", s.bodies[0]["body"])
}

func (s *ForgejoSuite) Test_InstallWebhooks_Creates() {
	s.reply("GET /api/v1/repos/stormkit/app/hooks", http.StatusOK, []map[string]any{
		{"id": 1, "config": map[string]string{"url": "https://ci.example.org/hook"}},
	})

	s.reply("POST /api/v1/repos/stormkit/app/hooks", http.StatusCreated, map[string]any{"id": 2})

	created, err := s.client.InstallWebhooks(InstallWebhooksParams{Repo: "forgejo/stormkit/app", AppID: types.ID(15)})

	s.NoError(err)
	s.True(created)

	body := s.bodies[1]
	config := body["config"].(map[string]any)

	s.Equal("gitea", body["type"])
	s.Equal([]any{"push", "pull_request"}, body["events"])
	secret, found := strings.CutPrefix(config["url"].(string), "https://api.example.org/app/webhooks/forgejo/")
	s.True(found)

	appID, err := utils.DecryptID(secret)
	s.NoError(err)
	s.Equal(types.ID(15), appID)
	s.Equal("json", config["content_type"])
}

// Test_InstallWebhooks_AlreadyInstalled verifies that an existing hook
// carrying the app secret is not added twice.
func (s *ForgejoSuite) Test_InstallWebhooks_AlreadyInstalled() {
	s.reply("GET /api/v1/repos/stormkit/app/hooks", http.StatusOK, []map[string]any{
		{"id": 1, "config": map[string]string{"url": "https://api.example.org/app/webhooks/forgejo/" + utils.EncryptID(types.ID(15))}},
	})

	created, err := s.client.InstallWebhooks(InstallWebhooksParams{Repo: "forgejo/stormkit/app", AppID: types.ID(15)})

	s.NoError(err)
	s.False(created)
	s.Len(s.requests, 1)
}

func (s *ForgejoSuite) Test_InstallWebhooks_Unauthorized() {
	s.reply("GET /api/v1/repos/stormkit/app/hooks", http.StatusForbidden, map[string]any{})

	_, err := s.client.InstallWebhooks(InstallWebhooksParams{Repo: "forgejo/stormkit/app", AppID: types.ID(15)})

	s.ErrorIs(err, oauth.ErrNotAuthorized)
}

func TestForgejoSuite(t *testing.T) {
	suite.Run(t, new(ForgejoSuite))
}
