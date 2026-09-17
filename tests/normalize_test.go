// Package tests holds golden and end-to-end suites that exercise
// Dashpot's packages together, as opposed to the unit tests living
// alongside each package.
package tests

import (
	"testing"

	"github.com/shyam-s00/dashpot/internal/normalize"
)

// goldenCase is one real-world-shaped tool call, repeated with the kind
// of trivial variation an agent actually produces between retries. All
// variants in a case must normalize to the same canonical key.
type goldenCase struct {
	name     string
	tool     string
	variants []string // raw `arguments` JSON, each a re-issue of "the same call"
}

// canonicalKey canonicalizes rawArgs and builds the tool+args key the
// same way the reflex engine will before calling ObserveBytes.
func canonicalKey(tool string, rawArgs string) string {
	b := normalize.NewKeyBuilder()
	return string(b.Build(tool, normalize.Canonicalize([]byte(rawArgs))))
}

func TestGolden_RepeatedCallsNormalizeIdentically(t *testing.T) {
	cases := []goldenCase{
		{
			// Claude Code's bash tool: agent re-runs the same failing
			// test, varying only key order and a "# retry" annotation.
			name: "claude_code_bash_retry_annotation",
			tool: "bash",
			variants: []string{
				`{"command":"pytest tests/test_auth.py","description":"Run auth tests"}`,
				`{"description":"Run auth tests","command":"pytest tests/test_auth.py"}`,
				`{"command":"pytest tests/test_auth.py # retry","description":"Run auth tests"}`,
				`{"command":"pytest tests/test_auth.py # attempt 2","description":"Run auth tests"}`,
			},
		},
		{
			// Cursor's run-command tool: fresh temp path/timestamp each
			// retry, but the command itself is unchanged.
			name: "cursor_blind_jiggle_temp_path_and_timestamp",
			tool: "run_terminal_cmd",
			variants: []string{
				`{"command":"cargo test","is_background":false,"cwd":"/tmp/tmp_1a2b3c4d5e"}`,
				`{"command":"cargo test","is_background":false,"cwd":"/tmp/tmp_9f8e7d6c5b"}`,
				`{"cwd":"/var/folders/xy/randomabc123/T/","command":"cargo test","is_background":false}`,
			},
		},
		{
			// Cline's grep tool: whitespace-only regex variation.
			name: "cline_regex_whitespace_spasm",
			tool: "search_files",
			variants: []string{
				`{"regex":"func  foo","path":"src"}`,
				`{"regex":"func foo","path":"src"}`,
				`{"path":"src","regex":"func   foo"}`,
			},
		},
		{
			// A retried write_file call with a fresh UUID request id
			// each time, same format, different value.
			name: "write_file_with_fresh_uuid_each_retry",
			tool: "write_file",
			variants: []string{
				`{"path":"/etc/app.conf","request_id":"550e8400-e29b-41d4-a716-446655440000"}`,
				`{"path":"/etc/app.conf","request_id":"6ba7b810-9dad-11d1-80b4-00c04fd430c8"}`,
			},
		},
		{
			// Same shape, but the id is a raw hex nonce rather than a
			// UUID — a different masking rule, still deterministic.
			name: "write_file_with_fresh_hex_nonce_each_retry",
			tool: "write_file",
			variants: []string{
				`{"path":"/etc/app.conf","request_id":"a1b2c3d4e5f6a1b2"}`,
				`{"path":"/etc/app.conf","request_id":"9f8e7d6c5b4a3f2e"}`,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if len(c.variants) < 2 {
				t.Fatalf("golden case %q needs at least 2 variants to prove anything", c.name)
			}
			want := canonicalKey(c.tool, c.variants[0])
			for i, variant := range c.variants[1:] {
				if got := canonicalKey(c.tool, variant); got != want {
					t.Errorf("variant %d: canonical key = %q, want %q\n  from: %s", i+1, got, want, variant)
				}
			}
		})
	}
}

// TestGolden_DistinctCallsStayDistinct: masking volatile noise must not
// erase the parts of a call that actually matter.
func TestGolden_DistinctCallsStayDistinct(t *testing.T) {
	cases := []struct {
		name string
		tool string
		a, b string
	}{
		{
			name: "different_commands",
			tool: "bash",
			a:    `{"command":"pytest tests/test_auth.py"}`,
			b:    `{"command":"pytest tests/test_billing.py"}`,
		},
		{
			name: "different_tools_same_args",
			tool: "", // tool passed per-arg below
			a:    `{"path":"/etc/hosts"}`,
			b:    `{"path":"/etc/hosts"}`,
		},
		{
			name: "different_regex_target",
			tool: "search_files",
			a:    `{"regex":"func foo"}`,
			b:    `{"regex":"func bar"}`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var ka, kb string
			if c.name == "different_tools_same_args" {
				ka = canonicalKey("read_file", c.a)
				kb = canonicalKey("write_file", c.b)
			} else {
				ka = canonicalKey(c.tool, c.a)
				kb = canonicalKey(c.tool, c.b)
			}
			if ka == kb {
				t.Errorf("distinct calls collapsed to the same key: %q", ka)
			}
		})
	}
}
