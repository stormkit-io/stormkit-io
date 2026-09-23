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

func (s *PageDataRenderSuite) Test_Render_Value() {
	out := s.renderer.render(`<title>{{data.title}}</title>`, s.document())
	s.Equal("<title>My video</title>", out)
}

func (s *PageDataRenderSuite) Test_Render_InnerWhitespace() {
	out := s.renderer.render(`<title>{{ data.title }}</title>`, s.document())
	s.Equal("<title>My video</title>", out)
}

func (s *PageDataRenderSuite) Test_Render_NestedKey() {
	out := s.renderer.render(`<span>{{data.owner.name}}</span>`, s.document())
	s.Equal("<span>Alice</span>", out)
}

func (s *PageDataRenderSuite) Test_Render_Numbers() {
	out := s.renderer.render(`{{data.views}}|{{data.ratio}}`, s.document())
	s.Equal("12|1.5", out)
}

// A default fires on absent, null and empty — never on false or zero, which are
// answers the upstream gave.
func (s *PageDataRenderSuite) Test_Render_Default_OnMissingNullAndEmpty() {
	document := s.document()
	document["subtitle"] = nil

	s.Equal("noindex", s.renderer.render(`{{data.robots:-noindex}}`, document))
	s.Equal("none", s.renderer.render(`{{data.subtitle:-none}}`, document))
	s.Equal("/og.png", s.renderer.render(`{{data.thumbnail:-/og.png}}`, document))
}

func (s *PageDataRenderSuite) Test_Render_Default_NotAppliedToFalseOrZero() {
	document := s.document()
	document["views"] = float64(0)

	s.Equal("false", s.renderer.render(`{{data.public:-true}}`, document))
	s.Equal("0", s.renderer.render(`{{data.views:-42}}`, document))
}

func (s *PageDataRenderSuite) Test_Render_Default_TrimmedAndOptional() {
	s.Equal("noindex", s.renderer.render(`{{ data.robots :- noindex }}`, s.document()))
	s.Equal("", s.renderer.render(`{{data.robots}}`, s.document()))
	s.Equal("", s.renderer.render(`{{data.robots:-}}`, s.document()))
}

// A value lands in attribute position as often as in text, so it is escaped
// either way: an unescaped quote from the API would end the attribute.
func (s *PageDataRenderSuite) Test_Render_EscapesUpstreamValues() {
	document := s.document()
	document["title"] = `" onload="alert(1)`

	out := s.renderer.render(`<video id="{{data.title}}">`, document)
	s.Equal(`<video id="&#34; onload=&#34;alert(1)">`, out)
	s.NotContains(out, `onload="alert`)
}

// The author's own default is their markup, so it is left alone.
func (s *PageDataRenderSuite) Test_Render_DoesNotEscapeDefault() {
	s.Equal(`a&b`, s.renderer.render(`{{data.missing:-a&b}}`, s.document()))
}

func (s *PageDataRenderSuite) Test_Render_WholeDocument() {
	document := map[string]any{"title": "<script>"}
	out := s.renderer.render(`<script id="__sk_data__" type="application/json">{{data}}</script>`, document)

	// json.Marshal escapes <, > and & so the block cannot close early.
	s.Equal("<script id=\"__sk_data__\" type=\"application/json\">{\"title\":\"\\u003cscript\\u003e\"}</script>", out)
}

func (s *PageDataRenderSuite) Test_Render_ObjectAndArrayValues() {
	out := s.renderer.render(`{{data.tags}}`, s.document())
	s.Equal(`[&#34;a&#34;,&#34;b&#34;]`, out)
}

func (s *PageDataRenderSuite) Test_Render_LeavesForeignTokensAlone() {
	out := s.renderer.render(`{{ other.title }} {{SK_REQUEST_ID}}`, s.document())
	s.Equal(`{{ other.title }} {{SK_REQUEST_ID}}`, out)
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
