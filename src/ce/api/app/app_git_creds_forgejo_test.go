package app_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth"
	"github.com/stormkit-io/stormkit-io/src/ce/api/oauth/forgejo"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stretchr/testify/suite"
	"golang.org/x/oauth2"
)

type GitCredsForgejoSuite struct {
	suite.Suite
	*factory.Factory

	conn       databasetest.TestDB
	server     *httptest.Server
	sshURL     string
	deployKeys []string
}

func (s *GitCredsForgejoSuite) SetupSuite() {
	utils.SetKeySize(1024)
}

func (s *GitCredsForgejoSuite) TearDownSuite() {
	utils.RestoreKeySize()
}

func (s *GitCredsForgejoSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.sshURL = "ssh://git@git.example.org:2222/stormkit/app.git"
	s.deployKeys = nil

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/repos/stormkit/app", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"full_name": "stormkit/app", "ssh_url": s.sshURL})
	})

	mux.HandleFunc("GET /api/v1/repos/stormkit/app/hooks", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{})
	})

	mux.HandleFunc("POST /api/v1/repos/stormkit/app/hooks", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	mux.HandleFunc("GET /api/v1/repos/stormkit/app/keys", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]any{})
	})

	mux.HandleFunc("POST /api/v1/repos/stormkit/app/keys", func(w http.ResponseWriter, r *http.Request) {
		body := map[string]any{}
		_ = json.NewDecoder(r.Body).Decode(&body)
		s.deployKeys = append(s.deployKeys, body["key"].(string))
		w.WriteHeader(http.StatusCreated)
	})

	s.server = httptest.NewServer(mux)

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

func (s *GitCredsForgejoSuite) AfterTest(_, _ string) {
	s.server.Close()
	s.conn.CloseTx()
	admin.ResetCache(context.Background())
}

// forgejoApp creates an app whose owner has connected Forgejo.
func (s *GitCredsForgejoSuite) forgejoApp() *factory.MockApp {
	usr := s.MockUser(nil)
	appl := s.MockApp(usr, map[string]any{"Repo": "forgejo/stormkit/app"})

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
	return appl
}

// Test_DeployKey verifies that builds receive the app's deploy key and the
// SSH address, never the user's token.
func (s *GitCredsForgejoSuite) Test_DeployKey() {
	appl := s.forgejoApp()

	creds, err := appl.GitCreds(context.Background())

	s.NoError(err)
	s.Equal(s.sshURL, creds.CloneURL)
	s.NotContains(creds.Token, "forgejo-token")

	decoded, err := utils.DecodeString(creds.Token)
	s.Require().NoError(err)

	pieces := strings.Split(string(decoded), "|")
	pkey := appl.PrivateKey(context.Background())

	s.Equal("git", pieces[0])
	s.Equal(pkey.SSHPubKey(), pieces[1])
	s.Equal(pkey.SSHPrivKey(), pieces[2])
	s.Equal([]string{strings.TrimSpace(pkey.SSHPubKey())}, s.deployKeys)
}

func (s *GitCredsForgejoSuite) Test_SSHDisabled() {
	s.sshURL = ""

	_, err := s.forgejoApp().GitCreds(context.Background())

	s.ErrorIs(err, app.ErrForgejoSSHDisabled)
	s.Empty(s.deployKeys)
}

func TestGitCredsForgejoSuite(t *testing.T) {
	suite.Run(t, new(GitCredsForgejoSuite))
}
