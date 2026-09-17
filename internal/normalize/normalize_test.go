package normalize

import "testing"

func TestCanonicalize_KeySorting(t *testing.T) {
	a := Canonicalize([]byte(`{"b":1,"a":2}`))
	b := Canonicalize([]byte(`{"a":2,"b":1}`))
	if string(a) != string(b) {
		t.Errorf("key order should not affect output: %s vs %s", a, b)
	}
}

func TestCanonicalize_NestedKeySorting(t *testing.T) {
	a := Canonicalize([]byte(`{"outer":{"z":1,"a":2},"first":true}`))
	b := Canonicalize([]byte(`{"first":true,"outer":{"a":2,"z":1}}`))
	if string(a) != string(b) {
		t.Errorf("nested key order should not affect output: %s vs %s", a, b)
	}
}

func TestCanonicalize_WhitespaceCollapsing(t *testing.T) {
	a := Canonicalize([]byte(`{"command":"ls -la /tmp"}`))
	b := Canonicalize([]byte(`{"command":"ls   -la  /tmp"}`))
	if string(a) != string(b) {
		t.Errorf("whitespace variation should not affect output: %s vs %s", a, b)
	}
}

func TestCanonicalize_LineEndings(t *testing.T) {
	a := Canonicalize([]byte(`{"script":"line1\nline2"}`))
	b := Canonicalize([]byte(`{"script":"line1\r\nline2"}`))
	if string(a) != string(b) {
		t.Errorf("CRLF vs LF should not affect output: %s vs %s", a, b)
	}
}

func TestCanonicalize_TrailingNoiseAndSemicolons(t *testing.T) {
	base := Canonicalize([]byte(`{"command":"pytest tests/test_auth.py"}`))
	cases := []string{
		`{"command":"pytest tests/test_auth.py # retry"}`,
		`{"command":"pytest tests/test_auth.py # attempt 2"}`,
		`{"command":"pytest tests/test_auth.py;"}`,
	}
	for _, c := range cases {
		got := Canonicalize([]byte(c))
		if string(got) != string(base) {
			t.Errorf("Canonicalize(%s) = %s, want %s", c, got, base)
		}
	}
}

func TestCanonicalize_MalformedInputDoesNotPanic(t *testing.T) {
	for _, raw := range [][]byte{
		nil,
		[]byte(""),
		[]byte(`{not json`),
		[]byte(`"just a string"`),
		[]byte(`[1,2,3]`),
	} {
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Canonicalize(%q) panicked: %v", raw, r)
				}
			}()
			Canonicalize(raw)
		}()
	}
}

func TestCanonicalize_DeterministicRepeat(t *testing.T) {
	raw := []byte(`{"command":"cargo test","path":"/tmp/build_a1b2c3d4"}`)
	first := string(Canonicalize(raw))
	for range 5 {
		if got := string(Canonicalize(raw)); got != first {
			t.Fatalf("Canonicalize not deterministic across calls: %q vs %q", got, first)
		}
	}
}
