package hosting

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stretchr/testify/suite"
)

type PageDataRenderSuite struct {
	suite.Suite
	renderer *pageData
}

func (s *PageDataRenderSuite) SetupTest() {
	s.renderer = &pageData{loader: &redirects.Loader{URL: "https://api.example.com/v/abc.json"}}
}

func (s *PageDataRenderSuite) document() map[string]any {
	var document map[string]any

	s.Require().NoError(json.Unmarshal([]byte(`{
		"title": "My video",
		"views": 12,
		"ratio": 1.5,
		"public": false,
		"thumbnail": "",
		"owner": { "name": "Alice" },
		"tags": ["a", "b"]
	}`), &document))

	return document
}

// render executes body against document with a 200 status and fails the test on
// a template error.
func (s *PageDataRenderSuite) render(body string, document map[string]any) string {
	out, err := s.renderer.render(body, document, http.StatusOK)
	s.Require().NoError(err)

	return string(out)
}

func (s *PageDataRenderSuite) Test_Render_Value() {
	s.Equal("<title>My video</title>", s.render(`<title>{{.title}}</title>`, s.document()))
	s.Equal("<title>My video</title>", s.render(`<title>{{ .title }}</title>`, s.document()))
}

func (s *PageDataRenderSuite) Test_Render_NestedKey() {
	s.Equal("<span>Alice</span>", s.render(`<span>{{.owner.name}}</span>`, s.document()))
}

func (s *PageDataRenderSuite) Test_Render_Numbers() {
	s.Equal("12|1.5", s.render(`{{.views}}|{{.ratio}}`, s.document()))
}

func (s *PageDataRenderSuite) Test_Render_MissingValueIsEmpty() {
	s.Equal(`<meta content="">`, s.render(`<meta content="{{.robots}}">`, s.document()))
}

// A default fires on absent, null and empty — never on false or zero, which are
// answers the upstream gave.
func (s *PageDataRenderSuite) Test_Render_Default_OnMissingNullAndEmpty() {
	document := s.document()
	document["subtitle"] = nil

	s.Equal("noindex", s.render(`{{.robots | default "noindex"}}`, document))
	s.Equal("none", s.render(`{{.subtitle | default "none"}}`, document))
	s.Equal("/og.png", s.render(`{{.thumbnail | default "/og.png"}}`, document))
}

func (s *PageDataRenderSuite) Test_Render_Default_NotAppliedToFalseOrZero() {
	document := s.document()
	document["views"] = float64(0)

	s.Equal("false", s.render(`{{.public | default true}}`, document))
	s.Equal("0", s.render(`{{.views | default 42}}`, document))
}

// A value lands in attribute position as often as in text, so it is escaped
// for whichever context it is in.
func (s *PageDataRenderSuite) Test_Render_EscapesUpstreamValues() {
	document := s.document()
	document["title"] = `" onload="alert(1)`
	document["link"] = "javascript:alert(1)"

	out := s.render(`<video id="{{.title}}"><a href="{{.link}}">`, document)

	s.NotContains(out, `onload="alert`)
	s.NotContains(out, "javascript:")
}

func (s *PageDataRenderSuite) Test_Render_WholeDocumentInScript() {
	document := map[string]any{"title": "</script>"}
	out := s.render(`<script id="__sk_data__" type="application/json">{{.}}</script>`, document)

	s.Equal(`<script id="__sk_data__" type="application/json">{"title":"\u003c/script\u003e"}</script>`, out)
}

func (s *PageDataRenderSuite) Test_Render_Conditionals() {
	body := `{{if eq status 410}}Unavailable{{else if .public}}Public{{else}}Private{{end}}`

	out, err := s.renderer.render(body, map[string]any{}, http.StatusGone)
	s.Require().NoError(err)
	s.Equal("Unavailable", string(out))

	s.Equal("Private", s.render(body, s.document()))
}

func (s *PageDataRenderSuite) Test_Render_Range() {
	s.Equal("<li>a</li><li>b</li>", s.render(`{{range .tags}}<li>{{.}}</li>{{end}}`, s.document()))
}

// A nested field on a missing object is an execution error; with guards it.
func (s *PageDataRenderSuite) Test_Render_WithGuardsMissingObjects() {
	s.Equal("", s.render(`{{with .author}}{{.name}}{{end}}`, s.document()))

	_, err := s.renderer.render(`{{.author.name}}`, s.document(), http.StatusOK)
	s.Error(err)
}

// When the API fails the document is empty, and {{if .}} keeps the page from
// reading nested fields that are not there.
func (s *PageDataRenderSuite) Test_Render_GuardsAPIFailure() {
	body := `{{if .}}{{.owner.name}}{{else if eq status 410}}Unavailable{{else}}Something went wrong{{end}}`

	out, err := s.renderer.render(body, map[string]any{}, 0)
	s.Require().NoError(err)
	s.Equal("Something went wrong", string(out))

	out, err = s.renderer.render(body, map[string]any{}, http.StatusGone)
	s.Require().NoError(err)
	s.Equal("Unavailable", string(out))

	s.Equal("Alice", s.render(body, s.document()))
}

// Documented limitations: comments are dropped, and a literal {{ is escaped.
func (s *PageDataRenderSuite) Test_Render_Limitations() {
	s.Equal("<p>hi</p>", s.render(`<!-- note --><p>hi</p>`, s.document()))
	s.Equal("<p>{{ name }}</p>", s.render(`<p>{{"{{"}} name }}</p>`, s.document()))
}

func (s *PageDataRenderSuite) Test_Render_ParseError() {
	_, err := s.renderer.render(`<title>{{.title</title>`, s.document(), http.StatusOK)
	s.Error(err)
}

func TestPageDataRender(t *testing.T) {
	suite.Run(t, new(PageDataRenderSuite))
}

type PageDataFetchSuite struct {
	suite.Suite
}

func (s *PageDataFetchSuite) SetupTest() {
	s.T().Setenv("STORMKIT_PAGE_DATA_INSECURE", "true")
}

func (s *PageDataFetchSuite) Test_Fetch() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.Equal("application/json", r.Header.Get("Accept"))
		w.Write([]byte(`{"title":"My video"}`))
	}))

	defer server.Close()

	result := (&pageData{loader: &redirects.Loader{URL: server.URL}}).fetch(context.Background())

	s.NoError(result.err)
	s.Equal(http.StatusOK, result.status)
	s.Equal("My video", result.document["title"])
}

func (s *PageDataFetchSuite) Test_Fetch_CarriesUpstreamStatus() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`not found`))
	}))

	defer server.Close()

	result := (&pageData{loader: &redirects.Loader{URL: server.URL}}).fetch(context.Background())

	// A non-2xx body is not the document, so failing to parse it is not an error.
	s.NoError(result.err)
	s.Equal(http.StatusNotFound, result.status)
}

// Nothing is kept between requests: caching belongs to the API.
func (s *PageDataFetchSuite) Test_Fetch_AlwaysFetches() {
	calls := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls++
		w.Write([]byte(`{"title":"My video"}`))
	}))

	defer server.Close()

	loader := &redirects.Loader{URL: server.URL}

	(&pageData{loader: loader}).fetch(context.Background())
	(&pageData{loader: loader}).fetch(context.Background())

	s.Equal(2, calls)
}

// Without the escape hatch a loader may not reach a private address, or it
// could be pointed at the instance's own metadata service.
func (s *PageDataFetchSuite) Test_Fetch_RefusesPrivateAddresses() {
	s.T().Setenv("STORMKIT_PAGE_DATA_INSECURE", "false")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"title":"My video"}`))
	}))

	defer server.Close()

	result := (&pageData{loader: &redirects.Loader{URL: server.URL}}).fetch(context.Background())

	s.Error(result.err)
	s.Nil(result.document)
}

func (s *PageDataFetchSuite) Test_Fetch_RefusesPlainHTTP() {
	s.T().Setenv("STORMKIT_PAGE_DATA_INSECURE", "false")

	result := (&pageData{loader: &redirects.Loader{URL: "http://api.example.com/v/abc.json"}}).fetch(context.Background())

	s.Error(result.err)
	s.Contains(result.err.Error(), "https")
}

// A burst of requests for one record reaches the upstream once.
func (s *PageDataFetchSuite) Test_Fetch_ConcurrentCallersShareOneRequest() {
	var calls atomic.Int32

	release := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		<-release
		w.Write([]byte(`{"title":"My video"}`))
	}))

	defer server.Close()

	loader := &redirects.Loader{URL: server.URL}
	results := make([]loaderResult, 10)
	wg := sync.WaitGroup{}

	for i := range results {
		wg.Add(1)

		go func() {
			defer wg.Done()
			results[i] = (&pageData{loader: loader}).fetch(context.Background())
		}()
	}

	s.Eventually(func() bool { return calls.Load() == 1 }, time.Second, 5*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	close(release)
	wg.Wait()

	s.Equal(int32(1), calls.Load())

	for _, result := range results {
		s.Equal("My video", result.document["title"])
	}
}

// Requests from different visitors each reach the API, so it can count them.
func (s *PageDataFetchSuite) Test_Fetch_DifferentVisitorsDoNotShare() {
	var calls atomic.Int32

	release := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		<-release
		w.Write([]byte(`{"title":"My video"}`))
	}))

	defer server.Close()

	loader := &redirects.Loader{URL: server.URL}
	wg := sync.WaitGroup{}

	for _, ip := range []string{"203.0.113.1", "203.0.113.2"} {
		wg.Add(1)

		go func() {
			defer wg.Done()
			(&pageData{loader: loader, visitor: pageDataVisitor{ip: ip}}).fetch(context.Background())
		}()
	}

	s.Eventually(func() bool { return calls.Load() == 2 }, time.Second, 5*time.Millisecond)
	close(release)
	wg.Wait()
}

// The caller that wins the race may disconnect; the others still get the document.
func (s *PageDataFetchSuite) Test_Fetch_SharedRequestSurvivesWinnerCancellation() {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"title":"My video"}`))
	}))

	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result := (&pageData{loader: &redirects.Loader{URL: server.URL}}).fetch(ctx)

	s.NoError(result.err)
	s.Equal("My video", result.document["title"])
}

func TestPageDataFetch(t *testing.T) {
	suite.Run(t, new(PageDataFetchSuite))
}
