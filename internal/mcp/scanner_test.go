package mcp

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

func TestScanner_ShortLines(t *testing.T) {
	sc := NewScanner(strings.NewReader("line one\nline two\nline three\n"))

	want := []string{"line one", "line two", "line three"}
	for i, w := range want {
		line, err := sc.Scan(nil)
		if err != nil {
			t.Fatalf("line %d: Scan: %v", i, err)
		}
		if string(line) != w {
			t.Errorf("line %d = %q, want %q", i, line, w)
		}
	}
	if _, err := sc.Scan(nil); err != io.EOF {
		t.Errorf("final Scan: got err %v, want io.EOF", err)
	}
}

func TestScanner_NoTrailingNewline(t *testing.T) {
	sc := NewScanner(strings.NewReader("only line, no newline"))

	line, err := sc.Scan(nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if string(line) != "only line, no newline" {
		t.Errorf("got %q", line)
	}
	if _, err := sc.Scan(nil); err != io.EOF {
		t.Errorf("second Scan: got err %v, want io.EOF", err)
	}
}

func TestScanner_EmptyInput(t *testing.T) {
	sc := NewScanner(strings.NewReader(""))
	if _, err := sc.Scan(nil); err != io.EOF {
		t.Errorf("got err %v, want io.EOF", err)
	}
}

// TestScanner_LineSpanningMultipleFills: a line larger than
// initialBufSize but under MaxLineSize forces several buffer refills.
func TestScanner_LineSpanningMultipleFills(t *testing.T) {
	big := strings.Repeat("x", 200*1024)
	sc := NewScanner(strings.NewReader(big + "\n" + "next\n"))

	line, err := sc.Scan(nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(line) != len(big) {
		t.Fatalf("got line length %d, want %d", len(line), len(big))
	}
	if string(line) != big {
		t.Fatalf("line content mismatch")
	}

	line, err = sc.Scan(nil)
	if err != nil || string(line) != "next" {
		t.Fatalf("second line = %q, err %v", line, err)
	}
}

// TestScanner_OversizedLineFailsOpen: an over-limit line is drained and
// forwarded raw, never silently dropped, and scanning resumes after.
func TestScanner_OversizedLineFailsOpen(t *testing.T) {
	oversized := strings.Repeat("y", MaxLineSize+10)
	sc := NewScanner(strings.NewReader(oversized + "\n" + "next\n"))

	var raw bytes.Buffer
	line, err := sc.Scan(&raw)
	if err != ErrLineTooLong {
		t.Fatalf("got err %v, want ErrLineTooLong", err)
	}
	if line != nil {
		t.Errorf("expected nil line on overflow, got %d bytes", len(line))
	}
	if got, want := raw.String(), oversized+"\n"; got != want {
		t.Errorf("raw passthrough = %d bytes, want %d bytes (the full oversized line + newline)", len(got), len(want))
	}

	// Scanning resumes normally on the next call.
	line, err = sc.Scan(nil)
	if err != nil || string(line) != "next" {
		t.Fatalf("line after overflow = %q, err %v", line, err)
	}
}

// TestScanner_OversizedLineNilRawOut confirms rawOut is optional — a
// caller that only wants to know a line was too long, without
// forwarding it, must not panic.
func TestScanner_OversizedLineNilRawOut(t *testing.T) {
	oversized := strings.Repeat("z", MaxLineSize+1)
	sc := NewScanner(strings.NewReader(oversized + "\n"))

	if _, err := sc.Scan(nil); err != ErrLineTooLong {
		t.Fatalf("got err %v, want ErrLineTooLong", err)
	}
}
