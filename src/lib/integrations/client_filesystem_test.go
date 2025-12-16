package integrations_test

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"github.com/stormkit-io/stormkit-io/src/mocks"
	"github.com/stretchr/testify/suite"
)

type FilesysSuite struct {
	suite.Suite

	tmpdir   string
	mockExec *mocks.CommandInterface
}

func (s *FilesysSuite) BeforeTest(suiteName, _ string) {
	tmpDir, err := os.MkdirTemp("", "deployment-")

	s.NoError(err)

	s.tmpdir = tmpDir

	// Set up mock command for tests that use it
	s.mockExec = &mocks.CommandInterface{}
	sys.DefaultCommand = s.mockExec

	// Create test artifacts only for non-Upload tests
	clientDir := path.Join(tmpDir, "client")
	s.NoError(os.MkdirAll(clientDir, 0774))
	s.NoError(os.WriteFile(path.Join(clientDir, "index.html"), []byte("Hello world"), 0664))
	s.NoError(file.ZipV2(file.ZipArgs{Source: []string{clientDir}, ZipName: path.Join(tmpDir, "sk-client.zip")}))
	s.NoError(file.ZipV2(file.ZipArgs{Source: []string{clientDir}, ZipName: path.Join(tmpDir, "sk-server.zip")}))
	s.NoError(file.ZipV2(file.ZipArgs{Source: []string{clientDir}, ZipName: path.Join(tmpDir, "sk-api.zip")}))
	s.NoError(file.ZipV2(file.ZipArgs{Source: []string{clientDir}, ZipName: path.Join(tmpDir, "sk-migrations.zip")}))
}

func (s *FilesysSuite) AfterTest(_, _ string) {
	sys.DefaultCommand = nil

	if strings.Contains(s.tmpdir, os.TempDir()) {
		os.RemoveAll(s.tmpdir)
	}
}

func (s *FilesysSuite) Test_Upload() {
	client := integrations.Filesys()
	dist := path.Join(s.tmpdir, "dist")

	s.NoError(os.Mkdir(dist, 0774))

	clientZip := path.Join(s.tmpdir, "sk-client.zip")
	serverZip := path.Join(s.tmpdir, "sk-server.zip")
	apiZip := path.Join(s.tmpdir, "sk-api.zip")
	migrationsZip := path.Join(s.tmpdir, "sk-migrations.zip")

	zipConfigs := []struct {
		zipPath   string
		targetDir string
	}{
		{clientZip, "client"},
		{serverZip, "server"},
		{apiZip, "api"},
		// No need for migrations as it will be copied to target dir
	}

	for _, config := range zipConfigs {
		s.mockExec.On("SetOpts", sys.CommandOpts{
			Name:   "unzip",
			Args:   []string{"-o", config.zipPath, "-d", path.Join(dist, "deployment-50919", config.targetDir)},
			Stdout: io.Discard,
			Stderr: os.Stderr,
		}).Return(s.mockExec).Once()

		s.mockExec.On("Run").Return(nil).Once()
	}

	result, err := client.Upload(integrations.UploadArgs{
		DistDir:       dist,
		AppID:         232,
		EnvID:         591,
		DeploymentID:  50919,
		ClientZip:     clientZip,
		ServerZip:     serverZip,
		APIZip:        apiZip,
		MigrationsZip: migrationsZip,
		ServerHandler: "stormkit-server.js:handler",
		APIHandler:    "stormkit-api.mjs:handler",
	})

	s.NoError(err)
	s.Greater(result.API.BytesUploaded, int64(1))
	s.Greater(result.Server.BytesUploaded, int64(1))
	s.Greater(result.Client.BytesUploaded, int64(1))
	s.Greater(result.Migrations.BytesUploaded, int64(1))
	s.Equal(fmt.Sprintf("local:%s/deployment-50919/client", dist), result.Client.Location)
	s.Equal(fmt.Sprintf("local:%s/deployment-50919/migrations/sk-migrations.zip", dist), result.Migrations.Location)
	s.Equal(fmt.Sprintf("local:%s/deployment-50919/server/stormkit-server.js:handler", dist), result.Server.Location)
	s.Equal(fmt.Sprintf("local:%s/deployment-50919/api/stormkit-api.mjs:handler", dist), result.API.Location)

	s.mockExec.AssertExpectations(s.T())
}

func (s *FilesysSuite) Test_Upload_WithMissingZips() {
	client := integrations.Filesys()
	dist := path.Join(s.tmpdir, "dist")

	s.NoError(os.Mkdir(dist, 0774))

	clientZip := path.Join(s.tmpdir, "sk-client.zip")
	clientDir := path.Join(dist, "deployment-50920", "client")

	s.mockExec.On("SetOpts", sys.CommandOpts{
		Name:   "unzip",
		Args:   []string{"-o", clientZip, "-d", clientDir},
		Stdout: io.Discard,
		Stderr: os.Stderr,
	}).Return(s.mockExec).Once()

	s.mockExec.On("Run").Return(nil).Once()

	result, err := client.Upload(integrations.UploadArgs{
		DistDir:      dist,
		AppID:        232,
		EnvID:        591,
		DeploymentID: 50920,
		ClientZip:    clientZip,
	})

	s.NoError(err)
	s.Greater(result.Client.BytesUploaded, int64(1))
	s.Equal(int64(0), result.Server.BytesUploaded)
	s.Equal(int64(0), result.API.BytesUploaded)
	s.Equal(int64(0), result.Migrations.BytesUploaded)
	s.Equal("", result.Server.Location)
	s.Equal("", result.API.Location)
	s.Equal("", result.Migrations.Location)

	s.mockExec.AssertExpectations(s.T())
}

func (s *FilesysSuite) Test_DeleteArtifacts() {
	client := integrations.Filesys()

	s.DirExists(s.tmpdir)
	s.NoError(client.DeleteArtifacts(context.Background(), integrations.DeleteArtifactsArgs{APILocation: s.tmpdir}))
	s.NoDirExists(s.tmpdir)
}

func (s *FilesysSuite) Test_GetFile() {
	client := integrations.Filesys()
	filePath := path.Join(s.tmpdir, "client", "index.html")

	file, err := client.GetFile(integrations.GetFileArgs{
		Location: fmt.Sprintf("local:%s", filePath),
	})

	s.NoError(err)

	stat, err := os.Stat(filePath)

	s.NoError(err)
	s.Equal(file.Size, stat.Size())
	s.Equal("Hello world", string(file.Content))
	s.Equal("text/html; charset=utf-8", file.ContentType)
}

func (s *FilesysSuite) Test_Invoke() {
	client := integrations.Filesys()
	reqURL := &url.URL{}

	s.mockExec.On("SetOpts", sys.CommandOpts{
		Name: "node",
		Args: []string{"-e", `require("./index.js").my_handler({"method":"PUT","url":"","path":"","headers":{"host":""},"captureLogs":true}, {}, (e,r) => console.log(JSON.stringify(r)))`},
		Env:  []string{},
		Dir:  s.tmpdir,
	}).Return(s.mockExec).Once()

	response := `{ 
		"body": "hello world!", 
		"status": 200,
		"headers": { 
			"content-type": "text/html" 
		}
	}`

	s.mockExec.On("CombinedOutput").Return([]byte(response), nil).Once()

	result, err := client.Invoke(integrations.InvokeArgs{
		URL:         reqURL,
		ARN:         fmt.Sprintf("local:%s:my_handler", path.Join(s.tmpdir, "index.js")),
		Method:      shttp.MethodPut,
		CaptureLogs: true,
	})

	s.NoError(err)
	s.NotEmpty(result)
	s.Equal(200, result.StatusCode)
	s.Equal("hello world!", string(result.Body))
	s.Equal("text/html", result.Headers.Get("content-type"))
}

func (s *FilesysSuite) Test_Invoke_WithServerCmd() {
	sys.DefaultCommand = nil

	s.NoError(os.WriteFile(path.Join(s.tmpdir, "index.js"), []byte(`
		const http = require('http');

		// Define the hostname and port
		const hostname = '127.0.0.1';
		const port = process.env.PORT;

		// Create the HTTP server
		const server = http.createServer((req, res) => {
			// Set the response HTTP header with HTTP status and Content type
			res.statusCode = 200;
			res.setHeader('Content-Type', 'text/plain');
			// Send the response body "Hello, World!"
			res.end('Hello, World!\n');
		});

		// Make the server listen on the specified port and hostname
		server.listen(port, hostname);
	`), 0664))

	client := integrations.Filesys()
	defer client.ProcessManager().KillAll()

	reqURL := &url.URL{}
	fileName := path.Join(s.tmpdir, "index.js")

	result, err := client.Invoke(integrations.InvokeArgs{
		URL:          reqURL,
		ARN:          fmt.Sprintf("local:%s:my_handler", fileName),
		Method:       shttp.MethodGet,
		Command:      fmt.Sprintf("node %s", fileName),
		DeploymentID: 1,
		CaptureLogs:  true,
	})

	s.NoError(err)
	s.NotEmpty(result)
	s.Equal("Hello, World!\n", string(result.Body))
}

func TestFilesys(t *testing.T) {
	suite.Run(t, &FilesysSuite{})
}

func BenchmarkInvokeWithServerCommand(b *testing.B) {
	s := new(FilesysSuite)
	s.SetT(&testing.T{})
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.BeforeTest("Test_Invoke_WithServerCmd", "")
		b.StartTimer()
		s.Test_Invoke_WithServerCmd()
		b.StopTimer()
		s.AfterTest("Test_Invoke_WithServerCmd", "")
	}
}
