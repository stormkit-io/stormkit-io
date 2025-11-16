package integrations

import (
	"bufio"
	"context"
	"errors"
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

	"github.com/goccy/go-yaml"
	"github.com/stormkit-io/stormkit-io/src/lib/html"
	"github.com/stormkit-io/stormkit-io/src/lib/shttp"
	"github.com/stormkit-io/stormkit-io/src/lib/shutdown"
	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/file"
	"github.com/stormkit-io/stormkit-io/src/lib/utils/sys"
	"go.uber.org/zap"
)

type ServerConfig struct {
	WorkDir string   `yaml:"workdir"`
	Setup   []string `yaml:"setup"`
	Stop    []string `yaml:"stop"`
}

type ProcessManager struct {
	mux           sync.Mutex
	services      map[string]*Service
	waitGroup     map[string]*sync.WaitGroup
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
	serverConfig *ServerConfig
	filePointer  int64
	port         int
	isCustomPort bool // Whether the service is using a custom port from environment variables
	maxIdle      int  // The max idle time in minutes
	killed       bool // Whether the service has been killed
	started      bool // Whether the service has been started
	isSettingUp  bool // Whether the service is currently setting up (running setup script)
}

func (s *Service) Pid() int {
	if s.cmd != nil && s.cmd.Process != nil {
		return s.cmd.Process.Pid
	}

	return 0
}

func (s *Service) Kill() {
	if s.cmd == nil || s.killed {
		slog.Debug(slog.LogOpts{
			Msg:     "service is already killed or not started yet",
			Level:   slog.DL2,
			Payload: []zap.Field{zap.String("arn", s.arn)},
		})
		return
	}

	if s.serverConfig != nil && s.serverConfig.Stop != nil {
		for _, script := range s.serverConfig.Stop {
			slog.Debug(slog.LogOpts{
				Msg:     "running stop script for service",
				Level:   slog.DL2,
				Payload: []zap.Field{zap.String("arn", s.arn)},
			})

			cmd := sys.Command(s.ctx, sys.CommandOpts{
				String: script,
				Dir:    s.cmd.Dir,
				Env:    s.cmd.Env,
				Stdout: s.cmd.Stdout,
				Stderr: s.cmd.Stderr,
			})

			if err := cmd.Run(); err != nil {
				slog.Errorf("error while running stop script: %s", err.Error())
			}
		}

		if s.cmd != nil && !utils.IsPortInUse(s.port) {
			s.cmd.Process = nil
		}

		slog.Debug(slog.LogOpts{
			Msg:     "finished running stop script for service",
			Level:   slog.DL2,
			Payload: []zap.Field{zap.String("arn", s.arn)},
		})
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

	s.pm.mux.Lock()
	delete(s.pm.services, s.arn)
	delete(s.pm.waitGroup, s.arn)
	delete(s.pm.customPortMap, s.port)
	s.pm.mux.Unlock()

	s.killed = true
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
		waitGroup:     map[string]*sync.WaitGroup{},
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

func (pm *ProcessManager) hasSetupScript(workDir string) bool {
	return file.Exists(path.Join(workDir, "stormkit.server.yml"))
}

type RunSetupScriptArgs struct {
	InvokeArgs *InvokeArgs
	WorkDir    string
	Vars       []string
	LogFile    *os.File
	Config     *ServerConfig
}

// runSetupScript runs the setup script if it exists in the given work directory.
func (pm *ProcessManager) runSetupScript(ctx context.Context, args RunSetupScriptArgs) error {
	for _, script := range args.Config.Setup {
		slog.Debug(slog.LogOpts{
			Msg:     "running setup script",
			Level:   slog.DL2,
			Payload: []zap.Field{zap.String("script", script)},
		})

		pm.QueueLog(args.InvokeArgs, script)

		cmd := sys.Command(ctx, sys.CommandOpts{
			Env:    args.Vars,
			Dir:    args.WorkDir,
			Stdout: args.LogFile,
			Stderr: args.LogFile,
			String: os.Expand(script, func(name string) string {
				return args.InvokeArgs.EnvVariables[name]
			}),
		})

		if err := cmd.Run(); err != nil {
			slog.Errorf("error while running setup script %s: %s", script, err.Error())
			return err
		}

		slog.Debug(slog.LogOpts{
			Msg:     "setup script finished successfully",
			Level:   slog.DL2,
			Payload: []zap.Field{zap.String("script", script)},
		})
	}

	return nil
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

		pm.mux.Lock()
		prev := pm.customPortMap[service.port]
		pm.mux.Unlock()

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

	lockFile := path.Join(workDir, "stormkit.lock")

	if pm.hasSetupScript(workDir) {
		yml, err := os.ReadFile(path.Join(workDir, "stormkit.server.yml"))

		if err != nil {
			return nil, err
		}

		config := ServerConfig{}

		if err := yaml.Unmarshal(yml, &config); err != nil {
			return nil, err
		}

		service.serverConfig = &config
		service.isSettingUp = true

		if config.WorkDir != "" {
			workDir = path.Join(workDir, config.WorkDir)

			// Make sure directory exists
			if err := os.MkdirAll(workDir, 0776); err != nil {
				return nil, fmt.Errorf("cannot create work directory: %s", err.Error())
			}
		}
	}

	go func(s *Service) {
		if s.serverConfig != nil && !file.Exists(lockFile) {
			// We need a different context because the request context is canceled as soon as the response is sent.
			// I guess the optimal way here would be if the Kill method is called, we would cancel the context.
			err := pm.runSetupScript(context.TODO(), RunSetupScriptArgs{
				InvokeArgs: args,
				Vars:       vars,
				WorkDir:    workDir,
				LogFile:    s.file,
				Config:     s.serverConfig,
			})

			s.isSettingUp = false

			slog.Debug(slog.LogOpts{
				Msg:     "finished running setup script for service",
				Level:   slog.DL2,
				Payload: []zap.Field{zap.String("arn", s.arn)},
			})

			if err != nil {
				pm.QueueLog(args, err.Error())
				return
			}

			if err := os.WriteFile(lockFile, []byte(""), 0755); err != nil {
				slog.Errorf("error while removing stormkit.server.yml file: %s", err.Error())
			}
		}

		s.isSettingUp = false

		service.cmd = sys.Command(ctx, sys.CommandOpts{
			String:      args.Command,
			Dir:         workDir,
			Env:         vars,
			Stdout:      outfile,
			Stderr:      outfile,
			SysProcAttr: getSysProcAttr(),
		}).Cmd()

		if err := s.cmd.Start(); err != nil {
			pm.QueueLog(args, err.Error())
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
		slog.Debug(slog.LogOpts{
			Msg:   "service not found, starting a new one",
			Level: slog.DL2,
		})

		var err error

		service, err = pm.Start(context.TODO(), &args, workDir)

		if err != nil {
			return nil, err
		}

		pm.addService(service, args.ARN)
	}

	if service != nil && service.isSettingUp {
		return &InvokeResult{
			StatusCode: http.StatusOK,
			Headers: http.Header{
				"Retry-After":  []string{"5"},
				"Content-Type": []string{"text/html"},
			},
			Body: html.MustRender(html.RenderArgs{
				PageTitle:   "Stormkit - Setting up service",
				PageHead:    `<meta http-equiv="refresh" content="5">`,
				PageContent: `<h1 class="text-center">Service is currently being set up, please try again later.</h1>`,
			}),
		}, nil
	}

	// Wait for service to start with a timeout
	if service != nil && !service.started {
		timeout := time.After(10 * time.Second)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-timeout:
				goto serviceNotReadyYet
			case <-ticker.C:
				if service.started {
					goto serviceNotReadyYet
				}
			}
		}
	}

serviceNotReadyYet:
	if service != nil && !service.started {
		slog.Debug(slog.LogOpts{
			Msg:     "service is not ready yet",
			Level:   slog.DL2,
			Payload: []zap.Field{zap.String("arn", args.ARN)},
		})

		return &InvokeResult{
			StatusCode: http.StatusOK,
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

	return pm.requestWithRetry(args, pm.GetService(args.ARN))
}

func (pm *ProcessManager) KillAll() error {
	pm.mux.Lock()
	services := pm.services
	pm.mux.Unlock()

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
	pm.mux.Lock()
	defer pm.mux.Unlock()

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

// addService adds a service with the given ARN.
func (pm *ProcessManager) addService(service *Service, ARN string) {
	pm.mux.Lock()
	defer pm.mux.Unlock()
	pm.services[ARN] = service

	if service.isCustomPort {
		pm.customPortMap[service.port] = service
	}
}

// Request the given URL within the allowed timeout. We're trying every 250ms to fetch the
// result from the server until allowed timeout is exhausted.
func (pm *ProcessManager) requestWithRetry(args InvokeArgs, service *Service) (*InvokeResult, error) {
	timeout := time.After(30 * time.Second)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return nil, errors.New("server is not up and running within allowed timeout")
		case <-ticker.C:
			res, err := pm.request(args, service)

			if err != nil {
				continue
			}

			return res, nil
		}
	}
}

// Request the given resource from the spawned server.
func (pm *ProcessManager) request(args InvokeArgs, service *Service) (*InvokeResult, error) {
	target := *args.URL
	target.Scheme = "http"
	target.Host = fmt.Sprintf("localhost:%d", service.port)

	res := shttp.Proxy(&shttp.RequestContext{
		Request: &http.Request{
			Header: args.Headers,
			Method: args.Method,
			URL:    args.URL,
			Body:   args.Body,
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
