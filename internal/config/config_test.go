package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func write(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".dashpot.yaml")
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestLoad_MissingFileIsNotAnError(t *testing.T) {
	f, err := Load(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if f.Threshold != 0 || f.TickDuration != 0 || f.AllowTools != nil || f.ToolThresholds != nil {
		t.Errorf("expected a zero File, got %+v", f)
	}
}

func TestLoad_FullConfig(t *testing.T) {
	path := write(t, `
threshold: 10
tick_duration: 30s
allow_tools:
  - get_status
  - poll_job
tool_thresholds:
  get_status: 50
`)
	f, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if f.Threshold != 10 {
		t.Errorf("Threshold = %d, want 10", f.Threshold)
	}
	if f.TickDuration != 30*time.Second {
		t.Errorf("TickDuration = %v, want 30s", f.TickDuration)
	}
	if len(f.AllowTools) != 2 || f.AllowTools[0] != "get_status" {
		t.Errorf("AllowTools = %v", f.AllowTools)
	}
	if f.ToolThresholds["get_status"] != 50 {
		t.Errorf("ToolThresholds[get_status] = %d, want 50", f.ToolThresholds["get_status"])
	}
}

func TestLoad_InvalidTickDurationErrors(t *testing.T) {
	path := write(t, "tick_duration: not-a-duration\n")
	if _, err := Load(path); err == nil {
		t.Error("expected an error for an invalid tick_duration")
	}
}

func TestLoad_InvalidYAMLErrors(t *testing.T) {
	path := write(t, "not: [valid: yaml")
	if _, err := Load(path); err == nil {
		t.Error("expected an error for malformed YAML")
	}
}
