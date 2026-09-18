package supervisor

import (
	"os"
	"os/signal"
	"syscall"
)

// IgnoreSIGPIPE stops a broken pipe from killing the process outright;
// a write past a closed pipe then surfaces as an ordinary error.
func IgnoreSIGPIPE() {
	signal.Ignore(syscall.SIGPIPE)
}

// NotifyShutdown reports SIGINT/SIGTERM on the returned channel so the
// caller can drive a graceful shutdown.
func NotifyShutdown() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	return ch
}
