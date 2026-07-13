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

// killProcessGroup kills the process group associated with the service.
// On Unix systems, this uses process group IDs (PGID) to kill all child processes.
func (s *Service) killProcessGroup() {
	if s.cmd.Process == nil {
		return
	}

	pgid, err := syscall.Getpgid(s.cmd.Process.Pid)

	// Stop children processes
	if err == nil {
		if err := syscall.Kill(-pgid, syscall.SIGTERM); err != nil {
			slog.Errorf("error while killing process group: %s", err.Error())
		}
	}
}

// getSysProcAttr returns the platform-specific process attributes.
// On Unix systems, this sets the process group ID for proper child process management.
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{Setpgid: true}
}
