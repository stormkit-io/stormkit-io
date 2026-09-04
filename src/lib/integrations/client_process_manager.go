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

	// startLocks holds one *sync.Mutex per ARN, serializing the
	// lookup-start-register sequence in Invoke so that two concurrent
	// requests for a cold ARN cannot both spawn a server.
	//
	// Entries are intentionally never removed, for the same reason as
	// ZipManager.locks: dropping a lock another goroutine already holds a
	// pointer to would hand the next caller a second mutex for the same ARN,
	// and the mutual exclusion would be an illusion.
	startLocks sync.Map // arn (string) -> *sync.Mutex

	// spawned holds every service whose process may still be alive,
	// including ones no longer reachable through services. It is what lets
	// ReapOrphans find a server that outlived its registration.
	spawnedMux sync.Mutex
	spawned    map[*Service]struct{}
}

type Service struct {
	arn         string
	ctx         context.Context
	pm          *ProcessManager
	timer       *time.Timer
	file        *os.File
	args        *InvokeArgs
	filePointer int64
	port        int

	isCustomPort bool // Whether the service is using a custom port from environment variables
	maxIdle      int  // The max idle time in minutes

	// mu guards every field below it. The process is spawned from a
	// goroutine while Kill can be called from the idle timer, from a
	// replacement start, or from shutdown, so cmd must never be read or
	// written without it.
	mu           sync.Mutex
	cmd          *exec.Cmd
	spawnedAt    time.Time // When the process was started; zero until then
	killedAt     time.Time // When the process was signalled; zero until then
	killed       bool      // Whether the service has been killed
	started      bool      // Whether the service has been started
	reaped       bool      // Whether cmd.Wait has returned for this process
	isNixWrapped bool      // Whether the server command is wrapped with nix develop
}

func (s *Service) Pid() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.pid()
}

// pid returns the process id, or 0 when the process is not running yet.
// Callers must hold mu.
func (s *Service) pid() int {
	if s.cmd != nil && s.cmd.Process != nil {
		return s.cmd.Process.Pid
	}

	return 0
}

// IsStarted reports whether the process has been spawned successfully.
func (s *Service) IsStarted() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.started
}

func (s *Service) isKilled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.killed
}

func (s *Service) nixWrapped() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.isNixWrapped
}

func (s *Service) Kill() {
	s.pm.unregister(s)

	// Deliberately still tracked as spawned: the process has only been asked
	// to stop. It is forgotten once cmd.Wait confirms it is gone, so that a
	// server which ignores SIGTERM stays visible to ReapOrphans, which
	// escalates.
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.killed {
		slog.Debug(slog.LogOpts{
			Msg:     "service is already killed",
			Level:   slog.DL2,
			Payload: []zap.Field{zap.String("arn", s.arn)},
		})

		return
	}

	// Set before the nil check below: the goroutine that spawns the process
	// reads this under the same lock and gives up when it is set, so a Kill
	// that lands before the spawn cannot leave a process nobody owns.
	s.killed = true
	s.killedAt = time.Now()

	// Nothing was ever spawned, so nothing will be: drop it now, since no
	// cmd.Wait will run to do it later.
	if s.cmd == nil {
		s.pm.forget(s)
	}

	// The process either has not been spawned yet - in which case the start
	// goroutine will see killed and never spawn it - or it has already been
	// reaped by cmd.Wait, and signalling its group now could hit an unrelated
	// process that has since been given the same id.
	if s.cmd == nil || s.cmd.Process == nil || s.reaped {
		slog.Debug(slog.LogOpts{
			Msg:     "service has no running process to kill",
			Level:   slog.DL2,
			Payload: []zap.Field{zap.String("arn", s.arn)},
		})
	} else {
		slog.Debug(slog.LogOpts{
			Msg:   "killing service",
			Level: slog.DL2,
			Payload: []zap.Field{
				zap.String("arn", s.arn),
				zap.Int("pid", s.pid()),
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

// orphanGracePeriod is how long a spawned service is left alone before
// ReapOrphans may consider it. It has to outlast the gap between spawning a
// service and registering it in Invoke, or the reaper would kill services that
// are merely still starting.
const orphanGracePeriod = 2 * time.Minute

// orphanSweepInterval is how often the reaper runs.
const orphanSweepInterval = time.Minute

func NewProcessManager() *ProcessManager {
	pm := &ProcessManager{
		services:      map[string]*Service{},
		customPortMap: map[int]*Service{},
		spawned:       map[*Service]struct{}{},
	}

	shutdown.Subscribe(pm.KillAll)

	go pm.reapOrphansPeriodically()

	return pm
}

// unregister removes a service from the lookup maps, but only where the maps
// still point at that exact service. Deleting by ARN and port alone would
// unregister a live replacement: killing an orphan would evict the healthy
// service that took its place, which the next sweep would then reap for being
// unregistered.
func (pm *ProcessManager) unregister(s *Service) {
	pm.servicesMux.Lock()

	if pm.services[s.arn] == s {
		delete(pm.services, s.arn)
	}

	pm.servicesMux.Unlock()

	pm.portMux.Lock()

	if pm.customPortMap[s.port] == s {
		delete(pm.customPortMap, s.port)
	}

	pm.portMux.Unlock()
}

// track records a service as owning a live process.
func (pm *ProcessManager) track(s *Service) {
	pm.spawnedMux.Lock()
	defer pm.spawnedMux.Unlock()

	pm.spawned[s] = struct{}{}
}

// forget drops a service from the spawned set, once its process is known to be
// gone or has been signalled.
func (pm *ProcessManager) forget(s *Service) {
	pm.spawnedMux.Lock()
	defer pm.spawnedMux.Unlock()

	delete(pm.spawned, s)
}

func (pm *ProcessManager) reapOrphansPeriodically() {
	for range time.Tick(orphanSweepInterval) {
		pm.ReapOrphans(orphanGracePeriod)
	}
}

// ReapOrphans kills every service that still has a running process but is no
// longer the service registered for its ARN, and so can never be reached by a
// request or by the idle timer again. Services spawned within grace are left
// alone, since Invoke registers a service shortly after starting it.
//
// It also escalates: a service that was asked to stop more than grace ago and
// whose process has still not been reaped gets a SIGKILL. Without that, a
// server that ignores SIGTERM keeps its port for as long as the host lives,
// which is the failure this whole mechanism exists to prevent.
//
// Nothing should end up here: it is the backstop for a service escaping its
// registration, and for a deployment whose folder is deleted underneath a
// server that is still running. Each reap is logged as an error because it
// means one of those happened.
func (pm *ProcessManager) ReapOrphans(grace time.Duration) int {
	pm.spawnedMux.Lock()
	candidates := make([]*Service, 0, len(pm.spawned))

	for s := range pm.spawned {
		candidates = append(candidates, s)
	}

	pm.spawnedMux.Unlock()

	reaped := 0

	for _, s := range candidates {
		s.mu.Lock()
		spawning := s.spawnedAt.IsZero() || time.Since(s.spawnedAt) < grace
		killed := s.killed
		lingering := killed && !s.reaped && !s.killedAt.IsZero() && time.Since(s.killedAt) >= grace
		pid := s.pid()

		if lingering {
			slog.Errorf("service ignored the termination signal, force killing, arn: %s, pid: %d", s.arn, pid)
			s.forceKillProcessGroup()
		}

		s.mu.Unlock()

		if lingering {
			reaped++
			continue
		}

		if spawning || killed {
			continue
		}

		pm.servicesMux.Lock()
		orphan := pm.services[s.arn] != s
		pm.servicesMux.Unlock()

		if !orphan {
			continue
		}

		slog.Errorf("reaping orphaned service, arn: %s, pid: %d", s.arn, pid)

		s.Kill()

		reaped++
	}

	return reaped
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
		isNixWrapped := hasNixFlake(workDir)

		// Creating the command and starting it happen under the lock, so that
		// a Kill arriving in between cannot decide there is no process to
		// signal moments before one exists. Whichever side gets the lock
		// first wins: Kill sets killed and the spawn is abandoned, or the
		// process is spawned and Kill has a pid to signal.
		s.mu.Lock()

		if s.killed {
			s.mu.Unlock()
			return
		}

		s.isNixWrapped = isNixWrapped
		s.cmd = sys.Command(ctx, sys.CommandOpts{
			String:      pm.BuildServerCommand(args.Command, workDir),
			Dir:         workDir,
			Env:         vars,
			Stdout:      outfile,
			Stderr:      outfile,
			SysProcAttr: getSysProcAttr(),
		}).Cmd()

		err := s.cmd.Start()
		cmd := s.cmd

		if err == nil {
			s.started = true
			s.spawnedAt = time.Now()
			pm.track(s)
		}

		s.mu.Unlock()

		if err != nil {
			pm.QueueLog(args, err.Error())
			slog.Errorf("error while starting service, arn: %s, err: %s", s.arn, err.Error())

			// Evict the service so that the next Invoke re-attempts the start.
			// Without this the ARN stays registered with started=false and every
			// subsequent request gets the warm-up interstitial forever.
			s.Kill()

			return
		}

		slog.Debug(slog.LogOpts{
			Msg:   "service started",
			Level: slog.DL2,
			Payload: []zap.Field{
				zap.String("arn", s.arn),
				zap.Int("port", s.port),
			},
		})

		// cmd is read without the lock from here on: it is never reassigned
		// once the process is spawned, and Wait must not block Kill.
		waitErr := cmd.Wait()

		s.mu.Lock()
		s.reaped = true
		killed := s.killed
		s.mu.Unlock()

		pm.forget(s)

		// Ignore the error here: it could be related to spawning background processes and
		// there is no easy way to understand if the cmd is a background process or not
		switch {
		case waitErr == nil:
			slog.Debug(slog.LogOpts{
				Msg:   "service finished successfully",
				Level: slog.DL2,
				Payload: []zap.Field{
					zap.String("arn", s.arn),
					zap.Int("pid", service.Pid()),
				},
			})
		case killed:
			// We sent the signal ourselves - the service went idle, was
			// replaced, or the host is shutting down. A server that does not
			// trap SIGTERM reports this as "signal: terminated", which is a
			// normal recycle and not something to page anyone about.
			slog.Debug(slog.LogOpts{
				Msg:   "service terminated by the process manager",
				Level: slog.DL2,
				Payload: []zap.Field{
					zap.String("arn", s.arn),
					zap.String("err", waitErr.Error()),
				},
			})
		default:
			slog.Errorf("service exited unexpectedly, arn: %s, err: %s", s.arn, waitErr.Error())
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

// startLock returns the mutex serializing starts for the given ARN.
func (pm *ProcessManager) startLock(arn string) *sync.Mutex {
	if v, ok := pm.startLocks.Load(arn); ok {
		return v.(*sync.Mutex)
	}

	actual, _ := pm.startLocks.LoadOrStore(arn, &sync.Mutex{})

	return actual.(*sync.Mutex)
}

// startOnce returns the service registered for the ARN, starting one if there
// is none.
//
// The lookup, the start and the registration are serialized per ARN. As three
// separate steps they let two concurrent requests for a cold ARN each spawn a
// server: the loser was dropped from the map and kept running with nothing
// tracking it, holding its port until the host restarted. The lock is released
// before the request is proxied, so it only serializes starting a service, not
// serving it.
func (pm *ProcessManager) startOnce(args *InvokeArgs, workDir string) (*Service, error) {
	lock := pm.startLock(args.ARN)
	lock.Lock()
	defer lock.Unlock()

	pm.servicesMux.Lock()
	service := pm.services[args.ARN]
	pm.servicesMux.Unlock()

	if service != nil {
		return service, nil
	}

	slog.Debug(slog.LogOpts{
		Msg:     "service not found, starting a new one",
		Level:   slog.DL2,
		Payload: []zap.Field{zap.String("arn", args.ARN)},
	})

	service, err := pm.Start(context.TODO(), args, workDir)

	if err != nil {
		return nil, err
	}

	pm.servicesMux.Lock()
	existing := pm.services[args.ARN]
	pm.services[args.ARN] = service
	pm.servicesMux.Unlock()

	// The start goroutine kills the service itself when the command fails to
	// spawn, and it can do so before we get here. Re-check after registering:
	// a Kill that ran first leaves nothing to unregister itself, and the entry
	// would sit in the map handing out warm-up pages until the request after
	// next evicted it.
	if service.isKilled() {
		pm.unregister(service)
	}

	// Unreachable while the ARN lock is held, kept so that a future caller
	// that registers a service without it cannot silently leak the one it
	// replaces.
	if existing != nil {
		existing.Kill()
	}

	return service, nil
}

// Invoke starts a new service if it doesn't exist yet, or waits for the existing one to be ready.
// It then sends the request to the service and returns the result.
// path is the path to the directory where the service is running.
func (pm *ProcessManager) Invoke(args InvokeArgs, workDir string) (*InvokeResult, error) {
	service := pm.GetService(args.ARN)

	if service != nil && service.isKilled() {
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
		var err error

		service, err = pm.startOnce(&args, workDir)

		if err != nil {
			return nil, err
		}

		if service.isCustomPort {
			pm.portMux.Lock()
			pm.customPortMap[service.port] = service
			pm.portMux.Unlock()
		}
	}

	if service != nil && !service.IsStarted() {
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

	if service != nil && service.nixWrapped() && !utils.IsPortInUse(service.port) {
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

// KillAll terminates every service this manager has spawned, registered or
// not, so that a shutdown does not leave servers behind.
func (pm *ProcessManager) KillAll() error {
	// Snapshot into a slice: Kill deletes from both maps, and ranging over
	// the live map while it does would be a concurrent iteration and write.
	pm.servicesMux.Lock()
	services := make([]*Service, 0, len(pm.services))
	seen := make(map[*Service]struct{}, len(pm.services))

	for _, service := range pm.services {
		services = append(services, service)
		seen[service] = struct{}{}
	}

	pm.servicesMux.Unlock()

	pm.spawnedMux.Lock()

	for service := range pm.spawned {
		if _, ok := seen[service]; !ok {
			services = append(services, service)
		}
	}

	pm.spawnedMux.Unlock()

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
		Payload: []zap.Field{zap.Int("count", len(services))},
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

		// Kill stops the timer under the same lock.
		service.mu.Lock()
		defer service.mu.Unlock()

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
	// The service can be killed between the readiness check in Invoke and
	// this call - by the idle timer, by a replacement, or by the reaper - in
	// which case there is nothing left to forward the request to.
	if service == nil {
		return nil, fmt.Errorf("service %q is no longer running", args.ARN)
	}

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
