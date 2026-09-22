# Dashpot

**A small, fixed-memory proxy that stops an unattended AI agent from looping on its tools.**

Dashpot wraps a stdio [MCP](https://modelcontextprotocol.io) server and sits between an agent
and its tools. It watches every tool call, recognizes when the same call is being repeated in
a tight loop, and steps in before that loop burns tokens, floods the tool server, or runs
unsupervised all night. It's one small static binary: no gateway to run, no SDK to adopt. You
change the `command` of a server entry in your MCP config, and that's it.

---

## Why

When a person watches an agent, a stuck tool call costs a few seconds: they notice and press
Ctrl+C. When nobody is watching — CI jobs, batch pipelines, scheduled agents, containers left
running overnight — the same failure compounds instead:

- The agent retries the same failing call over and over, or bounces between two calls, with
  no one there to interrupt it.
- A polling loop with no sleep tool checks a job's status every second, hammering the
  upstream API the whole time.
- Nothing bounds how long this goes on, so it runs until someone notices the bill or the logs.

Dashpot exists to catch that one failure mode reliably, without getting in the way of normal
tool use.

### The name

A dashpot is a mechanical damper whose resistance grows with velocity: slow, deliberate motion
meets almost no resistance, while sudden high-speed motion meets a lot. That's the shape of the
behavior here — normal tool use passes straight through, and only a rapid, repeating pattern
gets stopped.

---

## How it works

```
   agent (spawns MCP servers from its config)
        |   JSON-RPC lines over stdio
        v
   +-------------------------------------------------------+
   |  dashpot mcp-proxy [flags] -- <real MCP server>       |
   |                                                       |
   |  request path:   decide per tools/call                |
   |                  forward, or reply instead              |
   |  response path:  streamed through untouched            |
   +---------------------------+---------------------------+
                               |   stdio (supervised child)
                               v
                     the real MCP server
```

One Dashpot process wraps one server for the life of one session.

**Request path.** Only `tools/call` requests are inspected; `initialize`, `tools/list`,
notifications and everything else pass straight through. For each tool call, Dashpot decodes
the tool name and arguments, builds a canonical key from them, and checks a fixed-memory
frequency counter. Most calls are forwarded unchanged, byte for byte. A call is answered
by Dashpot itself, instead of being forwarded, only once it has clearly been repeated too many
times too quickly. This decision takes microseconds — negligible next to the milliseconds to
seconds a real tool call takes.

**Response path.** Responses stream from the server back to the agent as-is. Dashpot never
parses, decodes or rewrites a response body, so a large file read or a big search result costs
nothing extra to pass through.

**The repeat key.** Raw arguments are canonicalized before comparison, so cosmetic differences
don't hide a real repeat: object keys are sorted, whitespace is collapsed, trailing retry
annotations like `# retry` are stripped, and volatile values — UUIDs, timestamps, hex tokens,
temp-file paths — are masked out. `grep foo  src` and `grep foo src` become the same call, and
so do two otherwise-identical calls that only differ by a timestamp or a temp path.

**The frequency counter.** Repeat counts live in
[EpochSketch](https://github.com/shyam-s00/epochsketch), a fixed-size (about 512 KiB), decaying
frequency sketch. Decay is computed the moment a key is read, not on a timer, so there's no
background work and no growth in memory with session length or the number of distinct calls
seen. Each tick of silence halves a key's count, so a call repeated slowly is forgiven while a
rapid burst is not.

**Fail open.** If Dashpot hits an internal error while making a decision, it logs the error and
forwards the call rather than blocking it. It also logs, rather than silently allowing, the two
edge cases it doesn't fully inspect today: a JSON-RPC batch request, and a single request line
over 1 MiB. Both are still forwarded unchanged either way — the fix was making that visible,
not changing the behavior — so the proxy is never the reason a session breaks.

---

## What it does

| Capability | Details |
| :--- | :--- |
| Repeat / loop detection | Trips when the decayed repeat count for a canonicalized tool call reaches a threshold (default: 4 times within a 10s decay window) |
| Synthetic reply on trip | The repeated call is not forwarded. The agent instead gets a normal, well-formed tool result (`isError: false`) telling it plainly to stop repeating and try a different approach |
| Argument canonicalization | Whitespace, key order, retry comments, UUIDs, timestamps, hex tokens and temp paths are normalized away before comparing calls |
| Per-tool control | An allow list of tools that skip detection entirely (e.g. legitimate status polling), plus per-tool threshold overrides |
| Config file | Optional `.dashpot.yaml` for defaults; CLI flags override it |
| Child process supervision | Starts the real MCP server as a child, forwards its stderr untouched, relays SIGINT/SIGTERM to it (SIGTERM, then SIGKILL after a 2s grace period), and exits with the child's own exit code |
| Clean disconnect | When the agent closes its side of stdin, Dashpot closes the child's stdin too and lets it finish on its own |
| Visible bypass paths | Batch requests, oversized lines, and any recovered internal error are logged to stderr instead of passing silently |

---

## Quick start

Requires Go 1.27 or newer.

```sh
git clone https://github.com/shyam-s00/dashpot
cd dashpot
make build          # writes bin/dashpot
```

Wrap any stdio MCP server by putting `dashpot mcp-proxy [flags] --` in front of its command:

```sh
bin/dashpot mcp-proxy -- npx -y @modelcontextprotocol/server-filesystem /workspace
```

In an MCP client config, the same thing looks like:

```json
{
  "mcpServers": {
    "filesystem": {
      "command": "/usr/local/bin/dashpot",
      "args": ["mcp-proxy", "--", "npx", "-y", "@modelcontextprotocol/server-filesystem", "/workspace"]
    }
  }
}
```

### Flags

| Flag | Default | Meaning |
| :--- | :--- | :--- |
| `-threshold` | `4` | Decayed repeat count at which a call is intercepted |
| `-tick-duration` | `10s` | Decay interval; the count halves each tick |
| `-allow-tools` | none | Comma-separated tool names that skip detection |
| `-config` | `./.dashpot.yaml` | Path to a config file |

### Config file

```yaml
threshold: 4
tick_duration: 10s
allow_tools:
  - get_job_status        # legitimate polling
tool_thresholds:
  search: 8               # searching repeatedly is often fine
  run_tests: 6
```

A missing file is not an error. CLI flags take precedence over the file for `threshold`,
`tick_duration` and `allow_tools`; `tool_thresholds` is file-only.

---

## Limitations and non-goals

- **Not a security or policy tool.** Dashpot does not decide whether a tool call is safe or
  allowed, and it does not sandbox anything. It only reacts to repetition. Pair it with a
  dedicated MCP policy proxy, container isolation and least-privilege credentials if you need
  those.
- **Only what passes through it.** It governs `tools/call` traffic over stdio. An agent's
  built-in, non-MCP tools are invisible to it, and remote MCP servers over HTTP aren't
  supported.
- **No cost tracking.** It never sees LLM/model traffic, so it has no notion of tokens or
  dollars — only tool-call frequency.
- **Frequency-based, not intent-based.** It can't tell a legitimate rapid poll from a stuck
  loop except by rate. Use the allow list and per-tool thresholds for tools that are meant to
  be called often.

---

## Design principles

- **Fixed memory.** The frequency sketch and I/O buffers are sized once at startup; memory
  doesn't grow with session length or the number of distinct calls seen.
- **No idle work.** No polling loops or background sweeps; decay is computed at read time.
- **Responses are never decoded.** The response path is a raw, unbuffered copy.
- **Explicit over clever.** Which tools skip detection, or use a different threshold, is
  something you declare — never guessed from a tool's name.
- **Fail open.** An internal error is logged and the call is forwarded; Dashpot cannot become
  the reason a session breaks.

### Design targets

| Target | Value |
| :--- | :--- |
| Resident memory | about 10 MiB or less at steady state |
| Idle CPU | none |
| Decision latency | tens of microseconds |
| Binary size | under 12 MiB (about 5 MiB as built today) |

---

## Development

```sh
make build       # build bin/dashpot
make check       # gofmt, go vet, tests with the race detector
make test-e2e    # end-to-end suite: mock client -> dashpot -> mock server
```

The code is organized by concern: `internal/mcp` (the line protocol), `internal/normalize`
(canonical keys), `internal/reflex` (the sketch-backed decision), `internal/proxy` (per-request
routing), `internal/supervisor` (child process lifecycle) and `internal/config`.

## License

MIT. See [LICENSE](LICENSE).
