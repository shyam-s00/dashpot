package proxy

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/shyam-s00/dashpot/internal/reflex"
)

func newTestEngine(threshold uint32) *reflex.Engine {
	return reflex.New(reflex.Config{NumBuckets: 1024, TickDuration: time.Second, Threshold: threshold}, nil)
}

func TestProxy_PassesNonToolCallsThrough(t *testing.T) {
	const line = `{"jsonrpc":"2.0","id":1,"method":"initialize"}` + "\n"
	var childIn, out bytes.Buffer

	p := New(newTestEngine(4), strings.NewReader(line), &childIn, &out)
	if err := p.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if childIn.String() != line {
		t.Errorf("childIn = %q, want %q", childIn.String(), line)
	}
	if out.Len() != 0 {
		t.Errorf("out should be empty, got %q", out.String())
	}
}

func TestProxy_TripsOnRepeatedToolCall(t *testing.T) {
	const callFmt = `{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":"bash","arguments":{"command":"cargo test"}}}` + "\n"
	var input strings.Builder
	for i := 1; i <= 4; i++ {
		fmt.Fprintf(&input, callFmt, i)
	}
	var childIn, out bytes.Buffer

	p := New(newTestEngine(4), strings.NewReader(input.String()), &childIn, &out)
	if err := p.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := strings.Count(childIn.String(), `"method":"tools/call"`); got != 3 {
		t.Errorf("child received %d calls, want 3", got)
	}
	if !strings.Contains(out.String(), "CIRCUIT BREAKER") {
		t.Errorf("expected a circuit-breaker response in out, got %q", out.String())
	}
	if !strings.Contains(out.String(), `"id":4`) {
		t.Errorf("intercept response should echo id 4, got %q", out.String())
	}
}

func TestProxy_MalformedLinePassesThrough(t *testing.T) {
	const line = "not json at all\n"
	var childIn, out bytes.Buffer

	p := New(newTestEngine(4), strings.NewReader(line), &childIn, &out)
	if err := p.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if childIn.String() != line {
		t.Errorf("malformed line should still forward raw, got %q", childIn.String())
	}
}

func TestProxy_OversizedLineForwardsRawAndContinues(t *testing.T) {
	oversized := strings.Repeat("y", 1024*1024+10)
	const normalFmt = `{"jsonrpc":"2.0","id":1,"method":"initialize"}` + "\n"
	input := oversized + "\n" + normalFmt
	var childIn, out bytes.Buffer

	p := New(newTestEngine(4), strings.NewReader(input), &childIn, &out)
	if err := p.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !strings.Contains(childIn.String(), oversized) {
		t.Error("oversized line should still reach the child raw")
	}
	if !strings.HasSuffix(childIn.String(), normalFmt) {
		t.Error("scanning should resume normally after the oversized line")
	}
}
