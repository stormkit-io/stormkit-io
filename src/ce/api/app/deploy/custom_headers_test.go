package deploy_test

import (
	"os"
	"path"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/ce/api/app/deploy"
	"github.com/stretchr/testify/suite"
)

const exampleHeaders = `
/*
  # This will be applied all file types
  X-Message: Hello World!

  # These will be applied only for Javascript files
/*.js
  # Allow CORS for JS files.
  Access-Control-Allow-Origin: *
  Access-Control-Allow-Headers: *
  Access-Control-Allow-Methods: *
`

type CustomHeadersSuite struct {
	suite.Suite
}

func (s *CustomHeadersSuite) Test_ParseCustomHeadersFile() {
	tmpDir, err := os.MkdirTemp("", "build-manifest")
	s.NoError(err)
	defer os.RemoveAll(tmpDir)

	s.NoError(os.WriteFile(path.Join(tmpDir, "_headers"), []byte(exampleHeaders), 0644))

	headers, err := deploy.ParseHeadersFile(path.Join(tmpDir, "_headers"))
	s.NoError(err)

	s.Len(headers, 4)
	s.Equal("X-Message", headers[0].Key)
	s.Equal("Hello World!", headers[0].Value)
	s.Equal("/*", headers[0].Location)

	s.Equal("Access-Control-Allow-Origin", headers[1].Key)
	s.Equal("*", headers[1].Value)
	s.Equal("/*.js", headers[1].Location)

	s.Equal("Access-Control-Allow-Headers", headers[2].Key)
	s.Equal("*", headers[2].Value)
	s.Equal("/*.js", headers[2].Location)

	s.Equal("Access-Control-Allow-Methods", headers[3].Key)
	s.Equal("*", headers[3].Value)
	s.Equal("/*.js", headers[3].Location)
}

func (s *CustomHeadersSuite) Test_ParseCustomHeaders_Success() {
	headers, err := deploy.ParseHeaders("/*\nX-Message: Hello World!")
	s.NoError(err)
	s.Len(headers, 1)
	s.Equal("X-Message", headers[0].Key)
	s.Equal("Hello World!", headers[0].Value)
	s.Equal("/*", headers[0].Location)
}

func (s *CustomHeadersSuite) Test_ParseCustomHeaders_ValueWithColon() {
	headers, err := deploy.ParseHeaders("/*\nAccess-Control-Allow-Origin: https://example.com")
	s.NoError(err)
	s.Len(headers, 1)
	s.Equal("Access-Control-Allow-Origin", headers[0].Key)
	s.Equal("https://example.com", headers[0].Value)
	s.Equal("/*", headers[0].Location)
}

func (s *CustomHeadersSuite) Test_ParseCustomHeaders_Error() {
	headers, err := deploy.ParseHeaders("/*\nX-Message Hello World!")
	s.Error(err)
	s.Equal("invalid syntax (missing colon): X-Message Hello World!", err.Error())
	s.Empty(headers)
}

func (s *CustomHeadersSuite) Test_ParseCustomHeaders_Empty() {
	headers, err := deploy.ParseHeaders("")
	s.NoError(err)
	s.Empty(headers)
}

func (s *CustomHeadersSuite) Test_ApplyCustomHeaders() {
	headers, err := deploy.ParseHeaders(`
		/*
		X-Message: Hello World!

		/*.js
		X-Custom: Hi

		/*.html 
		X-Test: false
	`)

	s.NoError(err)

	result := deploy.ApplyHeaders("/test.js", nil, headers)
	s.Equal("Hello World!", result["X-Message"])
	s.Equal("Hi", result["X-Custom"])
	s.Equal("", result["X-Test"])
}

func TestCustomHeadersSuite(t *testing.T) {
	suite.Run(t, &CustomHeadersSuite{})
}
