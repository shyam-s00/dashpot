package mcp

import (
	"encoding/json"
	"testing"
)

func TestIsToolCall(t *testing.T) {
	cases := map[string]bool{
		"tools/call":        true,
		"initialize":        false,
		"notifications/foo": false,
		"tools/list":        false,
		"ping":              false,
		"":                  false,
	}
	for method, want := range cases {
		if got := IsToolCall(method); got != want {
			t.Errorf("IsToolCall(%q) = %v, want %v", method, got, want)
		}
	}
}

func TestRequest_PreservesIDShape(t *testing.T) {
	for _, raw := range []string{
		`{"jsonrpc":"2.0","id":42,"method":"tools/call","params":{}}`,
		`{"jsonrpc":"2.0","id":"abc-123","method":"tools/call","params":{}}`,
		`{"jsonrpc":"2.0","id":null,"method":"tools/call","params":{}}`,
	} {
		var req Request
		if err := json.Unmarshal([]byte(raw), &req); err != nil {
			t.Fatalf("Unmarshal(%s): %v", raw, err)
		}
		out, err := json.Marshal(req)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		var roundTrip Request
		if err := json.Unmarshal(out, &roundTrip); err != nil {
			t.Fatalf("Unmarshal round-trip: %v", err)
		}
		if string(roundTrip.ID) != string(req.ID) {
			t.Errorf("id round-trip: got %s, want %s", roundTrip.ID, req.ID)
		}
	}
}

func TestToolCallParams_Decode(t *testing.T) {
	var req Request
	raw := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"bash","arguments":{"command":"pytest"}}}`
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	var params ToolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		t.Fatalf("Unmarshal params: %v", err)
	}
	if params.Name != "bash" {
		t.Errorf("Name = %q, want %q", params.Name, "bash")
	}
	if string(params.Arguments) != `{"command":"pytest"}` {
		t.Errorf("Arguments = %s", params.Arguments)
	}
}
