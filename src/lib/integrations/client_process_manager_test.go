package integrations_test

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/stormkit-io/stormkit-io/src/lib/html"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stretchr/testify/suite"
)

type ProcessManagerSuite struct {
	suite.Suite

	tmpdir string
	pm     *integrations.ProcessManager
}

func (s *ProcessManagerSuite) SetupSuite() {
	tmpDir, err := os.MkdirTemp("", "tmp-integrations-pm-")

	s.NoError(err)

	s.tmpdir = tmpDir
	s.pm = integrations.Filesys().ProcessManager()

	s.NoError(os.WriteFile(path.Join(s.tmpdir, "index-auto-terminate.js"), []byte(`
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
			res.end('Hello - I will terminate myself now');

			// Exit the process
			process.exit(0);
		});

		// Make the server listen on the specified port and hostname
		server.listen(port, hostname);
	`), 0664))

	s.NoError(os.WriteFile(path.Join(s.tmpdir, "index.js"), []byte(`
		const http = require('http');
		const spawn = require('child_process').spawn;

		// Define the hostname and port
		const hostname = '127.0.0.1';
		const port = process.env.PORT;

		// Spawn a child process
        const child = spawn('node', ['-e', 'setTimeout(() => {}, 10000)'], { detached: true, stdio: 'ignore' });

		console.log(child.pid);

		// Create the HTTP server
		const server = http.createServer((req, res) => {
			// Set the response HTTP header with HTTP status and Content type
			res.statusCode = 200;
			res.setHeader('Content-Type', 'text/plain');
			// Send the response body "Hello, World!"
			res.end('Hello, ' + process.env.ORIGIN + '!\n');
		});

		// Make the server listen on the specified port and hostname
		server.listen(port, hostname);
	`), 0664))
}

func (s *ProcessManagerSuite) TearDownSuite() {
	if strings.Contains(s.tmpdir, os.TempDir()) {
		os.RemoveAll(s.tmpdir)
	}

	s.pm.KillAll()
}

func (s *ProcessManagerSuite) processExists(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Try sending a signal 0 to check if the process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func (s *ProcessManagerSuite) Test_ProcessKilligItself() {
	args := &integrations.InvokeArgs{
		URL:          &url.URL{},
		ARN:          fmt.Sprintf("local:%s:process_killing_itself", path.Join(s.tmpdir, "index-auto-terminate.js")),
		Method:       shttp.MethodGet,
		Command:      "node index-auto-terminate.js",
		HostName:     "example.org",
		CaptureLogs:  true,
		DeploymentID: 1,
	}

	service, err := s.pm.Start(context.Background(), args, s.tmpdir)
	s.NoError(err)
	s.NotNil(service)

	result, err := s.pm.Invoke(*args, s.tmpdir)

	s.NoError(err)
	s.NotEmpty(result)
	s.Equal("Hello - I will terminate myself now", string(result.Body))
	time.Sleep(1 * time.Second)
	s.Nil(s.pm.GetService(args.ARN))
}

func (s *ProcessManagerSuite) Test_RunningBackgroundService() {
	args := &integrations.InvokeArgs{
		URL:          &url.URL{},
		ARN:          fmt.Sprintf("local:%s:bg_service", path.Join(s.tmpdir, "index.js")),
		Method:       shttp.MethodGet,
		Command:      "node index.js &",
		HostName:     "example.org",
		CaptureLogs:  true,
		DeploymentID: 1,
	}

	service, err := s.pm.Start(context.Background(), args, s.tmpdir)
	s.NoError(err)
	s.NotNil(service)

	result, err := s.pm.Invoke(*args, s.tmpdir)

	s.NoError(err)
	s.NotEmpty(result)
	s.Equal("Hello, https://example.org!\n", string(result.Body))
	time.Sleep(1 * time.Second)
	s.NotNil(s.pm.GetService(args.ARN))
}

func (s *ProcessManagerSuite) Test_Invoke_WithServerCmd() {
	reqURL := &url.URL{}
	fileName := path.Join(s.tmpdir, "index.js")

	result, err := s.pm.Invoke(integrations.InvokeArgs{
		URL:          reqURL,
		ARN:          fmt.Sprintf("local:%s:with_server_cmd", fileName),
		Method:       shttp.MethodGet,
		Command:      "node index.js",
		HostName:     "example.org",
		CaptureLogs:  true,
		DeploymentID: 1,
	}, s.tmpdir)

	s.NoError(err)
	s.NotEmpty(result)
	s.Equal("Hello, https://example.org!\n", string(result.Body))
}

func (s *ProcessManagerSuite) Test_CustomPortHandling_Published() {
	args := &integrations.InvokeArgs{
		URL:          &url.URL{},
		ARN:          fmt.Sprintf("local:%s:custom_port_handling_published", path.Join(s.tmpdir, "index.js")),
		Method:       shttp.MethodGet,
		Command:      "node index.js",
		HostName:     "example.org",
		CaptureLogs:  true,
		IsPublished:  true,
		DeploymentID: 1,
		EnvVariables: map[string]string{
			"PORT": "9001",
		}}

	// Start the first service
	service, err := s.pm.Start(context.Background(), args, s.tmpdir)
	s.NoError(err)
	s.NotNil(service)

	time.Sleep(1 * time.Second)

	// Ensure the second service is running
	s.True(s.processExists(service.Pid()), "Service should be running")

	service.Kill()
}

func (s *ProcessManagerSuite) Test_CustomPortHandling_NotPublished() {
	args := &integrations.InvokeArgs{
		URL:          &url.URL{},
		ARN:          fmt.Sprintf("local:%s:custom_port_handling_not_published", path.Join(s.tmpdir, "index.js")),
		Method:       shttp.MethodGet,
		Command:      "node index.js",
		HostName:     "example.org",
		CaptureLogs:  true,
		IsPublished:  false,
		DeploymentID: 1,
		EnvVariables: map[string]string{
			"PORT": "9002",
		}}

	// Start the first service
	service, err := s.pm.Start(context.Background(), args, s.tmpdir)
	s.Error(err)
	s.Equal("custom ports are only available for published deployments, please remove the PORT environment variable to use dynamic ports", err.Error())
	s.Nil(service)
}

func (s *ProcessManagerSuite) Test_Invoke_WithExistingOrigin() {
	reqURL := &url.URL{}
	fileName := path.Join(s.tmpdir, "index.js")

	result, err := s.pm.Invoke(integrations.InvokeArgs{
		URL:         reqURL,
		ARN:         fmt.Sprintf("local:%s:with_existing_origin", fileName),
		Method:      shttp.MethodGet,
		Command:     "node index.js",
		HostName:    "example.org",
		CaptureLogs: true,
		EnvVariables: map[string]string{
			"ORIGIN": "my-origin.org",
		},
		DeploymentID: 1,
	}, s.tmpdir)

	s.NoError(err)
	s.NotEmpty(result)
	s.Equal("Hello, my-origin.org!\n", string(result.Body))
}

func (s *ProcessManagerSuite) Test_Kill_TerminatesChildProcesses() {
	callbackCalled := make(chan struct{})
	var callbackOnce sync.Once

	// Start the parent process using ProcessManager
	fileName := path.Join(s.tmpdir, "index.js")

	service, err := s.pm.Start(context.Background(), &integrations.InvokeArgs{
		Command:      "node index.js",
		ARN:          fmt.Sprintf("local:%s:parent_handler", fileName),
		CaptureLogs:  true,
		DeploymentID: 1,
		QueueLog: func(log *integrations.Log) {
			childPid := utils.StringToInt(log.Message)
			s.Greater(childPid, 0)

			callbackOnce.Do(func() {
				close(callbackCalled)
			})
		},
	}, s.tmpdir)

	s.NoError(err)
	s.NotNil(service)

	for {
		select {
		case <-callbackCalled:
			// Verify the parent process is running
			parentPID := service.Pid()
			s.True(s.processExists(parentPID), "Parent process should be running")

			// Call Kill on the service
			service.Kill()

			time.Sleep(1 * time.Second)

			// Verify the parent process is terminated
			s.False(s.processExists(parentPID), "Parent process should be terminated")

			return
		case <-time.After(5 * time.Second):
			s.Fail("Timeout waiting for QueueLog callback")
		}
	}
}

func (s *ProcessManagerSuite) Test_ProcessManager_Invoke_CustomPort_Unpublished() {
	result, err := s.pm.Invoke(integrations.InvokeArgs{
		URL:         &url.URL{},
		ARN:         "local:example:custom_port_unpublished",
		Method:      shttp.MethodGet,
		Command:     "node index.js",
		HostName:    "example.org",
		CaptureLogs: true,
		IsPublished: false,
		EnvVariables: map[string]string{
			"PORT": "9003",
		},
		DeploymentID: 1,
	}, s.tmpdir)

	s.NoError(err)
	s.Equal(http.StatusBadRequest, result.StatusCode)
	s.Equal(http.Header{"Content-Type": []string{"text/html"}}, result.Headers)
	s.Equal(strings.Join(strings.Fields(string(html.MustRender(html.RenderArgs{
		PageTitle: "Stormkit - Invalid request",
		PageContent: `<div class="container text-center">
			<h2>Custom ports are only available for published deployments</h2>
			<h3>Please remove the PORT environment variable to use dynamic ports,<br />or access this service via the published URL.</h3>
		</div>`,
	}))), " "), strings.Join(strings.Fields(string(result.Body)), " "))
}

func (s *ProcessManagerSuite) Test_StormkitServerConfig() {
	content, err := yaml.Marshal(map[string]any{
		"workdir": "./my_workdir",
		"setup": []string{
			"touch example.txt",
		},
		"stop": []string{
			"rm -f example.txt",
		},
	})

	s.NoError(err)
	s.NoError(os.WriteFile(path.Join(s.tmpdir, "stormkit.server.yml"), content, 0755))

	args := &integrations.InvokeArgs{
		URL:          &url.URL{},
		ARN:          fmt.Sprintf("local:%s:stormkit_server_config", path.Join(s.tmpdir, "index.js")),
		Method:       shttp.MethodGet,
		Command:      "node ../index.js", // We should be in the workdir and the script is in parent folder
		HostName:     "example.org",
		CaptureLogs:  true,
		DeploymentID: 1,
	}

	result, err := s.pm.Invoke(*args, s.tmpdir)
	s.NoError(err)
	s.NotNil(result)

	// First we should receive a message that the service is being set up
	s.NotEmpty(result.Body)
	s.Contains(string(result.Body), "Service is currently being set up, please try again later.")

	// Now we wait for the service to be set up
	time.Sleep(1 * time.Second)

	// Now we should be able to invoke the service
	result, err = s.pm.Invoke(*args, s.tmpdir)
	s.NoError(err)
	s.Equal("Hello, https://example.org!\n", string(result.Body))

	// As part of the setup, we should have created the example.txt file
	s.True(file.Exists(path.Join(s.tmpdir, "my_workdir", "example.txt")), "example.txt should exist in the workdir")
}

func TestProcessManager(t *testing.T) {
	suite.Run(t, &ProcessManagerSuite{})
}

func Benchmark_ProcessManagerInvoke(b *testing.B) {
	s := new(ProcessManagerSuite)
	s.SetT(&testing.T{})
	s.SetupSuite()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		fileName := path.Join(s.tmpdir, "index.js")

		s.pm.Invoke(integrations.InvokeArgs{
			URL:         &url.URL{},
			ARN:         fmt.Sprintf("local:%s:my_handler", fileName),
			Method:      shttp.MethodGet,
			Command:     "node index.js",
			CaptureLogs: true,
		}, s.tmpdir)
	}

	s.TearDownSuite()
}
