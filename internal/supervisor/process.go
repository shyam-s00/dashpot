// Package supervisor launches and manages the child MCP tool-server
// subprocess a Dashpot proxy wraps.
package supervisor

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Process wraps a running child subprocess: its stdin is piped for the
// caller to write to, stdout is returned for the caller to read, and
// stderr passes straight through to Dashpot's own.
type Process struct {
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	done    chan struct{}
	waitErr error
}

// Start launches argv[0] with argv[1:] as a child process.
func Start(argv []string) (*Process, io.Reader, error) {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	p := &Process{cmd: cmd, stdin: stdin, done: make(chan struct{})}
	go func() {
		p.waitErr = cmd.Wait()
		close(p.done)
	}()
	return p, stdout, nil
}

// Stdin returns the child's stdin for writing.
func (p *Process) Stdin() io.WriteCloser {
	return p.stdin
}

// Done is closed once the child has exited.
func (p *Process) Done() <-chan struct{} {
	return p.done
}

// ExitCode returns the child's exit code. Only meaningful after Done
// is closed.
func (p *Process) ExitCode() int {
	if p.waitErr == nil {
		return 0
	}
	if exitErr, ok := errors.AsType[*exec.ExitError](p.waitErr); ok {
		return exitErr.ExitCode()
	}
	return 1
}

// Stop signals the child to terminate, escalating to SIGKILL if it
// doesn't exit within timeout. Safe to call after the child has
// already exited.
func (p *Process) Stop(timeout time.Duration) {
	if p.cmd.Process == nil {
		return
	}
	select {
	case <-p.done:
		return
	default:
	}

	_ = p.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-p.done:
	case <-time.After(timeout):
		_ = p.cmd.Process.Kill()
		<-p.done
	}
}
