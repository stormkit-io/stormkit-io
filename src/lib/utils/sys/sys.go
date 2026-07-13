package sys

import (
	"context"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/google/shlex"
)

var _mu sync.Mutex

type CommandInterface interface {
	String() string
	SetOpts(opts CommandOpts) CommandInterface
	CombinedOutput() ([]byte, error)
	Output() ([]byte, error)
	Run() error
	Wait() error
	Start() error
	Cmd() *exec.Cmd
}

type CommandWrapper struct {
	ctx         context.Context
	cmd         string
	args        []string
	env         []string
	dir         string
	stdout      io.Writer
	stderr      io.Writer
	sysProcAttr *syscall.SysProcAttr
	_execCmd    *exec.Cmd
}

func (c CommandWrapper) Cmd() *exec.Cmd {
	_mu.Lock()
	defer _mu.Unlock()

	if c._execCmd == nil {
		cmd := exec.CommandContext(c.ctx, c.cmd, c.args...)
		cmd.Dir = c.dir
		cmd.Env = c.env
		cmd.Stdout = c.stdout
		cmd.Stderr = c.stderr
		cmd.SysProcAttr = c.sysProcAttr
		c._execCmd = cmd
	}

	return c._execCmd
}

// Name returns the name of the wrapped command.
func (c CommandWrapper) Name() string {
	return c.cmd
}

// Args returns the arguments of the wrapped command.
func (c CommandWrapper) Args() []string {
	return c.args
}

// Output executes the wrapped command and returns its standard output as a byte slice.
// It also returns an error if the command execution fails.
func (c CommandWrapper) Output() ([]byte, error) {
	return c.Cmd().Output()
}

// CombinedOutput executes the wrapped command and returns its combined standard output and standard error as a byte slice.
// It also returns an error if the command execution fails.
func (c CommandWrapper) CombinedOutput() ([]byte, error) {
	return c.Cmd().CombinedOutput()
}

// Start executes the wrapped command but does not wait for it to finish.
func (c CommandWrapper) Start() error {
	return c.Cmd().Start()
}

// Run executes the wrapped command and waits for it to finish.
func (c CommandWrapper) Run() error {
	return c.Cmd().Run()
}

// Wait waits for the wrapped command to finish executing.
func (c CommandWrapper) Wait() error {
	return c.Cmd().Wait()
}

// SetOpts is a no-op method that returns nil.
// It is included for testing purposes.
func (c CommandWrapper) SetOpts(opts CommandOpts) CommandInterface {
	return nil
}

// String returns the string representation of the command.
func (c CommandWrapper) String() string {
	return exec.Command(c.cmd, c.args...).String()
}

type CommandOpts struct {
	String      string // This is the command string, if provided will take precendence over Name and Args
	Name        string
	Args        []string
	Env         []string
	Dir         string
	Stdout      io.Writer
	Stderr      io.Writer
	SysProcAttr *syscall.SysProcAttr // This is used to set process attributes like Pdeathsig
}

var DefaultCommand CommandInterface

// Command returns a CommandInterface with the given command and arguments.
func Command(ctx context.Context, opts CommandOpts) CommandInterface {
	if DefaultCommand != nil {
		return DefaultCommand.SetOpts(opts)
	}

	if opts.Env == nil {
		opts.Env = []string{
			"PATH=" + os.Getenv("PATH"),
			"HOME=" + os.Getenv("HOME"),
		}
	}

	if opts.String != "" {
		pieces := strings.SplitN(opts.String, " ", 2)
		opts.Name = pieces[0]

		if len(pieces) > 1 {
			opts.Args, _ = shlex.Split(pieces[1])
		}
	}

	return CommandWrapper{
		ctx:         ctx,
		cmd:         opts.Name,
		args:        opts.Args,
		dir:         opts.Dir,
		env:         opts.Env,
		stdout:      opts.Stdout,
		stderr:      opts.Stderr,
		sysProcAttr: opts.SysProcAttr,
	}
}
