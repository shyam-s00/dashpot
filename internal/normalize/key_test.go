package normalize

import "testing"

func TestKeyBuilder_Build(t *testing.T) {
	b := NewKeyBuilder()
	got := string(b.Build("bash", []byte(`{"command":"ls"}`)))
	want := `bash:{"command":"ls"}`
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestKeyBuilder_DifferentToolsDifferentKeys(t *testing.T) {
	b := NewKeyBuilder()
	args := []byte(`{"path":"/etc/hosts"}`)
	k1 := string(b.Build("read_file", args))
	k2 := string(b.Build("write_file", args))
	if k1 == k2 {
		t.Errorf("different tool names must not collide: %q", k1)
	}
}

func TestKeyBuilder_ReusesBufferSafely(t *testing.T) {
	b := NewKeyBuilder()
	first := b.Build("bash", []byte(`{"command":"a"}`))
	firstCopy := string(first)

	// Build again — first now aliases the reused buffer and must not be
	// read after this point without having been copied.
	second := b.Build("bash", []byte(`{"command":"b"}`))

	if firstCopy == string(second) {
		t.Errorf("second Build should differ from the first: %q", second)
	}
	if firstCopy != "bash:{\"command\":\"a\"}" {
		t.Errorf("copied first key was corrupted: %q", firstCopy)
	}
}
