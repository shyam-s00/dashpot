package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireCat(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("cat"); err != nil {
		t.Skip("cat not available")
	}
}

func TestRun_Version(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--version"}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if stdout.Len() == 0 {
		t.Error("expected version output")
	}
}

func TestRun_NoArgsPrintsUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(nil, strings.NewReader(""), &stdout, &stderr)
	if code == 0 {
		t.Error("expected non-zero exit for no arguments")
	}
}

func TestRun_MissingChildCommand(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"mcp-proxy", "-threshold", "4"}, strings.NewReader(""), &stdout, &stderr)
	if code == 0 {
		t.Error("expected non-zero exit for a missing child command")
	}
}

func TestRunProxy_EchoesThroughRealChildAndExitsClean(t *testing.T) {
	requireCat(t)
	const input = `{"jsonrpc":"2.0","id":1,"method":"initialize"}` + "\n"
	var stdout, stderr bytes.Buffer

	code := run([]string{"mcp-proxy", "--", "cat"}, strings.NewReader(input), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if stdout.String() != input {
		t.Errorf("stdout = %q, want %q", stdout.String(), input)
	}
}

func TestRunProxy_TripsCircuitBreakerThroughRealChild(t *testing.T) {
	requireCat(t)
	const callFmt = `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"bash","arguments":{"command":"cargo test"}}}` + "\n"
	var input strings.Builder
	for i := 1; i <= 4; i++ {
		fmt.Fprintf(&input, callFmt, i)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"mcp-proxy", "-threshold", "4", "--", "cat"}, strings.NewReader(input.String()), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if got := strings.Count(stdout.String(), `"method":"tools/call"`); got != 3 {
		t.Errorf("child echoed %d calls back, want 3", got)
	}
	if !strings.Contains(stdout.String(), "CIRCUIT BREAKER") {
		t.Errorf("expected a circuit-breaker response, got %q", stdout.String())
	}
}

func TestRunProxy_ConfigFileLowersThreshold(t *testing.T) {
	requireCat(t)
	cfgPath := filepath.Join(t.TempDir(), ".dashpot.yaml")
	if err := os.WriteFile(cfgPath, []byte("threshold: 2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	const callFmt = `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"bash","arguments":{"command":"cargo test"}}}` + "\n"
	var input strings.Builder
	for i := 1; i <= 2; i++ {
		fmt.Fprintf(&input, callFmt, i)
	}
	var stdout, stderr bytes.Buffer

	code := run([]string{"mcp-proxy", "-config", cfgPath, "--", "cat"}, strings.NewReader(input.String()), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if got := strings.Count(stdout.String(), `"method":"tools/call"`); got != 1 {
		t.Errorf("child echoed %d calls back, want 1 (threshold 2 from config)", got)
	}
	if !strings.Contains(stdout.String(), "CIRCUIT BREAKER") {
		t.Errorf("expected the 2nd call to trip the breaker, got %q", stdout.String())
	}
}

func TestRunProxy_CLIFlagOverridesConfigFile(t *testing.T) {
	requireCat(t)
	cfgPath := filepath.Join(t.TempDir(), ".dashpot.yaml")
	if err := os.WriteFile(cfgPath, []byte("threshold: 2\n"), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	const line = `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"bash","arguments":{"command":"cargo test"}}}` + "\n"
	var stdout, stderr bytes.Buffer

	code := run([]string{"mcp-proxy", "-config", cfgPath, "-threshold", "4", "--", "cat"}, strings.NewReader(line), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit code = %d, stderr = %s", code, stderr.String())
	}
	if strings.Contains(stdout.String(), "CIRCUIT BREAKER") {
		t.Errorf("CLI -threshold 4 should override config's threshold 2, got %q", stdout.String())
	}
}
