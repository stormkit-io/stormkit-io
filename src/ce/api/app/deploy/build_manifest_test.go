package deploy_test

import (
	"encoding/json"
	"os"
	"path"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stormkit-io/stormkit-io/src/ce/api/app/redirects"
	"github.com/stormkit-io/stormkit-io/src/lib/factory"
	"github.com/stretchr/testify/suite"
)

type BuildManifestSuite struct {
	suite.Suite
	*factory.Factory

	tmpDir string
}

func (s *BuildManifestSuite) BeforeTest(suiteName, _ string) {
	tmpDir, err := os.MkdirTemp("", "build-manifest")
	s.NoError(err)
	s.tmpDir = tmpDir

	s.NoError(os.MkdirAll(path.Join(s.tmpDir, "/my/test"), 0755))
	s.NoError(os.WriteFile(path.Join(s.tmpDir, "/my/test/index.html"), []byte("Hello World!"), 0644))
	s.NoError(os.WriteFile(path.Join(s.tmpDir, "/my/_headers"), []byte(exampleHeaders), 0644))
	s.NoError(os.WriteFile(path.Join(s.tmpDir, "/index.js"), []byte("console.log('Hello World!')"), 0644))
	s.NoError(os.WriteFile(path.Join(s.tmpDir, "/index.js.map"), []byte("static files should ignore me"), 0644))
	s.NoError(os.WriteFile(path.Join(s.tmpDir, "/my/redirects.json"), []byte(`[{ "from": "stormkit.io", "to": "www.stormkit.io" }]`), 0644))
}

func (s *BuildManifestSuite) AfterTest(_, _ string) {
	s.NoError(os.RemoveAll(s.tmpDir))
}

func (s *BuildManifestSuite) Test_PrepareStaticFiles() {
	staticFiles := deploy.PrepareStaticFiles([]string{s.tmpDir}, []deploy.CustomHeader{{
		Key:      "X-Message",
		Value:    "Hello World!",
		Location: "/*",
	}})

	staticFilesJson, err := json.Marshal(staticFiles)
	s.NoError(err)

	expected := `{
		"/index.js": {
			"X-Message": "Hello World!",
			"content-type": "text/javascript; charset=utf-8",
			"etag": "\"20-d40ecc6b3efe529013ed3e79266989c82ecabc47\""
		},
		"/my/_headers": {
			"X-Message": "Hello World!",
			"content-type": "text/plain",
			"etag": "\"20-f1339f6e0b93551dedad8e3bbaba3a3f82e3eacc\""
		},
		"/my/test/index.html": {
			"X-Message": "Hello World!",
			"content-type": "text/html; charset=utf-8",
			"etag": "\"20-2ef7bde608ce5404e97d5f042f95f89f1c232871\""
		},
		"/my/redirects.json": {
			"X-Message": "Hello World!",
			"content-type": "application/json",
			"etag": "\"20-eb82b8d87a82748123086c72bd15134b9124e6b5\""
		}
	}`

	s.JSONEq(expected, string(staticFilesJson))
}

func (s *BuildManifestSuite) Test_ParseRedirects() {
	reds, err := deploy.ParseRedirects([]string{
		path.Join(s.tmpDir, "my", "redirects.json"),
		path.Join(s.tmpDir, "_redirects"),
	})

	s.NoError(err)
	s.Equal([]redirects.Redirect{{From: "stormkit.io", To: "www.stormkit.io"}}, reds)
}

func TestBuildManifestSuite(t *testing.T) {
	suite.Run(t, &BuildManifestSuite{})
}
