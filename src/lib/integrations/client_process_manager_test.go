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

	// Traps SIGTERM and stays up, the way a server with a graceful shutdown
	// that never completes would.
	s.NoError(os.WriteFile(path.Join(s.tmpdir, "index-ignores-sigterm.js"), []byte(`
		const http = require('http');
		const fs = require('fs');

		process.on('SIGTERM', () => {});

		fs.appendFileSync(process.env.PID_LOG, process.pid + "\n");

		const server = http.createServer((req, res) => {
			res.statusCode = 200;
			res.end('ok');
		});

		server.listen(process.env.PORT, '127.0.0.1');
	`), 0664))

	// Appends its own pid to PID_LOG on boot, so a test can count how many
	// processes a series of invocations actually spawned.
	s.NoError(os.WriteFile(path.Join(s.tmpdir, "index-pid-log.js"), []byte(`
		const http = require('http');
		const fs = require('fs');

		fs.appendFileSync(process.env.PID_LOG, process.pid + "\n");

		const server = http.createServer((req, res) => {
			res.statusCode = 200;
			res.end('ok');
		});

		server.listen(process.env.PORT, '127.0.0.1');
	`), 0664))
}

// pidLog creates an empty pid log file and returns its path.
func (s *ProcessManagerSuite) pidLog(name string) string {
	logPath := path.Join(s.tmpdir, name)

	s.NoError(os.WriteFile(logPath, []byte{}, 0664))

	return logPath
}

// spawnedPids returns the pids recorded in the given pid log.
func (s *ProcessManagerSuite) spawnedPids(logPath string) []int {
	contents, err := os.ReadFile(logPath)

	s.NoError(err)

	pids := []int{}

	for line := range strings.SplitSeq(strings.TrimSpace(string(contents)), "\n") {
		if line != "" {
			pids = append(pids, utils.StringToInt(line))
		}
	}

	return pids
}

// waitUntilGone polls until none of the given pids exist, up to 3 seconds.
func (s *ProcessManagerSuite) waitUntilGone(pids []int) {
	deadline := time.Now().Add(3 * time.Second)

	for time.Now().Before(deadline) {
		alive := false

		for _, pid := range pids {
			if s.processExists(pid) {
				alive = true
				break
			}
		}

		if !alive {
			return
		}

		time.Sleep(50 * time.Millisecond)
	}

	for _, pid := range pids {
		s.False(s.processExists(pid), "process %d should have been terminated", pid)
	}
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

// Two requests arriving for a cold ARN used to spawn a server each. Only one
// of them ended up in the services map; the other kept running and listening
// with nothing tracking it, so neither the idle timer nor a later kill could
// ever reach it.
func (s *ProcessManagerSuite) Test_Invoke_ConcurrentStart_SpawnsOneProcess() {
	logPath := s.pidLog("pids-concurrent.log")
	arn := fmt.Sprintf("local:%s:concurrent_start", path.Join(s.tmpdir, "index-pid-log.js"))

	args := integrations.InvokeArgs{
		URL:          &url.URL{},
		ARN:          arn,
		Method:       shttp.MethodGet,
		Command:      "node index-pid-log.js",
		HostName:     "example.org",
		DeploymentID: 1,
		EnvVariables: map[string]string{"PID_LOG": logPath},
	}

	var wg sync.WaitGroup

	for range 8 {
		wg.Go(func() {
			s.pm.Invoke(args, s.tmpdir)
		})
	}

	wg.Wait()

	result, err := s.invokeWithRetry(args, s.tmpdir)
	s.NoError(err)
	s.Equal("ok", string(result.Body))

	pids := s.spawnedPids(logPath)
	s.Len(pids, 1, "concurrent invocations must spawn exactly one process")

	service := s.pm.GetService(arn)
	s.Require().NotNil(service)
	s.Equal(pids[0], service.Pid())

	service.Kill()
	s.waitUntilGone(pids)
}

// The process is spawned from a goroutine, so a Kill can arrive before there
// is a pid to signal. It used to return early in that case and the process
// came up a moment later with nothing owning it.
func (s *ProcessManagerSuite) Test_Kill_BeforeSpawn_LeavesNoProcess() {
	logPath := s.pidLog("pids-kill-before-spawn.log")

	service, err := s.pm.Start(context.Background(), &integrations.InvokeArgs{
		URL:          &url.URL{},
		ARN:          fmt.Sprintf("local:%s:kill_before_spawn", path.Join(s.tmpdir, "index-pid-log.js")),
		Method:       shttp.MethodGet,
		Command:      "node index-pid-log.js",
		HostName:     "example.org",
		DeploymentID: 1,
		EnvVariables: map[string]string{"PID_LOG": logPath},
	}, s.tmpdir)

	s.NoError(err)
	s.Require().NotNil(service)

	// Racing the start goroutine on purpose: whichever side wins, the
	// service must not be left with a running process.
	service.Kill()

	time.Sleep(500 * time.Millisecond)

	s.waitUntilGone(s.spawnedPids(logPath))
}

// A service that is no longer the one registered for its ARN can never be
// reached by a request or by the idle timer, so it would run until the host
// restarted. The reaper is the backstop for that.
func (s *ProcessManagerSuite) Test_ReapOrphans_KillsUnregisteredService() {
	logPath := s.pidLog("pids-orphan.log")

	service, err := s.pm.Start(context.Background(), &integrations.InvokeArgs{
		URL:          &url.URL{},
		ARN:          fmt.Sprintf("local:%s:orphan", path.Join(s.tmpdir, "index-pid-log.js")),
		Method:       shttp.MethodGet,
		Command:      "node index-pid-log.js",
		HostName:     "example.org",
		DeploymentID: 1,
		EnvVariables: map[string]string{"PID_LOG": logPath},
	}, s.tmpdir)

	s.NoError(err)
	s.Require().NotNil(service)

	deadline := time.Now().Add(3 * time.Second)

	for time.Now().Before(deadline) && service.Pid() == 0 {
		time.Sleep(50 * time.Millisecond)
	}

	pid := service.Pid()
	s.Require().Greater(pid, 0)

	// The service was never registered, so it is an orphan by definition.
	s.GreaterOrEqual(s.pm.ReapOrphans(0), 1)
	s.waitUntilGone([]int{pid})

	// The suite shares one process manager, so other tests may leave their
	// own services behind: assert on this one rather than on a global count.
	s.pm.ReapOrphans(0)
	s.False(s.processExists(pid))
}

// A registered service is doing its job and must survive a sweep, however long
// it has been up.
func (s *ProcessManagerSuite) Test_ReapOrphans_KeepsRegisteredService() {
	logPath := s.pidLog("pids-registered.log")
	arn := fmt.Sprintf("local:%s:registered", path.Join(s.tmpdir, "index-pid-log.js"))

	args := integrations.InvokeArgs{
		URL:          &url.URL{},
		ARN:          arn,
		Method:       shttp.MethodGet,
		Command:      "node index-pid-log.js",
		HostName:     "example.org",
		DeploymentID: 1,
		EnvVariables: map[string]string{"PID_LOG": logPath},
	}

	_, err := s.invokeWithRetry(args, s.tmpdir)
	s.NoError(err)

	service := s.pm.GetService(arn)
	s.Require().NotNil(service)

	pid := service.Pid()
	s.Require().Greater(pid, 0)
	s.pm.ReapOrphans(0)
	s.True(s.processExists(pid), "a registered service must not be reaped")

	service.Kill()
	s.waitUntilGone([]int{pid})
}

// Killing a service must not unregister a different one that has since taken
// its ARN, or reaping an orphan would evict the healthy service serving that
// ARN and the next sweep would reap that one too.
func (s *ProcessManagerSuite) Test_Kill_DoesNotUnregisterItsReplacement() {
	logPath := s.pidLog("pids-replacement.log")
	arn := fmt.Sprintf("local:%s:replacement", path.Join(s.tmpdir, "index-pid-log.js"))

	args := integrations.InvokeArgs{
		URL:          &url.URL{},
		ARN:          arn,
		Method:       shttp.MethodGet,
		Command:      "node index-pid-log.js",
		HostName:     "example.org",
		DeploymentID: 1,
		EnvVariables: map[string]string{"PID_LOG": logPath},
	}

	// An unregistered service holding the same ARN, as an escaped start would.
	orphan, err := s.pm.Start(context.Background(), &args, s.tmpdir)
	s.NoError(err)
	s.Require().NotNil(orphan)

	_, err = s.invokeWithRetry(args, s.tmpdir)
	s.NoError(err)

	registered := s.pm.GetService(arn)
	s.Require().NotNil(registered)
	s.NotSame(orphan, registered)

	orphan.Kill()

	s.Same(registered, s.pm.GetService(arn), "killing the orphan must leave the registered service in place")
	s.True(s.processExists(registered.Pid()), "the registered service must still be running")

	pids := s.spawnedPids(logPath)
	registered.Kill()
	s.waitUntilGone(pids)
}

// SIGTERM is a request, not a guarantee. A server that traps it and never
// exits used to keep its port for the lifetime of the host.
func (s *ProcessManagerSuite) Test_ReapOrphans_ForceKillsServiceThatIgnoresSigterm() {
	logPath := s.pidLog("pids-ignores-sigterm.log")

	service, err := s.pm.Start(context.Background(), &integrations.InvokeArgs{
		URL:          &url.URL{},
		ARN:          fmt.Sprintf("local:%s:ignores_sigterm", path.Join(s.tmpdir, "index-ignores-sigterm.js")),
		Method:       shttp.MethodGet,
		Command:      "node index-ignores-sigterm.js",
		HostName:     "example.org",
		DeploymentID: 1,
		EnvVariables: map[string]string{"PID_LOG": logPath},
	}, s.tmpdir)

	s.NoError(err)
	s.Require().NotNil(service)

	deadline := time.Now().Add(3 * time.Second)

	for time.Now().Before(deadline) && len(s.spawnedPids(logPath)) == 0 {
		time.Sleep(50 * time.Millisecond)
	}

	pid := service.Pid()
	s.Require().Greater(pid, 0)

	service.Kill()

	time.Sleep(500 * time.Millisecond)
	s.True(s.processExists(pid), "the server is expected to survive SIGTERM")

	s.GreaterOrEqual(s.pm.ReapOrphans(0), 1)
	s.waitUntilGone([]int{pid})
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

// idleArgs returns the invocation of a long-running deployment in a shared
// environment, so tests can publish one deployment over another.
func (s *ProcessManagerSuite) idleArgs(name string, deploymentID types.ID, published bool, maxIdle string) integrations.InvokeArgs {
	vars := map[string]string{
		"PID_LOG":           s.pidLog(fmt.Sprintf("pids-%s.log", name)),
		"STORMKIT_MAX_IDLE": maxIdle,
	}

	return integrations.InvokeArgs{
		URL:          &url.URL{},
		ARN:          fmt.Sprintf("local:%s:%s", path.Join(s.tmpdir, "index-pid-log.js"), name),
		Method:       shttp.MethodGet,
		Command:      "node index-pid-log.js",
		HostName:     "example.org",
		EnvID:        659,
		DeploymentID: deploymentID,
		IsPublished:  published,
		EnvVariables: vars,
	}
}

// killService stops the service registered for the ARN and waits for its process to exit.
func (s *ProcessManagerSuite) killService(arn string) {
	service := s.pm.GetService(arn)
	s.Require().NotNil(service)

	pid := service.Pid()
	service.Kill()
	s.waitUntilGone([]int{pid})
}

// Test_MaxIdleZero_KeepsOnlyPublishedWarm verifies that STORMKIT_MAX_IDLE=0
// keeps the published deployment warm, and that a deployment replaced by a
// newer publish falls back to the default idle timeout without further traffic.
func (s *ProcessManagerSuite) Test_MaxIdleZero_KeepsOnlyPublishedWarm() {
	first := s.idleArgs("max_idle_first", 6591, true, "0")
	second := s.idleArgs("max_idle_second", 6592, true, "0")

	_, err := s.invokeWithRetry(first, s.tmpdir)
	s.Require().NoError(err)

	firstService := s.pm.GetService(first.ARN)
	s.Require().NotNil(firstService)
	s.Equal(time.Duration(0), firstService.IdleTimeout())
	s.False(firstService.HasIdleTimer())

	_, err = s.invokeWithRetry(second, s.tmpdir)
	s.Require().NoError(err)

	secondService := s.pm.GetService(second.ARN)
	s.Require().NotNil(secondService)
	s.Equal(time.Duration(0), secondService.IdleTimeout())
	s.False(secondService.HasIdleTimer())
	s.Equal(10*time.Minute, firstService.IdleTimeout())
	s.True(firstService.HasIdleTimer(), "the replaced deployment must wind down without further requests")

	// Rolling back to the first deployment keeps it warm again.
	_, err = s.invokeWithRetry(first, s.tmpdir)
	s.Require().NoError(err)

	s.Equal(time.Duration(0), firstService.IdleTimeout())
	s.False(firstService.HasIdleTimer())
	s.Equal(10*time.Minute, secondService.IdleTimeout())
	s.True(secondService.HasIdleTimer())

	s.killService(first.ARN)
	s.killService(second.ARN)
}

// Test_MaxIdleZero_PreviewUsesDefault verifies that a deployment that is not
// published is never kept warm.
func (s *ProcessManagerSuite) Test_MaxIdleZero_PreviewUsesDefault() {
	args := s.idleArgs("max_idle_preview", 6593, false, "0")

	_, err := s.invokeWithRetry(args, s.tmpdir)
	s.Require().NoError(err)

	service := s.pm.GetService(args.ARN)
	s.Require().NotNil(service)
	s.Equal(10*time.Minute, service.IdleTimeout())
	s.True(service.HasIdleTimer())

	s.killService(args.ARN)
}

func (s *ProcessManagerSuite) Test_MaxIdle_InvalidValueUsesDefault() {
	for _, value := range []string{"abc", "-5", ""} {
		args := s.idleArgs("max_idle_invalid_"+value, 6594, true, value)

		_, err := s.invokeWithRetry(args, s.tmpdir)
		s.Require().NoError(err)

		service := s.pm.GetService(args.ARN)
		s.Require().NotNil(service)
		s.Equal(10*time.Minute, service.IdleTimeout(), "value %q", value)

		s.killService(args.ARN)
	}
}

func (s *ProcessManagerSuite) Test_MaxIdle_CustomValue() {
	args := s.idleArgs("max_idle_custom", 6595, false, "3")

	_, err := s.invokeWithRetry(args, s.tmpdir)
	s.Require().NoError(err)

	service := s.pm.GetService(args.ARN)
	s.Require().NotNil(service)
	s.Equal(3*time.Minute, service.IdleTimeout())

	s.killService(args.ARN)
}

// Test_ReapOrphans_StopsServiceWhoseFolderWasRemoved verifies that deleting a
// deployment's artifacts stops its runtime, even when it is kept warm.
func (s *ProcessManagerSuite) Test_ReapOrphans_StopsServiceWhoseFolderWasRemoved() {
	workDir := path.Join(s.tmpdir, "deleted-deployment")
	s.Require().NoError(os.MkdirAll(workDir, 0775))

	script, err := os.ReadFile(path.Join(s.tmpdir, "index-pid-log.js"))
	s.Require().NoError(err)
	s.Require().NoError(os.WriteFile(path.Join(workDir, "index-pid-log.js"), script, 0664))

	args := s.idleArgs("deleted_folder", 6596, true, "0")
	args.ARN = fmt.Sprintf("local:%s:deleted_folder", path.Join(workDir, "index-pid-log.js"))

	_, err = s.invokeWithRetry(args, workDir)
	s.Require().NoError(err)

	service := s.pm.GetService(args.ARN)
	s.Require().NotNil(service)

	pid := service.Pid()
	s.Require().Greater(pid, 0)

	s.pm.ReapOrphans(0)
	s.True(s.processExists(pid), "a service whose folder exists must not be reaped")

	s.Require().NoError(os.RemoveAll(workDir))
	s.GreaterOrEqual(s.pm.ReapOrphans(0), 1)
	s.waitUntilGone([]int{pid})
	s.Nil(s.pm.GetService(args.ARN))
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
