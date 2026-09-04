//go:build !windows

package integrations

import (
	"syscall"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

func init() {
	slog.Debug(slog.LogOpts{
		Msg:   "using unix client process manager",
		Level: slog.DL1,
	})
}

// killProcessGroup asks the process group associated with the service to
// terminate. On Unix systems, this uses process group IDs (PGID) to reach all
// child processes.
func (s *Service) killProcessGroup() {
	s.signalProcessGroup(syscall.SIGTERM)
}

// forceKillProcessGroup terminates the process group without giving it a
// chance to shut down. It is the escalation for a server that ignores, or
// never finishes handling, the SIGTERM sent by killProcessGroup.
func (s *Service) forceKillProcessGroup() {
	s.signalProcessGroup(syscall.SIGKILL)
}

func (s *Service) signalProcessGroup(sig syscall.Signal) {
	if s.cmd == nil || s.cmd.Process == nil {
		return
	}

	pgid, err := syscall.Getpgid(s.cmd.Process.Pid)

	// Stop children processes
	if err == nil {
		if err := syscall.Kill(-pgid, sig); err != nil {
			slog.Errorf("error while signalling process group: %s", err.Error())
		}
	}
}

// getSysProcAttr returns the platform-specific process attributes.
// On Unix systems, this sets the process group ID for proper child process management.
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
