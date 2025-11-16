//go:build windows

package integrations

import (
	"syscall"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
)

func init() {
	slog.Debug(slog.LogOpts{
		Msg:   "using windows client process manager",
		Level: slog.DL1,
	})
}

// killProcessGroup kills the process associated with the service.
// On Windows, process groups work differently. The process will be killed
// via the standard Process.Kill() method in the main Kill() function.
func (s *Service) killProcessGroup() {
	if s.cmd.Process == nil {
		return
	}

	// On Windows, killing the parent process should terminate child processes
	// if they were created with CREATE_NEW_PROCESS_GROUP flag.
	// The syscall.SysProcAttr{Setpgid: true} used in Start() doesn't apply on Windows.
	if err := s.cmd.Process.Kill(); err != nil {
		slog.Errorf("error while killing process on Windows: %s", err.Error())
	}
}

// getSysProcAttr returns the platform-specific process attributes.
// On Windows, this creates a new process group for better child process management.
func getSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}
