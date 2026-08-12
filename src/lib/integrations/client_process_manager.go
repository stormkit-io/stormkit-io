package integrations

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/stormkit-io/stormkit-io/src/ce/api/admin"
	"github.com/stormkit-io/stormkit-io/src/lib/html"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shutdown"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"go.uber.org/zap"
)

type ProcessManager struct {
	servicesMux   sync.Mutex
	portMux       sync.Mutex
	services      map[string]*Service
	customPortMap map[int]*Service
}

type Service struct {
	arn          string
	ctx          context.Context
	pm           *ProcessManager
	cmd          *exec.Cmd
	timer        *time.Timer
	file         *os.File
	args         *InvokeArgs
	filePointer  int64
	port         int
	isCustomPort bool // Whether the service is using a custom port from environment variables
	maxIdle      int  // The max idle time in minutes
	killed       bool // Whether the service has been killed
	started      bool // Whether the service has been started
	isNixWrapped bool // Whether the server command is wrapped with nix develop
}

func (s *Service) Pid() int {
	if s.cmd != nil && s.cmd.Process != nil {
		return s.cmd.Process.Pid
	}

	return 0
}

func (s *Service) Kill() {
	s.pm.servicesMux.Lock()
	delete(s.pm.services, s.arn)
	s.pm.servicesMux.Unlock()

	s.pm.portMux.Lock()
	delete(s.pm.customPortMap, s.port)
	s.pm.portMux.Unlock()

	if s.killed {
		slog.Debug(slog.LogOpts{
			Msg:     "service is already killed",
			Level:   slog.DL2,
			Payload: []zap.Field{zap.String("arn", s.arn)},
		})

		return
	}

	s.killed = true

	// If this is happening,
	if s.cmd == nil {
		slog.Debug(slog.LogOpts{
			Msg:     "service not started yet",
			Level:   slog.DL2,
			Payload: []zap.Field{zap.String("arn", s.arn)},
		})

		return
	}

	if s.cmd.Process != nil {
		slog.Debug(slog.LogOpts{
			Msg:   "killing service",
			Level: slog.DL2,
			Payload: []zap.Field{
				zap.String("arn", s.arn),
				zap.Int("pid", s.cmd.Process.Pid),
			},
		})

		s.killProcessGroup()
	}

	if s.timer != nil {
		s.timer.Stop()
	}

	if s.file != nil {
		if err := s.file.Close(); err != nil {
			slog.Errorf("error while closing log file: %s", err.Error())
		}

		if err := os.Remove(s.file.Name()); err != nil {
			slog.Errorf("error while removing log file: %s", err.Error())
		}
	}
}

func (s *Service) processLogs(input io.ReadSeeker, start int64) error {
	if _, err := input.Seek(start, 0); err != nil {
		return err
	}

	scanner := bufio.NewScanner(input)

	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		advance, token, err = bufio.ScanLines(data, atEOF)
		start += int64(advance)
		return
	})

	logs := []string{}

	for scanner.Scan() {
		s.filePointer = start
		logs = append(logs, scanner.Text())
	}

	if len(logs) > 0 {
		s.pm.QueueLog(s.args, strings.Join(logs, "\n"))
	}

	return scanner.Err()
}

func (s *Service) logger() {
	input, err := os.Open(s.file.Name())

	if err != nil {
		slog.Errorf("error while opening file: %s", err.Error())
		return
	}

	for {
		select {
		case <-s.ctx.Done():
			slog.Debug(slog.LogOpts{
				Msg:     "context canceled for service, stopping logger",
				Level:   slog.DL2,
				Payload: []zap.Field{zap.String("arn", s.arn)},
			})

			// canceled
			if input != nil {
				input.Close()
			}
			return
		default:
			if err := s.processLogs(input, s.filePointer); err != nil {
				slog.Errorf("error while processing logs: %s", err.Error())
			}

			time.Sleep(1 * time.Second) // Simulate work with a sleep
		}
	}
}

func NewProcessManager() *ProcessManager {
	pm := &ProcessManager{
		services:      map[string]*Service{},
		customPortMap: map[int]*Service{},
	}

	shutdown.Subscribe(pm.KillAll)

	return pm
}

func (pm *ProcessManager) QueueLog(args *InvokeArgs, data string) {
	if args.QueueLog == nil {
		return
	}

	args.QueueLog(&Log{
		Timestamp: time.Now().UTC().Unix(),
		Message:   data,
	})
}

func hasNixFlake(workDir string) bool {
	if !admin.MustConfig().IsAutoInstallEnabled() {
		return false
	}

	return file.Exists(path.Join(workDir, "flake.nix"))
}

// BuildServerCommand wraps command with nix develop when flake.nix is present in workDir,
// so that all nix-provided libraries are available at runtime.
func (pm *ProcessManager) BuildServerCommand(command, workDir string) string {
	if hasNixFlake(workDir) {
		// Quote the command so it is passed as a single argument to sh -c.
		// Without quoting, shlex splits multi-word commands (e.g. "node build")
		// into separate nix --command args, causing sh to run only the first word.
		quoted := "'" + strings.ReplaceAll(command, "'", `'\''`) + "'"
		return `nix --extra-experimental-features "nix-command flakes" develop --command sh -c ` + quoted
	}

	return command
}

// Start starts a new service with the given arguments and working directory.
// It creates a new command with the given command string, working directory, and environment variables.
// It also creates a log file in the temporary directory to capture the output of the command.
// It returns a Service object that can be used to interact with the started service.
// If the service has a setup script, it runs it before starting the command.
// It also finds an available port for the service to listen on.
// If the command fails to start, it returns an error.
// The service is automatically killed when the context is canceled or when the command finishes.
func (pm *ProcessManager) Start(ctx context.Context, args *InvokeArgs, workDir string) (*Service, error) {
	outfile, err := os.Create(path.Join(os.TempDir(), fmt.Sprintf("logs-d-%s.txt", args.DeploymentID.String())))

	if err != nil {
		slog.Errorf("cannot open log file: %s", err.Error())
		return nil, err
	}

	if !args.IsPublished && args.EnvVariables["PORT"] != "" {
		return nil, fmt.Errorf("custom ports are only available for published deployments, please remove the PORT environment variable to use dynamic ports")
	}

	port, err := findAvailablePort(args)

	if err != nil {
		return nil, fmt.Errorf("cannot find an available port: %s", err.Error())
	}

	vars := prepareEnvironmentVariables(args, port)
	maxIdleInMinutes := 10

	if maxIdle, ok := args.EnvVariables["STORMKIT_MAX_IDLE"]; ok {
		maxIdleInMinutes = utils.StringToInt(maxIdle)
	}

	service := &Service{
		port:         port,
		pm:           pm,
		arn:          args.ARN,
		file:         outfile,
		ctx:          ctx,
		args:         args,
		maxIdle:      maxIdleInMinutes,
		isCustomPort: args.EnvVariables["PORT"] != "",
	}

	if service.isCustomPort {
		slog.Debug(slog.LogOpts{
			Msg:     "service is using custom port, checking for previous service on the same port",
			Level:   slog.DL2,
			Payload: []zap.Field{zap.String("arn", service.arn)},
		})

		pm.portMux.Lock()
		prev := pm.customPortMap[service.port]
		pm.portMux.Unlock()

		// Kill the previous service on the same port if it exists.
		if prev != nil && prev.arn != service.arn {
			slog.Debug(slog.LogOpts{
				Msg:   "found previous service on the same port, killing it",
				Level: slog.DL2,
				Payload: []zap.Field{
					zap.String("previous_arn", prev.arn),
					zap.String("new_arn", service.arn),
					zap.Int("port", service.port),
				},
			})

			prev.Kill()
		}
	}

	go func(s *Service) {
		s.isNixWrapped = hasNixFlake(workDir)

		if s.killed {
			return
		}

		s.cmd = sys.Command(ctx, sys.CommandOpts{
			String:      pm.BuildServerCommand(args.Command, workDir),
			Dir:         workDir,
			Env:         vars,
			Stdout:      outfile,
			Stderr:      outfile,
			SysProcAttr: getSysProcAttr(),
		}).Cmd()

		if err := s.cmd.Start(); err != nil {
			pm.QueueLog(args, err.Error())
			slog.Errorf("error while starting service, arn: %s, err: %s", s.arn, err.Error())

			// Evict the service so that the next Invoke re-attempts the start.
			// Without this the ARN stays registered with started=false and every
			// subsequent request gets the warm-up interstitial forever.
			s.Kill()

			return
		}

		s.started = true
		slog.Debug(slog.LogOpts{
			Msg:   "service started",
			Level: slog.DL2,
			Payload: []zap.Field{
				zap.String("arn", s.arn),
				zap.Int("port", s.port),
			},
		})

		// Ignore error here: it could be related to spawning background processes and
		// there is no easy way to understand if the cmd is a background process or not
		if err := s.cmd.Wait(); err != nil {
			slog.Errorf("error while waiting for service to finish, arn: %s, err: %s", s.arn, err.Error())
		} else {
			slog.Debug(slog.LogOpts{
				Msg:   "service finished successfully",
				Level: slog.DL2,
				Payload: []zap.Field{
					zap.String("arn", s.arn),
					zap.Int("pid", service.Pid()),
				},
			})
		}

		// Check if the port is still in use after the service has finished and kill service
		// if the port is not in use anymore.
		if !utils.IsPortInUse(s.port) {
			slog.Debug(slog.LogOpts{
				Msg:   "service finished and port is not in use anymore",
				Level: slog.DL2,
				Payload: []zap.Field{
					zap.String("arn", s.arn),
					zap.Int("port", s.port),
				},
			})
			s.Kill()
		}
	}(service)

	if args.CaptureLogs {
		go service.logger()
	}

	return service, nil
}

// Invoke starts a new service if it doesn't exist yet, or waits for the existing one to be ready.
// It then sends the request to the service and returns the result.
// path is the path to the directory where the service is running.
func (pm *ProcessManager) Invoke(args InvokeArgs, workDir string) (*InvokeResult, error) {
	service := pm.GetService(args.ARN)

	if service != nil && service.killed {
		slog.Debug(slog.LogOpts{
			Msg:     "service was previously killed, removing from the list",
			Level:   slog.DL2,
			Payload: []zap.Field{zap.String("arn", args.ARN)},
		})

		service.Kill()
		service = nil
	}

	if !args.IsPublished && args.EnvVariables["PORT"] != "" {
		return &InvokeResult{
			StatusCode: http.StatusBadRequest,
			Headers: http.Header{
				"Content-Type": []string{"text/html"},
			},
			Body: html.MustRender(html.RenderArgs{
				PageTitle: "Stormkit - Invalid request",
				PageContent: `
					<div class="container text-center">
						<h2>Custom ports are only available for published deployments</h2>
						<h3>Please remove the PORT environment variable to use dynamic ports,<br />or access this service via the published URL.</h3>
					</div>
				`,
			}),
		}, nil
	}

	slog.Debug(slog.LogOpts{
		Msg:   "invoking service",
		Level: slog.DL2,
		Payload: []zap.Field{
			zap.String("arn", args.ARN),
			zap.String("host", args.HostName),
		},
	})

	if service == nil {
		pm.servicesMux.Lock()
		service = pm.services[args.ARN]
		pm.servicesMux.Unlock()

		if service == nil {
			slog.Debug(slog.LogOpts{
				Msg:   "service not found, starting a new one",
				Level: slog.DL2,
			})

			var err error

			service, err = pm.Start(context.TODO(), &args, workDir)

			if err != nil {
				return nil, err
			}

			pm.servicesMux.Lock()
			existing := pm.services[args.ARN]
			pm.services[args.ARN] = service
			pm.servicesMux.Unlock()

			if existing != nil {
				existing.Kill()
			}
		}

		if service.isCustomPort {
			pm.portMux.Lock()
			pm.customPortMap[service.port] = service
			pm.portMux.Unlock()
		}
	}

	if service != nil && !service.started {
		slog.Debug(slog.LogOpts{
			Msg:     "service is not ready yet",
			Level:   slog.DL2,
			Payload: []zap.Field{zap.String("arn", args.ARN)},
		})

		return &InvokeResult{
			StatusCode: http.StatusServiceUnavailable,
			Headers: http.Header{
				"Retry-After":  []string{"1"},
				"Content-Type": []string{"text/html"},
			},
			Body: html.MustRender(html.RenderArgs{
				PageTitle:   "Stormkit - Setting up service",
				PageHead:    `<meta http-equiv="refresh" content="1">`,
				PageContent: `<h1 class="text-center">Service not yet started, retry in a bit.</h1>`,
			}),
		}, nil
	}

	if service != nil && service.isNixWrapped && !utils.IsPortInUse(service.port) {
		slog.Debug(slog.LogOpts{
			Msg:     "nix is installing dependencies",
			Level:   slog.DL2,
			Payload: []zap.Field{zap.String("arn", args.ARN)},
		})

		return &InvokeResult{
			StatusCode: http.StatusServiceUnavailable,
			Headers: http.Header{
				"Retry-After":  []string{"5"},
				"Content-Type": []string{"text/html"},
			},
			Body: html.MustRender(html.RenderArgs{
				PageTitle:   "Stormkit - Installing dependencies",
				PageHead:    `<meta http-equiv="refresh" content="5">`,
				PageContent: `<h1 class="text-center">Installing dependencies, please wait...</h1>`,
			}),
		}, nil
	}

	return pm.requestWithRetry(args, pm.GetService(args.ARN))
}

func (pm *ProcessManager) KillAll() error {
	pm.servicesMux.Lock()
	services := pm.services
	pm.servicesMux.Unlock()

	slog.Debug(slog.LogOpts{
		Msg:     "killing all services",
		Level:   slog.DL2,
		Payload: []zap.Field{zap.Int("count", len(services))},
	})

	for _, service := range services {
		service.Kill()
	}

	slog.Debug(slog.LogOpts{
		Msg:     "all services killed",
		Level:   slog.DL2,
		Payload: []zap.Field{zap.Int("remaining_count", len(services))},
	})

	return nil
}

// GetService returns a service for the given ARN.
func (pm *ProcessManager) GetService(ARN string) *Service {
	pm.servicesMux.Lock()
	defer pm.servicesMux.Unlock()

	service := pm.services[ARN]

	if service == nil {
		return nil
	}

	if service.maxIdle > 0 {
		killAfterInactivity := time.Minute * time.Duration(service.maxIdle)

		if service.timer == nil {
			service.timer = time.AfterFunc(killAfterInactivity, func() {
				slog.Debug(slog.LogOpts{
					Msg:     "service has been idle for too long, killing it",
					Level:   slog.DL2,
					Payload: []zap.Field{zap.String("arn", service.arn)},
				})

				service.Kill()
			})
		} else {
			service.timer.Reset(killAfterInactivity)
		}
	}

	return service
}

// waitForPort polls via TCP dial until the port is accepting connections or the timeout elapses.
func (pm *ProcessManager) waitForPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	addr := fmt.Sprintf("localhost:%d", port)

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 250*time.Millisecond)

		if err == nil {
			conn.Close()
			return nil
		}

		time.Sleep(250 * time.Millisecond)
	}

	return fmt.Errorf("server on port %d did not become available within the allowed timeout", port)
}

// requestWithRetry waits for the service port to accept TCP connections, then forwards
// the request exactly once. Separating port-readiness polling from the HTTP send ensures
// the streaming body is never consumed on a failed connection attempt.
func (pm *ProcessManager) requestWithRetry(args InvokeArgs, service *Service) (*InvokeResult, error) {
	if err := pm.waitForPort(service.port, 30*time.Second); err != nil {
		return nil, err
	}

	return pm.request(args, service)
}

// Request the given resource from the spawned server.
func (pm *ProcessManager) request(args InvokeArgs, service *Service) (*InvokeResult, error) {
	target := *args.URL
	target.Scheme = "http"
	target.Host = fmt.Sprintf("localhost:%d", service.port)

	res := shttp.Proxy(&shttp.RequestContext{
		Request: &http.Request{
			Header:        args.Headers,
			Method:        args.Method,
			URL:           args.URL,
			Body:          args.Body,
			ContentLength: args.ContentLength,
		},
	}, shttp.ProxyArgs{
		Target:          target.String(),
		FollowRedirects: utils.Ptr(false),
	})

	if res.Error != nil {
		return nil, res.Error
	}

	var data []byte

	if res.Data != nil {
		data = res.Data.([]byte)
	}

	// Remove keep-alive header as we're serving http 2 and it's not compatible with it.
	// See: https://developer.mozilla.org/en-US/docs/Web/HTTP/Headers/Keep-Alive
	res.Headers.Del("keep-alive")
	res.Headers.Del("connection")

	return &InvokeResult{
		StatusCode: res.Status,
		Headers:    res.Headers,
		Body:       data,
	}, nil
}

// findAvailablePort tries to find the first available port in the given range.
func findAvailablePort(args *InvokeArgs) (int, error) {
	var port int

	// Allow overwriting the port via environment variables.
	if args.EnvVariables != nil {
		if p := args.EnvVariables["PORT"]; p != "" {
			port = utils.StringToInt(p)
		}

		if port != 0 {
			return port, nil
		}
	}

	listener, err := net.Listen("tcp", ":0")

	if err != nil {
		return 0, err
	}

	if listener != nil {
		port = listener.Addr().(*net.TCPAddr).Port
		listener.Close()
	}

	return port, nil
}

// prepareEnvironmentVariables prepares the environment variables for the service.
func prepareEnvironmentVariables(args *InvokeArgs, port int) []string {
	vars := []string{}

	for k, v := range args.EnvVariables {
		vars = append(vars, fmt.Sprintf("%s=%s", k, v))
	}

	// Include origin in the environment variables if it's missing
	// https://github.com/stormkit-io/app-stormkit-io/issues/589
	if args.EnvVariables["ORIGIN"] == "" {
		vars = append(vars, fmt.Sprintf("ORIGIN=https://%s", args.HostName))
	}

	vars = append(
		vars,
		fmt.Sprintf("PATH=%s", os.Getenv("PATH")),
		fmt.Sprintf("HOME=%s", os.Getenv("HOME")),
		fmt.Sprintf("PORT=%d", port),
	)

	return vars
}
