// Package config reads the optional .dashpot.yaml file: per-tool
// overrides and exclusions the CLI flags can't express as cleanly.
package config

import (
	"fmt"
	"os"
	"time"

	"go.yaml.in/yaml/v4"
)

// File is the parsed config. A zero File (no file present) means
// "use built-in defaults" for every field.
type File struct {
	Threshold      uint32
	TickDuration   time.Duration
	AllowTools     []string
	ToolThresholds map[string]uint32
}

// raw mirrors the on-disk YAML shape before duration parsing.
type raw struct {
	Threshold      uint32            `yaml:"threshold"`
	TickDuration   string            `yaml:"tick_duration"`
	AllowTools     []string          `yaml:"allow_tools"`
	ToolThresholds map[string]uint32 `yaml:"tool_thresholds"`
}

// Load reads and parses path. A missing file is not an error — the
// config file is optional — and returns a zero File.
func Load(path string) (File, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return File{}, nil
	}
	if err != nil {
		return File{}, err
	}

	var r raw
	if err := yaml.Unmarshal(data, &r); err != nil {
		return File{}, fmt.Errorf("parsing %s: %w", path, err)
	}

	f := File{
		Threshold:      r.Threshold,
		AllowTools:     r.AllowTools,
		ToolThresholds: r.ToolThresholds,
	}
	if r.TickDuration != "" {
		d, err := time.ParseDuration(r.TickDuration)
		if err != nil {
			return File{}, fmt.Errorf("parsing %s: tick_duration: %w", path, err)
		}
		f.TickDuration = d
	}
	return f, nil
}
