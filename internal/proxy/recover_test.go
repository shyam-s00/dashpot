package proxy

import (
	"bytes"
	"strings"
	"testing"

	"github.com/shyam-s00/dashpot/internal/reflex"
)

// TestSafeDecide_RecoversFromPanic forces a nil-pointer panic inside
// decide and checks it falls back to ActionPass instead of propagating,
// and that the recovery is logged rather than silent.
func TestSafeDecide_RecoversFromPanic(t *testing.T) {
	var childIn, out, logs bytes.Buffer
	p := New(newTestEngine(4), strings.NewReader(""), &childIn, &out, &logs)
	p.keys = nil

	line := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"bash","arguments":{}}}`)
	action, _ := p.safeDecide(line)
	if action != reflex.ActionPass {
		t.Errorf("action after recovered panic = %v, want ActionPass", action)
	}
	if !strings.Contains(logs.String(), "internal error") {
		t.Errorf("expected the recovered panic to be logged, got %q", logs.String())
	}
}
