package snippetshandlers_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/buildconf/snippetshandlers"
	"github.com/stormkit-io/stormkit-io/src/ce/api/user/usertest"
	"github.com/stretchr/testify/suite"

	"github.com/stormkit-io/stormkit-io/src/lib/database/databasetest"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp/shttptest"
)

type HandlerSnippetsGetSuite struct {
	suite.Suite
	*factory.Factory

	conn databasetest.TestDB
}

func (s *HandlerSnippetsGetSuite) BeforeTest(suiteName, _ string) {
	s.conn = databasetest.InitTx(suiteName)
	s.Factory = factory.New(s.conn)
	admin.SetMockLicense()
}

func (s *HandlerSnippetsGetSuite) AfterTest(_, _ string) {
	s.conn.CloseTx()
	admin.ResetMockLicense()
}

func (s *HandlerSnippetsGetSuite) Test_Success_WithFiltering() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	snippets := []*buildconf.Snippet{
		{Title: "Snippet 1", Content: "<script>console.log('snippet 1')</script>", Enabled: false, Prepend: false, Location: "body", AppID: app.ID, EnvID: env.ID, Rules: &buildconf.SnippetRule{Hosts: []string{"example.org", "example.com"}}},
		{Title: "Snippet 2", Content: "<script>console.log('snippet 2')</script>", Enabled: true, Prepend: true, Location: "body", AppID: app.ID, EnvID: env.ID},
		{Title: "Snippet 3", Content: "<script>console.log('snippet 3')</script>", Enabled: false, Prepend: false, Location: "head", AppID: app.ID, EnvID: env.ID},
		{Title: "Snippet 4", Content: "<script>console.log('snippet 4')</script>", Enabled: true, Prepend: true, Location: "head", AppID: app.ID, EnvID: env.ID, Rules: &buildconf.SnippetRule{Hosts: []string{"example.org", "example.com"}}},
	}

	s.NoError(buildconf.SnippetsStore().Insert(context.Background(), snippets))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/snippets?appId=%s&envId=%s&hosts=example.org,example.com&title=Snippet+4", app.ID.String(), env.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	jsonVal := fmt.Sprintf(`{
		"snippets": [
		  {
			"content": "<script>console.log('snippet 4')</script>",
			"enabled": true,
			"id": "%d",
			"location": "head",
			"prepend": true,
			"interpolate": false,
			"rules": {
				"hosts": ["example.org", "example.com"],
				"path": ""
			},
			"title": "Snippet 4"
		  }
		],
		"pagination": {
			"hasNextPage": false
		}
	  }`,
		snippets[3].ID,
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(jsonVal, response.String())
}

func (s *HandlerSnippetsGetSuite) Test_Success_WithoutFiltering() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	snippets := []*buildconf.Snippet{
		{Title: "Snippet 1", Content: "<script>console.log('snippet 1')</script>", Enabled: true, Prepend: true, Location: "body", AppID: app.ID, EnvID: env.ID},
		{Title: "Snippet 2", Content: "<script>console.log('snippet 2')</script>", Enabled: false, Prepend: false, Location: "head", AppID: app.ID, EnvID: env.ID, Rules: &buildconf.SnippetRule{Hosts: []string{"example.org"}}},
	}

	s.NoError(buildconf.SnippetsStore().Insert(context.Background(), snippets))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/snippets?appId=%s&envId=%s", app.ID.String(), env.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	jsonVal := fmt.Sprintf(`{
		"snippets": [
		  {
			"content": "<script>console.log('snippet 1')</script>",
			"enabled": true,
			"id": "%d",
			"location": "body",
			"prepend": true,
			"interpolate": false,
			"rules": null,
			"title": "Snippet 1"
		  },
		  {
			"content": "<script>console.log('snippet 2')</script>",
			"enabled": false,
			"id": "%d",
			"location": "head",
			"prepend": false,
			"interpolate": false,
			"rules": {
				"hosts": ["example.org"],
				"path": ""
			},
			"title": "Snippet 2"
		  }
		],
		"pagination": {
			"hasNextPage": false
		}
	  }`,
		snippets[0].ID,
		snippets[1].ID,
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(jsonVal, response.String())
}

func (s *HandlerSnippetsGetSuite) Test_Success_Pagination() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	snippetshandlers.DefaultSnippetsLimit = 1

	defer func() {
		snippetshandlers.DefaultSnippetsLimit = 50
	}()

	snippets := []*buildconf.Snippet{
		{Title: "Snippet 1", Content: "<script>console.log('snippet 1')</script>", Enabled: true, Prepend: true, Location: "body", AppID: app.ID, EnvID: env.ID},
		{Title: "Snippet 2", Content: "<script>console.log('snippet 2')</script>", Enabled: false, Prepend: false, Location: "head", AppID: app.ID, EnvID: env.ID, Rules: &buildconf.SnippetRule{Hosts: []string{"example.org"}}},
	}

	s.NoError(buildconf.SnippetsStore().Insert(context.Background(), snippets))

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/snippets?appId=%s&envId=%s", app.ID.String(), env.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	jsonVal := fmt.Sprintf(`{
		"snippets": [
		  {
			"content": "<script>console.log('snippet 1')</script>",
			"enabled": true,
			"id": "%d",
			"location": "body",
			"prepend": true,
			"interpolate": false,
			"rules": null,
			"title": "Snippet 1"
		  }
		],
		"pagination": {
			"hasNextPage": true,
			"afterId": "1"
		}
	  }`,
		snippets[0].ID,
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(jsonVal, response.String())

	response = shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/snippets?appId=%s&envId=%s&afterId=1", app.ID.String(), env.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	jsonVal = fmt.Sprintf(`{
		"snippets": [
		  {
			"content": "<script>console.log('snippet 2')</script>",
			"enabled": false,
			"id": "%d",
			"location": "head",
			"prepend": false,
			"interpolate": false,
			"rules": {
			  "hosts": ["example.org"],
			  "path": ""
			},
			"title": "Snippet 2"
		  }
		],
		"pagination": {
			"hasNextPage": false
		}
	  }`,
		snippets[1].ID,
	)

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(jsonVal, response.String())
}

func (s *HandlerSnippetsGetSuite) Test_Success_NoSnippets() {
	usr := s.MockUser()
	app := s.MockApp(usr)
	env := s.MockEnv(app)

	response := shttptest.RequestWithHeaders(
		shttp.NewRouter().RegisterService(snippetshandlers.Services).Router().Handler(),
		shttp.MethodGet,
		fmt.Sprintf("/snippets?appId=%s&envId=%s", app.ID.String(), env.ID.String()),
		nil,
		map[string]string{
			"Authorization": usertest.Authorization(usr.ID),
		},
	)

	jsonVal := `{
		"snippets": [],
		"pagination": {
			"hasNextPage": false
		}
	}`

	s.Equal(http.StatusOK, response.Code)
	s.JSONEq(jsonVal, response.String())
}

func TestHandlerSnippetsGet(t *testing.T) {
	suite.Run(t, &HandlerSnippetsGetSuite{})
}
