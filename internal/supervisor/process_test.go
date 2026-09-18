package supervisor

import (
	"bufio"
	"os/exec"
	"testing"
	"time"
)

func requireShell(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}
}

func TestProcess_ExitCode(t *testing.T) {
	requireShell(t)
	proc, _, err := Start([]string{"sh", "-c", "exit 3"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	<-proc.Done()
	if got := proc.ExitCode(); got != 3 {
		t.Errorf("ExitCode = %d, want 3", got)
	}
}

func TestProcess_EchoesStdinToStdout(t *testing.T) {
	requireShell(t)
	proc, stdout, err := Start([]string{"cat"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer proc.Stop(time.Second)

	if _, err := proc.Stdin().Write([]byte("hello\n")); err != nil {
		t.Fatalf("Write: %v", err)
	}
	_ = proc.Stdin().Close()

	line, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil {
		t.Fatalf("ReadString: %v", err)
	}
	if line != "hello\n" {
		t.Errorf("got %q, want %q", line, "hello\n")
	}
}

func TestProcess_StopEscalatesToKill(t *testing.T) {
	requireShell(t)
	proc, _, err := Start([]string{"sh", "-c", "trap '' TERM; sleep 30"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	start := time.Now()
	proc.Stop(200 * time.Millisecond)
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Errorf("Stop took %v, want well under 2s", elapsed)
	}
}

func TestProcess_StopIsSafeAfterExit(t *testing.T) {
	requireShell(t)
	proc, _, err := Start([]string{"sh", "-c", "exit 0"})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	<-proc.Done()
	proc.Stop(time.Second) // must not block or panic
}
