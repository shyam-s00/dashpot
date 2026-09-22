// Command dashpot is a loop-breaking proxy for AI agent tool calls.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/shyam-s00/dashpot/internal/config"
	"github.com/shyam-s00/dashpot/internal/proxy"
	"github.com/shyam-s00/dashpot/internal/reflex"
	"github.com/shyam-s00/dashpot/internal/supervisor"
)

const defaultConfigPath = ".dashpot.yaml"

// syncWriter serializes writes to stdout: the intercept path and the
// raw child-response copy both write to it from separate goroutines,
// and their bytes must never interleave mid-line.
type syncWriter struct {
	mu sync.Mutex
	w  io.Writer
}

func (s *syncWriter) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Write(p)
}

const version = "dashpot dev"

const usage = `usage: dashpot mcp-proxy [flags] -- <mcp-server> [args...]

flags:
  -threshold uint       estimate at which a repeated tool call is intercepted (default 4)
  -tick-duration dur    reflex engine decay tick (default 10s)
  -allow-tools string   comma-separated tool names that skip the reflex engine
  -config path          path to a .dashpot.yaml config file (default ./.dashpot.yaml)
`

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}
	switch args[0] {
	case "--version":
		fmt.Fprintln(stdout, version)
		return 0
	case "mcp-proxy":
		return runProxy(args[1:], stdin, stdout, stderr)
	default:
		fmt.Fprintf(stderr, "dashpot: unknown command %q\n", args[0])
		return 2
	}
}

func runProxy(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	dashpotArgs, childArgv, ok := splitArgs(args)
	if !ok || len(childArgv) == 0 {
		fmt.Fprint(stderr, usage)
		return 2
	}

	cfg, err := config.Load(extractFlagValue(dashpotArgs, "config", defaultConfigPath))
	if err != nil {
		fmt.Fprintf(stderr, "dashpot: %v\n", err)
		return 2
	}

	defaultThreshold, defaultTick := uint(4), 10*time.Second
	if cfg.Threshold != 0 {
		defaultThreshold = uint(cfg.Threshold)
	}
	if cfg.TickDuration != 0 {
		defaultTick = cfg.TickDuration
	}

	fs := flag.NewFlagSet("mcp-proxy", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.String("config", defaultConfigPath, "path to a .dashpot.yaml config file")
	threshold := fs.Uint("threshold", defaultThreshold, "estimate at which a repeated tool call is intercepted")
	tick := fs.Duration("tick-duration", defaultTick, "reflex engine decay tick")
	allowTools := fs.String("allow-tools", "", "comma-separated tool names that skip the reflex engine")
	if err := fs.Parse(dashpotArgs); err != nil {
		return 2
	}

	supervisor.IgnoreSIGPIPE()

	proc, childOut, err := supervisor.Start(childArgv)
	if err != nil {
		fmt.Fprintf(stderr, "dashpot: starting child: %v\n", err)
		return 1
	}

	skip := splitCSV(*allowTools)
	if len(skip) == 0 {
		skip = cfg.AllowTools
	}
	exempt := reflex.NewExemptions(skip, cfg.ToolThresholds)
	engine := reflex.New(reflex.Config{
		NumBuckets:   8192,
		TickDuration: *tick,
		Threshold:    uint32(*threshold),
	}, exempt)

	out := &syncWriter{w: stdout}
	p := proxy.New(engine, stdin, proc.Stdin(), out, stderr)

	clientErr := make(chan error, 1)
	go func() { clientErr <- p.Run() }()

	copyErr := make(chan error, 1)
	go func() {
		_, err := io.Copy(out, childOut)
		copyErr <- err
	}()

	sig := supervisor.NotifyShutdown()

	select {
	case <-sig:
		proc.Stop(2 * time.Second)
	case <-proc.Done():
		// child exited on its own
	case err := <-clientErr:
		if err == nil {
			_ = proc.Stdin().Close() // client disconnected cleanly; let the child finish
		} else {
			fmt.Fprintf(stderr, "dashpot: %v\n", err)
			proc.Stop(2 * time.Second)
		}
	case err := <-copyErr:
		if err != nil {
			fmt.Fprintf(stderr, "dashpot: %v\n", err)
		}
		proc.Stop(2 * time.Second)
	}

	<-proc.Done()
	return proc.ExitCode()
}

// splitArgs splits args at "--" into Dashpot's own flags and the child
// command to run.
func splitArgs(args []string) (dashpotArgs, childArgv []string, ok bool) {
	for i, a := range args {
		if a == "--" {
			return args[:i], args[i+1:], true
		}
	}
	return nil, nil, false
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

// extractFlagValue scans args for -name/--name (space- or =-separated)
// without needing a full flag.FlagSet, since -config's value must be
// known before the real FlagSet's other defaults can be built.
func extractFlagValue(args []string, name, def string) string {
	for i, a := range args {
		switch {
		case a == "-"+name || a == "--"+name:
			if i+1 < len(args) {
				return args[i+1]
			}
		case strings.HasPrefix(a, "-"+name+"="):
			return strings.TrimPrefix(a, "-"+name+"=")
		case strings.HasPrefix(a, "--"+name+"="):
			return strings.TrimPrefix(a, "--"+name+"=")
		}
	}
	return def
}
