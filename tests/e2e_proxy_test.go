package tests

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/shyam-s00/dashpot/internal/proxy"
	"github.com/shyam-s00/dashpot/internal/reflex"
)

var dashpotBin, mockserverBin, mockclientBin string

// TestMain builds the real dashpot binary plus the mockserver/mockclient
// test doubles once, shared across this package's tests, instead of
// recompiling per test.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "dashpot-e2e-*")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)

	dashpotBin = build(dir, "dashpot", "github.com/shyam-s00/dashpot/cmd/dashpot")
	mockserverBin = build(dir, "mockserver", "github.com/shyam-s00/dashpot/tests/mockserver")
	mockclientBin = build(dir, "mockclient", "github.com/shyam-s00/dashpot/tests/mockclient")

	os.Exit(m.Run())
}

func build(dir, name, pkg string) string {
	out := dir + "/" + name
	cmd := exec.Command("go", "build", "-o", out, pkg)
	if output, err := cmd.CombinedOutput(); err != nil {
		panic(fmt.Sprintf("building %s: %v\n%s", pkg, err, output))
	}
	return out
}

// TestE2E_LoopInterceptionThroughRealBinaries chains the actual
// compiled binaries — mockclient spawns dashpot, dashpot spawns
// mockserver — mirroring how a real MCP client deploys Dashpot.
func TestE2E_LoopInterceptionThroughRealBinaries(t *testing.T) {
	cmd := exec.Command(mockclientBin, dashpotBin, "mcp-proxy", "-threshold", "4", "--", mockserverBin)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("mockclient run: %v", err)
	}

	summary := stdout.String()
	forwarded := extractInt(t, summary, "forwarded=")
	tripped := extractInt(t, summary, "tripped=")
	serverCount := extractInt(t, summary, `"count":`)

	if forwarded != 3 {
		t.Errorf("forwarded = %d, want 3 (calls 1-3 pass)", forwarded)
	}
	if tripped != 7 {
		t.Errorf("tripped = %d, want 7 (calls 4-10 intercepted)", tripped)
	}
	if serverCount != 3 {
		t.Errorf("mock server recorded %d executions, want 3", serverCount)
	}
}

func extractInt(t *testing.T, s, key string) int {
	t.Helper()
	i := strings.Index(s, key)
	if i < 0 {
		t.Fatalf("%q not found in summary: %s", key, s)
	}
	rest := s[i+len(key):]
	end := strings.IndexAny(rest, " \n},")
	if end < 0 {
		end = len(rest)
	}
	n, err := strconv.Atoi(rest[:end])
	if err != nil {
		t.Fatalf("parsing %q from summary %q: %v", key, s, err)
	}
	return n
}

// TestE2E_MemoryFootprintBounded drives 50,000 tool calls through the
// in-process proxy pipeline and asserts heap growth stays bounded,
// rather than an absolute RSS ceiling (see the design doc's rationale).
func TestE2E_MemoryFootprintBounded(t *testing.T) {
	const (
		warmup   = 5_000
		requests = 50_000
		// keyspace deliberately exceeds numBuckets so the fixed-size
		// table must keep evicting old keys for new ones throughout.
		keyspace  = 4_000
		numBucket = 1024
	)

	engine := reflex.New(reflex.Config{
		NumBuckets:   numBucket,
		TickDuration: time.Second,
		Threshold:    1 << 30, // unreachable: this measures memory, not trips
	}, nil)

	runLines(engine, genLines(warmup, keyspace))

	// Built before snapshotting: generating 50,000 lines itself
	// allocates megabytes, which must not be mistaken for the proxy's
	// own growth.
	measured := genLines(requests, keyspace)

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	runLines(engine, measured)

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	grew := int64(after.HeapAlloc) - int64(before.HeapAlloc)
	const maxGrowthBytes = 2 << 20 // 2 MiB slack for GC/runtime bookkeeping noise
	if grew > maxGrowthBytes {
		t.Errorf("heap grew by %d bytes over %d requests (before=%d after=%d), want <= %d",
			grew, requests, before.HeapAlloc, after.HeapAlloc, maxGrowthBytes)
	}
}

// genLines builds n synthetic tools/call lines up front, cycling
// through keyspace distinct commands.
func genLines(n, keyspace int) string {
	var b strings.Builder
	for i := range n {
		fmt.Fprintf(&b,
			`{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"bash","arguments":{"command":"cargo test %d"}}}`+"\n",
			i, i%keyspace,
		)
	}
	return b.String()
}

// runLines streams input through one long-lived Proxy.Run call,
// discarding output — the steady-state shape Dashpot actually runs in
// production, not one Proxy per request.
func runLines(engine *reflex.Engine, input string) {
	p := proxy.New(engine, strings.NewReader(input), io.Discard, io.Discard, io.Discard)
	_ = p.Run()
}
