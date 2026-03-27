package snippetshandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/appcache"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/snippetshandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stormkit-io/stormkit-io/src/ee/api/audit"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
	"gopkg.in/guregu/null.v3"

	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

type HandlerSnippetsAddSuite struct {
	suite.Suite
	*factory.Factory

	conn             databasetest.TestDB
	mockCacheService *mocks.CacheInterface
}

func (s *HandlerSnippetsAddSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	s.mockCacheService = &mocks.CacheInterface{}
	appcache.DefaultCacheService = s.mockCacheService
	admin.SetMockLicense()
}

func (s *HandlerSnippetsAddSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	appcache.DefaultCacheService = nil
	admin.ResetMockLicense()
}

func (s *HandlerSnippetsAddSuite) Test_Success() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	ctx := context.Background()

	snippets := []*buildconf.Snippet{
		{Title: "Snippet 1", Content: "<script>console.log('snippet 1')</script>", Enabled: false, Prepend: false, Location: "body", AppID: app.ID, EnvID: env.ID},
		{Title: "Snippet 2", Content: "<script>console.log('snippet 2')</script>", Enabled: true, Prepend: true, Location: "body", AppID: app.ID, EnvID: env.ID},
		{Title: "Snippet 3", Content: "<script>console.log('snippet 3')</script>", Enabled: false, Prepend: false, Location: "head", AppID: app.ID, EnvID: env.ID},
		{Title: "Snippet 4", Content: "<script>console.log('snippet 4')</script>", Enabled: true, Prepend: true, Location: "head", AppID: app.ID, EnvID: env.ID},
	}

	s.NoError(buildconf.SnippetsStore().Insert(ctx, snippets))
	s.NoError(buildconf.DomainStore().Insert(ctx, &buildconf.DomainModel{
		EnvID:      env.ID,
		AppID:      app.ID,
		Name:       "www.stormkit.io",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
		Token:      null.StringFrom(utils.RandomToken(32)),
	}))

	s.mockCacheService.On("Reset", env.ID).Return(nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/snippets",
		map[string]any{
			"snippets": []map[string]any{
				{
					"title":    "Added Snippet 1",
					"content":  "Hello World 1",
					"enabled":  true,
					"prepend":  false,
					"location": "head",
				},
				{
					"title":    "Added Snippet 2",
					"content":  "Hello World 2",
					"enabled":  false,
					"prepend":  true,
					"location": "body",
					"rules": map[string]any{
						"hosts": []string{"www.stormkit.io", "sample-project.stormkit:8888", "app.stormkit:8888"},
						"path":  "^/my-path/.*/end",
					}},
			},
			"appId": app.ID.String(),
			"envId": env.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	allSnippets, err := buildconf.SnippetsStore().SnippetsByEnvID(ctx, buildconf.SnippetFilters{
		EnvID: env.ID,
	})

	s.NoError(err)
	s.Equal(http.StatusCreated, response.Code)
	s.Len(allSnippets, 6)

	// Snippet 1
	s.Equal("Added Snippet 1", allSnippets[4].Title)
	s.Equal("Hello World 1", allSnippets[4].Content)
	s.Equal(types.ID(5), allSnippets[4].ID)
	s.True(allSnippets[4].Enabled)
	s.False(allSnippets[4].Prepend)

	// Snippet 2
	s.Equal("Added Snippet 2", allSnippets[5].Title)
	s.Equal("Hello World 2", allSnippets[5].Content)
	s.Equal(types.ID(6), allSnippets[5].ID)
	s.False(allSnippets[5].Enabled)
	s.True(allSnippets[5].Prepend)

	expected := `{
		"snippets": [
			{
				"id": "5",
				"enabled": true,
				"prepend": false,
				"content": "Hello World 1",
				"title": "Added Snippet 1",
				"location": "head",
				"rules": null
			},
			{
				"id": "6",
				"enabled": false,
				"prepend": true,
				"content": "Hello World 2",
				"title": "Added Snippet 2",
				"location": "body",
				"rules": { 
					"hosts": ["www.stormkit.io", "*.dev"],
					"path": "^/my-path/.*/end"
				}
			}
		]
	}`

	s.JSONEq(expected, response.String())

	audits, err := audit.NewStore().SelectAudits(context.Background(), audit.AuditFilters{
		EnvID: env.ID,
	})

	s.NoError(err)
	s.Len(audits, 1)
	s.Equal(audit.Audit{
		ID:          audits[0].ID,
		Timestamp:   audits[0].Timestamp,
		Action:      "CREATE:SNIPPET",
		EnvName:     env.Name,
		AppID:       app.ID,
		EnvID:       env.ID,
		TeamID:      app.TeamID,
		UserID:      usr.ID,
		UserDisplay: usr.Display(),
		Diff: &audit.Diff{
			New: audit.DiffFields{
				Snippets: []string{"Added Snippet 1", "Added Snippet 2"},
			},
		},
	}, audits[0])
}

func (s *HandlerSnippetsAddSuite) Test_Success_ResetOnlyRelatedDomains() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)
	ctx := context.Background()

	s.NoError(buildconf.DomainStore().Insert(ctx, &buildconf.DomainModel{
		EnvID:      env.ID,
		AppID:      app.ID,
		Name:       "www.stormkit.io",
		Verified:   true,
		VerifiedAt: utils.NewUnix(),
		Token:      null.StringFrom(utils.RandomToken(32)),
	}))

	s.mockCacheService.On("Reset", env.ID, "www.stormkit.io").Return(nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/snippets",
		map[string]any{
			"snippets": []map[string]any{
				{
					"title":    "Added Snippet 2",
					"content":  "Hello World 2",
					"enabled":  false,
					"prepend":  true,
					"location": "body",
					"rules": map[string]any{
						"hosts": []string{"www.stormkit.io"},
						"path":  "^/my-path/.*/end",
					}},
			},
			"appId": app.ID.String(),
			"envId": env.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusCreated, response.Code)
}

func (s *HandlerSnippetsAddSuite) Test_Success_ResetAll() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	s.mockCacheService.On("Reset", env.ID, fmt.Sprintf("^%s(?:--\\d+)?", app.DisplayName)).Return(nil)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/snippets",
		map[string]any{
			"snippets": []map[string]any{
				{
					"title":    "Added Snippet 2",
					"content":  "Hello World 2",
					"enabled":  false,
					"prepend":  true,
					"location": "body",
					"rules": map[string]any{
						"hosts": []string{"*.dev"},
						"path":  "^/my-path/.*/end",
					}},
			},
			"appId": app.ID.String(),
			"envId": env.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusCreated, response.Code)
}

func (s *HandlerSnippetsAddSuite) Test_Fail_Duplicate() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	snippets := []*buildconf.Snippet{
		{Title: "Snippet 1", Content: "Hello world", Enabled: false, Prepend: false, Location: "body", AppID: app.ID, EnvID: env.ID},
	}

	s.NoError(buildconf.SnippetsStore().Insert(context.Background(), snippets))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/snippets",
		map[string]any{
			"snippets": []map[string]any{
				{
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

func (s *HandlerSnippetsAddSuite) Test_Fail_InvalidHost() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/snippets",
		map[string]any{
			"snippets": []map[string]any{
				{
					"title":    "Added Snippet 1",
					"content":  "Hello World 1",
					"enabled":  true,
					"prepend":  false,
					"location": "head",
				},
				{
					"title":    "Added Snippet 2",
					"content":  "Hello World 2",
					"enabled":  false,
					"prepend":  true,
					"location": "body",
					"rules": map[string]any{
						"hosts": []string{"www.stormkit.io", "sample-project.stormkit:8888", "app.stormkit:8888"},
						"path":  "^/my-path/.*/end",
					}},
			},
			"appId": app.ID.String(),
			"envId": env.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(`{"errors": ["Invalid or missing domain name(s): www.stormkit.io"]}`, response.String())
}

func (s *HandlerSnippetsAddSuite) TestInvalidRequest_NoSnippets() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/snippets",
		map[string]string{
			"appId": app.ID.String(),
			"envId": env.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(response.String(), `{"errors": ["Nothing to add."]}`)
}

func (s *HandlerSnippetsAddSuite) TestInvalidRequest_InvalidLocation() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/snippets",
		map[string]any{
			"snippets": []map[string]any{
				{"title": "Added Snippet 1", "content": "Hello World 1", "enabled": true, "prepend": false, "location": "body"},
				{"title": "Added Snippet 2", "content": "Hello World 2", "enabled": false, "prepend": true, "location": "invalid"},
			},
			"appId": app.ID.String(),
			"envId": env.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(response.String(), `{"errors": ["Location must be either 'head' or 'body'."]}`)
}

func (s *HandlerSnippetsAddSuite) TestInvalidRequest_InvalidTitle() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/snippets",
		map[string]any{
			"snippets": []map[string]any{
				{"title": "", "content": "Hello World 1", "enabled": true, "prepend": false, "location": "body"},
			},
			"appId": app.ID.String(),
			"envId": env.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(response.String(), `{"errors": ["Snippet title is a required field."]}`)
}

func (s *HandlerSnippetsAddSuite) TestInvalidRequest_InvalidContent() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/snippets",
		map[string]any{
			"snippets": []map[string]any{
				{"title": "Title", "content": "", "enabled": true, "prepend": false, "location": "body"},
			},
			"appId": app.ID.String(),
			"envId": env.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(response.String(), `{"errors": ["Snippet content is a required field."]}`)
}

func (s *HandlerSnippetsAddSuite) TestInvalidRequest_InvalidPathRegexp() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodPost,
		"/snippets",
		map[string]any{
			"snippets": []map[string]any{
				{"title": "valid", "content": "Hello World 1", "enabled": true, "prepend": false, "location": "body", "rules": map[string]string{
					"path": "[invalid-regexp",
				}},
			},
			"appId": app.ID.String(),
			"envId": env.ID.String(),
		},
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	s.Equal(http.StatusBadRequest, response.Code)
	s.JSONEq(response.String(), `{"errors": ["Snippet path must be a valid regular expression."]}`)
}

func TestHandlerSnippetsAdd(t *testing.T) {
	suite.Run(t, &HandlerSnippetsAddSuite{})
}
