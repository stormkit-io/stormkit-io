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

	"github.com/stormkit-io/stormkit-io/src/lib/html"
	"github.com/stormkit-io/stormkit-io/src/lib/integrations"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/nixstore"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/nixstore/nixstoretest"
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

// invokeWithRetry retries Invoke until the service responds without Retry-After
// (i.e. it is up and serving real traffic). Fails the test after 3 seconds.
func (s *ProcessManagerSuite) invokeWithRetry(args integrations.InvokeArgs, workDir string) (*integrations.InvokeResult, error) {
	deadline := time.Now().Add(3 * time.Second)

	for time.Now().Before(deadline) {
		result, err := s.pm.Invoke(args, workDir)

		if err != nil {
			return nil, err
		}

		if result.Headers.Get("Retry-After") == "" {
			return result, nil
		}

		time.Sleep(50 * time.Millisecond)
	}

	return nil, fmt.Errorf("service did not become ready within 3s")
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

	result, err := s.invokeWithRetry(*args, s.tmpdir)

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

	result, err := s.invokeWithRetry(*args, s.tmpdir)

	s.NoError(err)
	s.NotEmpty(result)
	s.Equal("Hello, https://example.org!\n", string(result.Body))
	time.Sleep(1 * time.Second)
	s.NotNil(s.pm.GetService(args.ARN))
}

func (s *ProcessManagerSuite) Test_Invoke_WithServerCmd() {
	reqURL := &url.URL{}
	fileName := path.Join(s.tmpdir, "index.js")

	result, err := s.invokeWithRetry(integrations.InvokeArgs{
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

func (s *ProcessManagerSuite) Test_Invoke_WarmingUp_ReturnsServiceUnavailable() {
	fileName := path.Join(s.tmpdir, "index.js")

	result, err := s.pm.Invoke(integrations.InvokeArgs{
		URL:          &url.URL{},
		ARN:          fmt.Sprintf("local:%s:warming_up", fileName),
		Method:       shttp.MethodGet,
		Command:      "node index.js",
		HostName:     "example.org",
		CaptureLogs:  true,
		DeploymentID: 1,
	}, s.tmpdir)

	s.NoError(err)
	s.Require().NotNil(result)
	s.Equal("1", result.Headers.Get("Retry-After"), "expected the warming-up interstitial")
	s.Equal(http.StatusServiceUnavailable, result.StatusCode)
	s.Contains(string(result.Body), "Service not yet started")
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

	result, err := s.invokeWithRetry(integrations.InvokeArgs{
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

func (s *ProcessManagerSuite) Test_Kill_AlreadyKilled() {
	args := integrations.InvokeArgs{
		URL:          &url.URL{},
		ARN:          fmt.Sprintf("local:%s:kill_already_killed", path.Join(s.tmpdir, "index.js")),
		Method:       shttp.MethodGet,
		Command:      "node index.js",
		HostName:     "example.org",
		CaptureLogs:  true,
		DeploymentID: 1,
	}

	_, err := s.invokeWithRetry(args, s.tmpdir)
	s.NoError(err)

	service := s.pm.GetService(args.ARN)
	s.NotNil(service)

	service.Kill()
	s.Nil(s.pm.GetService(args.ARN))

	// Calling Kill a second time must not panic and must not remove a
	// replacement service that may have been registered for the same ARN.
	s.NotPanics(func() { service.Kill() })
	s.Nil(s.pm.GetService(args.ARN))
}

func (s *ProcessManagerSuite) Test_Invoke_StaleProcess() {
	arn := fmt.Sprintf("local:%s:stale_process", path.Join(s.tmpdir, "index.js"))

	args := integrations.InvokeArgs{
		URL:          &url.URL{},
		ARN:          arn,
		Method:       shttp.MethodGet,
		Command:      "node index.js",
		HostName:     "example.org",
		CaptureLogs:  true,
		DeploymentID: 1,
	}

	// First invocation starts the service.
	result, err := s.invokeWithRetry(args, s.tmpdir)
	s.NoError(err)
	s.Equal("Hello, https://example.org!\n", string(result.Body))

	// Kill the running service to simulate a crashed process.
	service := s.pm.GetService(arn)
	s.NotNil(service)
	service.Kill()
	s.Nil(s.pm.GetService(arn))

	// Second invocation must restart and succeed.
	result, err = s.invokeWithRetry(args, s.tmpdir)
	s.NoError(err)
	s.Equal("Hello, https://example.org!\n", string(result.Body))
}

func (s *ProcessManagerSuite) Test_Invoke_FailedStart_EvictsService() {
	arn := fmt.Sprintf("local:%s:failed_start", path.Join(s.tmpdir, "index.js"))

	args := integrations.InvokeArgs{
		URL:          &url.URL{},
		ARN:          arn,
		Method:       shttp.MethodGet,
		Command:      "stormkit-binary-that-does-not-exist",
		HostName:     "example.org",
		CaptureLogs:  true,
		DeploymentID: 1,
	}

	result, err := s.pm.Invoke(args, s.tmpdir)
	s.NoError(err)
	s.Require().NotNil(result)
	s.Equal(http.StatusServiceUnavailable, result.StatusCode)

	// The failed start must evict the service, otherwise every subsequent
	// Invoke would keep returning the warm-up interstitial forever.
	deadline := time.Now().Add(3 * time.Second)

	for time.Now().Before(deadline) {
		if s.pm.GetService(arn) == nil {
			break
		}

		s.pm.Invoke(args, s.tmpdir)
		time.Sleep(50 * time.Millisecond)
	}

	s.Nil(s.pm.GetService(arn), "expected the service to be evicted after a failed start")

	// A subsequent invocation with a working command must be able to start.
	args.Command = "node index.js"

	result, err = s.invokeWithRetry(args, s.tmpdir)
	s.NoError(err)
	s.Equal("Hello, https://example.org!\n", string(result.Body))
}

func (s *ProcessManagerSuite) Test_BuildServerCommand_WithoutFlake() {
	cmd := s.pm.BuildServerCommand(integrations.BuildServerCommandParams{
		Command: "node index.js",
		WorkDir: s.tmpdir,
	})

	s.Equal("node index.js", cmd)
}

func (s *ProcessManagerSuite) Test_BuildServerCommand_WithFlake() {
	flakePath := path.Join(s.tmpdir, "flake.nix")
	s.NoError(os.WriteFile(flakePath, []byte("{}"), 0664))

	defer os.Remove(flakePath)

	cmd := s.pm.BuildServerCommand(integrations.BuildServerCommandParams{
		Command: "node index.js",
		WorkDir: s.tmpdir,
	})

	s.Equal(`nix --extra-experimental-features "nix-command flakes" develop --command sh -c 'node index.js'`, cmd)
}

// A service that goes idle is killed, so its packages survive a collection only
// while a profile roots them.
func (s *ProcessManagerSuite) Test_BuildServerCommand_RootsTheEnvironment() {
	flakePath := path.Join(s.tmpdir, "flake.nix")
	s.NoError(os.WriteFile(flakePath, []byte("{}"), 0664))

	defer os.Remove(flakePath)

	originalStore, originalProfiles := nixstore.DefaultPath, nixstore.ProfilesDir
	nixstore.DefaultPath = s.T().TempDir()
	nixstore.ProfilesDir = path.Join(s.T().TempDir(), "stormkit")
	restore := nixstoretest.StubLookPath(true)

	defer func() {
		nixstore.DefaultPath, nixstore.ProfilesDir = originalStore, originalProfiles
		restore()
	}()

	cmd := s.pm.BuildServerCommand(integrations.BuildServerCommandParams{
		Command: "node index.js",
		WorkDir: s.tmpdir,
		AppID:   types.ID(42),
		EnvID:   types.ID(7),
	})

	s.Contains(cmd, `--profile "`+path.Join(nixstore.ProfilesDir, "app-42-env-7")+`"`)
	s.Contains(cmd, `--command sh -c 'node index.js'`)
	s.DirExists(nixstore.ProfilesDir)
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
