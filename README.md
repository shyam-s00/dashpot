# Dashpot: High-Level Specification & Architecture Roadmap

**A Zero-Overhead Runtime Governor and Policy Sidecar for Autonomous AI Agent Workloads**

---

## 1. Project Purpose & Vision

### 1.1 The Problem
Autonomous AI agents are increasingly deployed in headless backend environments—executing multi-step software engineering tasks, infrastructure remediation, data processing, and batch operations inside ephemeral containers and virtualized sandboxes.

Unlike interactive developer tooling where a human engineer observes the terminal and can manually intervene, headless agent pipelines operate unattended. In these unconstrained environments, agent loops present critical operational and financial liabilities:
- **Velocity Thrash & API Lockouts:** An agent encountering a persistent failure or polling a long-running background task often enters a rapid, tight execution loop. This burns tokens, pollutes model context windows, and triggers aggressive rate limits or account bans from third-party APIs.
- **Unbounded Blast Radius:** An autonomous agent equipped with shell, database, or cloud management tools can hallucinate destructive actions—such as recursive directory deletions, forced repository rewrites, or database drops—with zero human oversight.
- **Runaway Resource Consumption:** Without strict out-of-band ceilings, an agent caught in an unresolvable state can cycle indefinitely, consuming infrastructure capacity and cloud budgets before failure is detected.

### 1.2 The Solution
**Dashpot** is a lightweight, single-binary runtime governor that sits as a transparent sidecar or process wrapper between an autonomous agent client and its tools (operating over the Model Context Protocol, or MCP). 

Drawing its name from the mechanical dashpot—a viscous damper that absorbs kinetic energy and resists motion proportionally to velocity—Dashpot stabilizes unstable agent execution loops:
1. **Viscous Damping:** Smooths bursty, high-frequency tool invocations by introducing progressive micro-delays rather than abruptly terminating valid operations.
2. **Blast-Radius Firewalling:** Intercepts known destructive commands and confinement violations before tool execution occurs.
3. **Deterministic Hard Stops:** Enforces session-level resource and invocation caps, providing structured synthetic feedback that guides the agent to summarize and terminate gracefully.

---

## 2. Core Architectural Principles

Dashpot adheres to strict systems and performance guarantees designed for high-density production environments:

- **Zero-Dependency Single Binary:** Written in pure Go with zero C-bindings or heavyweight external runtimes. Dashpot compiles into a compact, statically linked binary suitable for minimal container base images (`scratch` or `alpine`).
- **Bounded Resource Footprint:** Memory consumption is strictly capped at initialization using fixed-size probabilistic tracking structures and pooled I/O buffers. Memory does not grow with session length, tool variety, or invocation volume.
- **Zero-Idle Overhead:** Dynamic temporal decay is evaluated intrinsically at observation time. Dashpot runs zero background reaper tickers or maintenance goroutines when idle.
- **Microsecond Pass-Through Latency:** Hot-path inspections (frame scanning, argument normalization, security checks, and velocity observation) execute in the microsecond range, introducing negligible overhead relative to tool execution and model inference.
- **Fail-Open Wire Resilience:** If an unexpected parsing error or internal panic occurs, Dashpot logs the diagnostic and fails open—transparently forwarding the raw payload to ensure the governor itself never becomes an accidental point of failure.

---

## 3. System Architecture & The Three-Valve Model

Dashpot inspects line-delimited JSON-RPC frames over standard I/O (or local domain sockets). While non-tool methods (handshakes, notifications, schema queries) stream through instantaneously, every tool execution request passes through a sequential three-valve governance pipeline:

```
                      ┌─────────────────────────────────────────┐
                      │        Autonomous Agent Runner          │
                      └────────────────────┬────────────────────┘
                                           │ JSON-RPC (tools/call)
                                           ▼
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                              DASHPOT RUNTIME GOVERNOR                                  │
│                                                                                        │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐  │
│  │ 1. Frame Scanner & Argument Canonicalizer                                        │  │
│  │    Normalizes whitespace, key ordering, and dynamic noise tokens                 │  │
│  └───────────────────────────────────────┬──────────────────────────────────────────┘  │
│                                          ▼                                             │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐  │
│  │ VALVE 1: BLAST-RADIUS & SECURITY GATE                                            │  │
│  │ Matches destructive signatures, sandbox escapes, and metadata SSRF requests      │  │
│  └───────────────────┬──────────────────────────────────────────┬───────────────────┘  │
│                      │ Passed                                   │ Denied               │
│                      ▼                                          ▼                      │
│  ┌──────────────────────────────────────────────────┐  ┌────────────────────────────┐  │
│  │ VALVE 2: SESSION BUDGET & CEILING REGULATOR      │  │ SYNTHETIC POLICY REFLEX    │  │
│  │ Verifies cumulative calls and duration limits    │  │ Structured intercept reply │  │
│  └───────────────────┬──────────────────────────────┘  └─────────────┬──────────────┘  │
│                      │ Within Budget                                 │                 │
│                      ▼                                               │                 │
│  ┌──────────────────────────────────────────────────┐                │                 │
│  │ VALVE 3: KINETIC VELOCITY DAMPER                 │                │                 │
│  │ Computes instantaneous velocity via decay sketch │                │                 │
│  │ - Low: Immediate pass-through                    │                │                 │
│  │ - Medium: Injects progressive micro-backoff      │                │                 │
│  │ - High: Emits cognitive reflex warning           │                │                 │
│  └───────────────────┬──────────────────────────────┘                │                 │
│                      │ Forward                                       │                 │
│                      ▼                                               │                 │
│  ┌──────────────────────────────────────────────────┐                │                 │
│  │ FLIGHT RECORDER & TELEMETRY                      │                │                 │
│  │ Streams structured JSONL event records to stderr │                │                 │
│  └───────────────────┬──────────────────────────────┘                │                 │
└──────────────────────┼───────────────────────────────────────────────┼─────────────────┘
                       │ Raw Passthrough                               │ Synthetic Reply
                       ▼                                               ▼
┌─────────────────────────────────────────────┐       ┌──────────────────────────────────┐
│ Upstream MCP Tool Server                    │       │ Agent Context Stream             │
│ (Bash, Filesystem, Database, Git, Cloud CLI)│       │ (Ingests feedback as tool output)│
└─────────────────────────────────────────────┘       └──────────────────────────────────┘
```

### 3.1 Valve 1: Blast-Radius & Security Gate
Evaluates tool arguments in single-pass microsecond time before execution.
- **Destructive Command Blocking:** Denies irreversible shell and data manipulation patterns (e.g., recursive root deletions, force-pushes to version control, table drops).
- **Filesystem Boundary Confinement:** Prevents path traversal outside designated workspace mounts.
- **Instance Metadata & SSRF Shielding:** Blocks network tools from accessing cloud provider metadata services (e.g., link-local addresses).
- **Feedback Mechanism:** When blocked, Dashpot synthesizes a structured tool result describing the policy violation, allowing the agent to formulate an alternative non-destructive approach without crashing the harness.

### 3.2 Valve 2: Session Budget & Ceiling Regulator
Provides hard bounds on autonomous workloads to eliminate unbounded execution.
- **Call Volume Ceilings:** Enforces a maximum number of cumulative tool invocations per session.
- **Session Duration Ceilings:** Enforces wall-clock execution deadlines.
- **Graceful Termination:** Upon reaching a ceiling, Dashpot synthesizes a final warning instructing the agent to complete its summary and conclude the task, allowing clean pod shutdown.

### 3.3 Valve 3: Kinetic Velocity Damper
Tracks the recency and frequency of tool-call patterns using a decaying sub-linear memory sketch.
- **Tier 1 (Normal Cadence):** Operations execute with zero artificial delay.
- **Tier 2 (Progressive Damping):** High-frequency retries or status polling encounter calculated micro-delays (e.g., progressive exponential backoff). The tool call succeeds, but the pipeline is rhythmically paced to prevent downstream rate-limiting.
- **Tier 3 (Reflex Circuit Break):** If an identical pattern repeats at high velocity without progress, Dashpot intercepts the call and delivers a cognitive reflection prompt advising the model to stop retrying and reconsider its hypothesis.

### 3.4 The Flight Recorder (Structured Telemetry)
Dashpot writes compact, single-line JSONL events to standard error or a dedicated logging socket. Each record captures the timestamp, tool name, execution latency, velocity metric, and the governor action taken (passed, damped, or intercepted). This stream integrates directly with container log collectors (e.g., FluentBit, Vector, Datadog) for cluster-wide auditability.

---

## 4. Supported Deployment Modalities

Dashpot is designed to deploy seamlessly across standard container and virtualization environments without requiring SDK integrations or modifications to agent application code:

1. **Kubernetes Sidecar Container:**
   Deploys alongside the agent or tool container within a Kubernetes Pod. Intercepts traffic via shared Unix domain sockets or standard IPC.
2. **Ephemeral MicroVM Process Wrapper:**
   Acts as the parent supervisor inside isolated micro-virtual machines (e.g., Firecracker, cloud container instances). Spawns the tool server as a supervised child process:
   ```sh
   dashpot mcp-proxy --max-calls 50 -- /usr/local/bin/mcp-server
   ```
3. **Multi-Server Gateway (Local Proxy):**
   Aggregates multiple underlying tool servers behind a unified governance layer, standardizing access control across heterogeneous tool providers.

---

## 5. Architectural Roadmap

```
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                                   DASHPOT ROADMAP                                      │
├────────────────────────────────────────────────────────────────────────────────────────┤
│  Phase 1: Kinetic Damper & Core Hardening                                              │
│  - Core streaming JSON-RPC frame proxy with subprocess lifecycle supervision.          │
│  - Fixed-memory decaying velocity sketch with intrinsic temporal decay.                │
│  - Progressive viscous micro-delay injection on high-velocity polling.                 │
│  - Argument canonicalization (normalizing quotes, flag order, volatile noise tokens).  │
│  - Rapid alternating cycle detection (fixed-size circular key history).                │
│  - Session tool-call volume ceiling with clean exit sequencing.                        │
├────────────────────────────────────────────────────────────────────────────────────────┤
│  Phase 2: Blast-Radius Firewall & Structured Telemetry                                 │
│  - Declarative policy engine (configurable pattern deny-lists via YAML or env).        │
│  - Filesystem scratch confinement checks and metadata SSRF protection.                 │
│  - High-performance JSONL audit stream (Flight Recorder) for container log ingestion.  │
│  - Context-aware cognitive reflex prompts (differentiating status polling vs actions). │
│  - Standardized container packaging and Kubernetes sidecar deployment manifests.      │
├────────────────────────────────────────────────────────────────────────────────────────┤
│  Phase 3: Multi-Server Gateway & Enterprise Governance                                 │
│  - Gateway mode: multiplexing multiple upstream MCP servers behind one governor.       │
│  - OpenTelemetry native metrics and distributed trace context propagation.             │
│  - Estimated session budget/cost tracking and enforcement.                             │
│  - Zero-parsing streaming response hashing for automated output progress detection.   │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 6. Summary

Dashpot bridges the critical gap between powerful autonomous models and safe production infrastructure. By combining the physical principles of kinetic damping with deterministic blast-radius gating and strict resource ceilings, Dashpot provides the missing operational safety harness required to run autonomous AI agents at enterprise scale.
