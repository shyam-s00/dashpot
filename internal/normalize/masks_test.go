package normalize

import "testing"

func TestMask_UUID(t *testing.T) {
	got := Mask("request 550e8400-e29b-41d4-a716-446655440000 failed")
	want := "request <UUID> failed"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMask_HexToken(t *testing.T) {
	got := Mask("commit a1b2c3d4e5f6 failed")
	want := "commit <HEX> failed"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMask_HexBoundary(t *testing.T) {
	// Under 8 hex characters is common, meaningful content (a short
	// flag value, a color code) — not the kind of high-entropy nonce
	// this mask targets. At/above 8, it is.
	if got := Mask("color abcd12"); got != "color abcd12" {
		t.Errorf("6-char hex should not be masked, got %q", got)
	}
	if got := Mask("commit abcd1234"); got != "commit <HEX>" {
		t.Errorf("8-char hex should be masked, got %q", got)
	}
}

func TestMask_ISODate(t *testing.T) {
	got := Mask("created 2024-03-15T10:30:00Z")
	want := "created <TS>"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMask_UnixTimestamp(t *testing.T) {
	got := Mask("expires 1710498600")
	want := "expires <TS>"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMask_TmpPath(t *testing.T) {
	cases := map[string]string{
		"read /tmp/tmp_123abc/file.txt":          "read <TMP_PATH>",
		"read /var/folders/xy/abc123/T/file.txt": "read <TMP_PATH>",
	}
	for input, want := range cases {
		if got := Mask(input); got != want {
			t.Errorf("Mask(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMask_UUIDBeforeHex(t *testing.T) {
	// A UUID also matches the looser hex-token pattern; it must be
	// masked as <UUID>, not fragmented into multiple <HEX> pieces.
	got := Mask("id 550e8400-e29b-41d4-a716-446655440000")
	want := "id <UUID>"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMask_UseCase1_TempPathVariation(t *testing.T) {
	a := Mask("pytest /tmp/tmp_123/test.py --foo")
	b := Mask("pytest /tmp/tmp_456/test.py --foo")
	if a != b {
		t.Errorf("temp path variation should mask identically: %q vs %q", a, b)
	}
}
