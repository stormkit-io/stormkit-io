package snippetshandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/snippetshandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"
)

type HandlerSnippetsPutSuite struct {
	suite.Suite
	*factory.Factory

	conn             databasetest.TestDB
	mockCacheService *mocks.CacheInterface
}

func (s *HandlerSnippetsPutSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockCacheService = &mocks.CacheInterface{}
	appcache.DefaultCacheService = s.mockCacheService
	admin.SetMockLicense()
}

func (s *HandlerSnippetsPutSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	appcache.DefaultCacheService = nil
	admin.ResetMockLicense()
}

func (s *HandlerSnippetsPutSuite) Test_Success_ChangeLocation() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	ctx := context.TODO()

	snippets := []*buildconf.Snippet{
		{Title: "Snippet 1", Content: "Hello World 1", Enabled: false, Prepend: false, Location: "body", AppID: app.ID, EnvID: env.ID},
		{Title: "Snippet 2", Content: "<script>console.log('snippet 2')</script>", Enabled: true, Prepend: true, Location: "body", AppID: app.ID, EnvID: env.ID},
		{Title: "Snippet 3", Content: "<script>console.log('snippet 3')</script>", Enabled: false, Prepend: false, Location: "head", AppID: app.ID, EnvID: env.ID},
		{Title: "Snippet 4", Content: "<script>console.log('snippet 4')</script>", Enabled: true, Prepend: true, Location: "head", AppID: app.ID, EnvID: env.ID},
	}

	s.NoError(buildconf.SnippetsStore().Insert(ctx, snippets))
	s.NoError(buildconf.DomainStore().Insert(ctx, &buildconf.DomainModel{
		EnvID:      env.ID,
		AppID:      app.ID,
		Name:       "www.example.org",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
		Token:      null.StringFrom(utils.RandomToken(32)),
	}))

	// Should update all because snippet hosts change from empty => "www.example.org" and "*.dev"
	s.mockCacheService.On("Reset", env.ID).Return(nil)

	now := time.Now().Add(-time.Duration(time.Minute * 10))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/snippets",
		map[string]any{
			"appId": app.ID.String(),
			"envId": env.ID.String(),
			"snippet": map[string]any{
				"title":    "Edited Snippet 1",
				"content":  "New content",
				"enabled":  snippets[0].Enabled,
				"prepend":  snippets[0].Prepend,
				"location": "head",
				"id":       snippets[0].ID.String(),
				"rules": map[string]any{
					"hosts": []string{"www.example.org", "example.stormkit:8888"},
				},
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	snippet, err := buildconf.SnippetsStore().SnippetByID(context.Background(), snippets[0].ID)

	s.NoError(err)
	s.Equal(http.StatusOK, response.Code)

	s.Equal("Edited Snippet 1", snippet.Title)
	s.Equal("New content", snippet.Content)
	s.Equal(snippets[0].ID, snippet.ID)
	s.Equal(snippets[0].Enabled, snippet.Enabled)
	s.Equal(snippets[0].Prepend, snippet.Prepend)
	s.Equal("head", snippet.Location)
	s.Equal([]string{"www.example.org", "*.dev"}, snippet.Rules.Hosts)

	// Should also update the environment's updated at property
	environment, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)
	s.NoError(err)
	s.True(environment.UpdatedAt.Time.After(now))

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.NoError(err)
	s.Len(audits, 1)
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "UPDATE:SNIPPET",
		EnvName:     env.Name,
		EnvID:       env.ID,
		AppID:       app.ID,
		TeamID:      app.TeamID,
		UserID:      usr.ID,
		UserDisplay: usr.Display(),
		Diff: &audit.Diff{
			Old: audit.DiffFields{
				SnippetTitle:    snippets[0].Title,
				SnippetContent:  snippets[0].Content,
				SnippetLocation: snippets[0].Location,
				SnippetRules:    snippets[0].Rules,
				SnippetPrepend:  audit.Bool(snippets[0].Prepend),
				SnippetEnabled:  audit.Bool(snippets[0].Enabled),
			},
			New: audit.DiffFields{
				SnippetTitle:    "Edited Snippet 1",
				SnippetContent:  "New content",
				SnippetLocation: "head",
				SnippetPrepend:  audit.Bool(snippets[0].Prepend),
				SnippetEnabled:  audit.Bool(snippets[0].Enabled),
				SnippetRules: &buildconf.SnippetRule{
					Hosts: []string{"www.example.org", "*.dev"},
				},
			},
		},
	}, audits[0])
}

func (s *HandlerSnippetsPutSuite) Test_Success_ResetOnlyRelatedDomains() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	ctx := context.TODO()

	snippets := []*buildconf.Snippet{
		{Title: "Snippet 1", Content: "Hello World 1", Enabled: false, Prepend: false, Location: "body", AppID: app.ID, EnvID: env.ID},
	}

	s.NoError(buildconf.SnippetsStore().Insert(ctx, snippets))
	s.NoError(buildconf.DomainStore().Insert(ctx, &buildconf.DomainModel{
		EnvID:      env.ID,
		AppID:      app.ID,
		Name:       "www.example.org",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
		Token:      null.StringFrom(utils.RandomToken(32)),
	}))

	expectations := []map[string]any{
		// The first call should reset all domains because we're switching from `no host` => `host`
		{"newHosts": []string{"www.Example.org"}, "args": []any{env.ID}},
		// The second call should reset only www.example.org
		{"newHosts": []string{"www.example.org"}, "args": []any{env.ID, "www.example.org"}},
		// The third call should reset all domains (because we're changing www.example.org => *.dev)
		{"newHosts": []string{"*.dev"}, "args": []any{env.ID, fmt.Sprintf("^%s(?:--\\d+)?", app.DisplayName), "www.example.org"}},
	}

	for i, exp := range expectations {
		s.mockCacheService.On("Reset", exp["args"].([]any)...).Return(nil).Once()

		response := shttptest.RequestWithHeaders(
			shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
			shttp.MethodPut,
			"/snippets",
			map[string]any{
				"appId": app.ID.String(),
				"envId": env.ID.String(),
				"snippet": map[string]any{
					"title":    fmt.Sprintf("Edited Snippet %d", i),
					"content":  snippets[0].Content,
					"enabled":  snippets[0].Enabled,
					"prepend":  snippets[0].Prepend,
					"location": "head",
					"id":       snippets[0].ID.String(),
					"rules": map[string]any{
						"hosts": exp["newHosts"].([]string),
					},
				},
			},
			map[string]string{
				"Authorization": usertest.Authorization(usr.ID),
			},
		)

		s.mockCacheService.AssertExpectations(s.T())
		s.Equal(http.StatusOK, response.Code)
	}
}

func (s *HandlerSnippetsPutSuite) Test_Fail_InvalidHost() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	ctx := context.TODO()

	snippets := []*buildconf.Snippet{
		{Title: "Snippet 1", Content: "Hello World 1", Enabled: false, Prepend: false, Location: "body", AppID: app.ID, EnvID: env.ID},
	}

	s.NoError(buildconf.SnippetsStore().Insert(ctx, snippets))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/snippets",
		map[string]any{
			"appId": app.ID.String(),
			"envId": env.ID.String(),
			"snippet": map[string]any{
				"title":    "Edited Snippet 1",
				"content":  snippets[0].Content,
				"enabled":  snippets[0].Enabled,
				"prepend":  snippets[0].Prepend,
				"location": "head",
				"id":       snippets[0].ID.String(),
				"rules": map[string]any{
					"hosts": []string{"www.example.org", "example.stormkit:8888"},
				},
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"error": "Invalid or missing domain name(s): www.example.org" }`, response.String())
}

func (s *HandlerSnippetsPutSuite) Test_Success_SavingSameObject() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	snippets := []*buildconf.Snippet{
		{Title: "Snippet 1", Content: "Hello World 1", Enabled: false, Prepend: false, Location: "body", AppID: app.ID, EnvID: env.ID},
	}

	s.NoError(buildconf.SnippetsStore().Insert(context.Background(), snippets))

	s.mockCacheService.On("Reset", env.ID).Return(nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/snippets",
		map[string]any{
			"appId": app.ID.String(),
			"envId": env.ID.String(),
			"snippet": map[string]any{
				"title":    snippets[0].Title,
				"content":  snippets[0].Content,
				"enabled":  snippets[0].Enabled,
				"prepend":  snippets[0].Prepend,
				"location": snippets[0].Location,
				"id":       snippets[0].ID.String(),
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusOK, response.Code)

	// Should not generate any log when the content does not change
	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.NoError(err)
	s.Len(audits, 0)
}

func (s *HandlerSnippetsPutSuite) Test_InvalidRequest_NoSnippet() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/snippets",
		map[string]any{
			"appId": app.ID.String(),
			"envId": env.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(response.String(), `{"error": "Nothing to update."}`)

	// Should not update the environment's updated at property
	environment, err := buildconf.NewStore().EnvironmentByID(context.Background(), env.ID)
	s.NoError(err)
	s.False(environment.UpdatedAt.Valid)
}

func (s *HandlerSnippetsPutSuite) Test_InvalidRequest_NoSnippetID() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/snippets",
		map[string]any{
			"appId": app.ID.String(),
			"envId": env.ID.String(),
			"snippet": map[string]any{
				"title":    "Added Snippet 2",
				"content":  "Hello World 2",
				"enabled":  false,
				"prepend":  true,
				"location": "invalid",
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(response.String(), `{"error": "Nothing to update."}`)
}

func (s *HandlerSnippetsPutSuite) Test_InvalidRequest_InvalidLocation() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/snippets",
		map[string]any{
			"appId": app.ID.String(),
			"envId": env.ID.String(),
			"snippet": map[string]any{
				"title":    "Added Snippet 2",
				"content":  "Hello World 2",
				"enabled":  false,
				"prepend":  true,
				"location": "invalid",
				"id":       "4",
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(response.String(), `{"error": "Location must be either 'head' or 'body'."}`)
}

func (s *HandlerSnippetsPutSuite) Test_InvalidRequest_InvalidTitle() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/snippets",
		map[string]any{
			"appId": app.ID.String(),
			"envId": env.ID.String(),
			"snippet": map[string]any{
				"title":    "",
				"content":  "Hello World 2",
				"enabled":  false,
				"prepend":  true,
				"location": "body",
				"id":       "4",
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(response.String(), `{"error": "Snippet title is a required field."}`)
}

func (s *HandlerSnippetsPutSuite) Test_Fail_Duplicate() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	snippets := []*buildconf.Snippet{
		{Title: "Snippet 1", Content: "Hello world", Enabled: false, Prepend: false, Location: "body", AppID: app.ID, EnvID: env.ID},
		{Title: "Snippet 2", Content: "Hello world 2", Enabled: false, Prepend: false, Location: "body", AppID: app.ID, EnvID: env.ID},
	}

	s.NoError(buildconf.SnippetsStore().Insert(context.Background(), snippets))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/snippets",
		map[string]any{
			"snippets": []map[string]any{
				{
					"id":       snippets[1].ID.String(),
					"title":    "Snippet 2",
					"content":  "Hello world",
					"location": "body",
					"rules":    nil,
				},
			},
			"appId": app.ID.String(),
			"envId": env.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusConflict, response.Code)
}

func (s *HandlerSnippetsPutSuite) Test_InvalidRequest_InvalidContent() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPut,
		"/snippets",
		map[string]any{
			"appId": app.ID.String(),
			"envId": env.ID.String(),
			"snippet": map[string]any{
				"title":    "Title",
				"content":  "",
				"enabled":  false,
				"prepend":  true,
				"location": "body",
				"id":       "4",
			},
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(response.String(), `{"error": "Snippet content is a required field."}`)
}

func TestHandlerSnippetsPut(t *testing.T) {
	suite.Run(t, &HandlerSnippetsPutSuite{})
}
