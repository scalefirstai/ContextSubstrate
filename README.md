<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="demo/files/logo-horizontal-dark.svg">
    <source media="(prefers-color-scheme: light)" srcset="demo/files/logo-horizontal-light.svg">
    <img alt="ContextSubstrate" src="demo/files/logo-horizontal-light.svg" width="420">
  </picture>
</p>

<p align="center">

[![CI](https://github.com/scalefirstai/ContextSubstrate/actions/workflows/ci.yml/badge.svg)](https://github.com/scalefirstai/ContextSubstrate/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/scalefirstai/ContextSubstrate)](https://goreportcard.com/report/github.com/scalefirstai/ContextSubstrate)
[![Go Reference](https://pkg.go.dev/badge/github.com/scalefirstai/ContextSubstrate.svg)](https://pkg.go.dev/github.com/scalefirstai/ContextSubstrate)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Release](https://img.shields.io/github/v/release/scalefirstai/ContextSubstrate)](https://github.com/scalefirstai/ContextSubstrate/releases/latest)

</p>

<p align="center"><strong>Reproducible, debuggable, contestable AI agent execution.</strong></p>

<p align="center">
  <img src="demo/ctx-demo.svg" alt="ctx demo" width="800">
</p>

---

## The Problem

AI agents are black boxes. When an agent rewrites your code, generates a report, or makes a decision, you can't answer basic questions:

- **What exactly did it do?** — No structured record of inputs, steps, and outputs.
- **Can I reproduce it?** — Same prompt, different day, different result. No way to tell why.
- **What changed between runs?** — Two runs, two outcomes. Where did they diverge?
- **Can I trust this output?** — No provenance chain linking an artifact back to the execution that produced it.

Every other layer of the software stack has observability and auditability. AI agent execution doesn't. **ContextSubstrate fixes that.**

## What is `ctx`?

`ctx` is a CLI tool that captures AI agent executions as **immutable, content-addressed context packs** — the same primitives developers already trust: files, hashes, diffs, and CLI workflows.

Think of it as **`git` for agent runs**: every execution gets a SHA-256 hash, can be inspected, diffed against another run, replayed for verification, and traced back to its outputs.

```
Agent Run → ctx pack → ctx://sha256:abc123…
                            ↓
              Inspect  (ctx show)
              Compare  (ctx diff)
              Replay   (ctx replay)
              Verify   (ctx verify)
              Fork     (ctx fork)
```

## Quick Start

```bash
# Install ctx CLI (system-wide — don't clone this repo into your project)
curl -fsSL https://raw.githubusercontent.com/scalefirstai/ContextSubstrate/main/scripts/install.sh | bash

# Initialize in YOUR project
cd your-project
ctx init

# Tell your agent
echo "Use 'ctx' for execution tracking" >> AGENTS.md
```

### Other Installation Methods

**Pre-built binaries** — download from [GitHub Releases](https://github.com/scalefirstai/ContextSubstrate/releases/latest) for Linux, macOS, and Windows (amd64/arm64).

**From source** (requires Go 1.23+):

```bash
go install github.com/scalefirstai/ContextSubstrate/cmd/ctx@latest
```

**Build locally:**

```bash
git clone https://github.com/scalefirstai/ContextSubstrate.git
cd ContextSubstrate
make build        # produces ./ctx binary
make install      # installs to $GOPATH/bin
```

### First Run

```bash
# Pack an agent execution log into an immutable context pack
ctx pack execution.json
# → ctx://sha256:a1b2c3d4…

# Inspect what the agent did
ctx show a1b2c3

# List all captured runs
ctx log
```

## Features

### Context Packs — Immutable Execution Records

Every agent run is captured as a **context pack**: an immutable, content-addressed snapshot containing prompts, inputs, tool calls, model parameters, outputs, and environment metadata.

Packs are identified by SHA-256 hashes (`ctx://sha256:…`), making them tamper-evident, deduplicatable, and shareable by reference.

```bash
ctx pack execution.json     # Create a pack from an execution log
ctx show <hash>             # Inspect pack contents
ctx log                     # List all packs, ordered by creation date
```

### Deterministic Replay — Verify Reproducibility

Re-execute a recorded agent run step-by-step. The replay engine tracks **fidelity** at three levels:

| Fidelity | Meaning | Exit Code |
|----------|---------|-----------|
| **Exact** | All steps reproduced identically | `0` |
| **Degraded** | Non-deterministic steps diverged | `1` |
| **Failed** | Deterministic steps could not reproduce | `2` |

```bash
ctx replay <hash>
# Reports: environment drift, missing inputs, step divergence
```

### Decision Drift Detection — Diff Any Two Runs

Structured comparison between two context packs. Identifies exactly where and why agent behavior diverged:

- **Prompt drift** — system prompt or user prompts changed
- **Tool drift** — different tools invoked
- **Parameter drift** — same tool, different parameters
- **Reasoning drift** — intermediate step outputs differ
- **Output drift** — final artifacts diverged

```bash
ctx diff <hash-a> <hash-b>           # JSON output (machine-readable)
ctx diff <hash-a> <hash-b> --human   # Human-readable summary
```

### Contestable Outputs — Artifact Provenance

Every artifact an agent produces can be traced back to the execution that created it. Sidecar metadata files (`.ctx.json`) link outputs to their context pack, recording the pack hash, inputs used, tools involved, and confidence level.

```bash
ctx verify output.txt
# → Pack: sha256:a1b2c3…  Created: 2026-01-15  Tools: read_file, write_file  Status: verified
```

### Context Sharing — Fork and Iterate

Create mutable drafts from immutable packs. Edit the draft, then finalize into a new pack — maintaining full lineage.

```bash
ctx fork <hash>
# → Draft created at .ctx/drafts/<hash>.draft.json
# Edit the draft, then finalize into a new pack
```

### Token Optimization — Smart Context Selection

Index your codebase, track changes between commits, and generate optimized context packs that select the most relevant files and symbols for a given task — within a token budget.

```bash
ctx index --commit <sha>                                  # Index a git commit
ctx delta --base <sha1> --head <sha2> --human             # File-level change detection
ctx optimize --task "fix auth bug" --token-cap 16000      # Generate optimized pack
ctx metrics --limit 20                                     # Token savings dashboard
ctx benchmark --commits 10                                 # Cold vs warm comparison
```

## Command Reference

### Core Commands

| Command | Description |
|---------|-------------|
| `ctx init` | Initialize a `.ctx/` store in the current directory |
| `ctx pack <log-file>` | Create an immutable context pack from an execution log |
| `ctx show <hash>` | Inspect a context pack's contents |
| `ctx log` | List all finalized context packs |
| `ctx diff <hash-a> <hash-b>` | Compare two packs and produce a drift report (`--human` for readable output) |
| `ctx replay <hash>` | Re-execute an agent run step-by-step with fidelity tracking |
| `ctx verify <artifact>` | Validate artifact provenance via sidecar metadata |
| `ctx fork <hash>` | Create a mutable draft from an existing pack |
| `ctx completion <shell>` | Generate shell completions (bash, zsh, fish, powershell) |

### Token Optimization Commands

| Command | Description |
|---------|-------------|
| `ctx index` | Index a git commit into the context graph (`--commit <sha>`, default HEAD) |
| `ctx delta` | Compute file-level changes between commits (`--base`, `--head` required) |
| `ctx optimize` | Generate an optimized context pack for a task (`--task`, `--token-cap`, `--include-tests`) |
| `ctx metrics` | Display token savings dashboard (`--limit N`) |
| `ctx benchmark` | Compare cold vs warm token usage across commits (`--commits N`) |

### Global Flags

| Flag | Description |
|------|-------------|
| `--verbose`, `-v` | Enable verbose output |
| `--version` | Show version, commit, and build date |

## How It Works

### Execution Log Format

The `ctx pack` command accepts a JSON execution log capturing everything about an agent run:

```json
{
  "model": {
    "identifier": "gpt-4",
    "parameters": { "temperature": 0.0 }
  },
  "system_prompt": "You are a helpful assistant.",
  "prompts": [
    { "role": "user", "content": "Summarize this file" }
  ],
  "inputs": [
    { "name": "readme.md", "content": "# Hello World" }
  ],
  "steps": [
    {
      "index": 0,
      "type": "tool_call",
      "tool": "read_file",
      "parameters": { "path": "readme.md" },
      "output": "# Hello World",
      "deterministic": true,
      "timestamp": "2026-01-15T10:30:00Z"
    }
  ],
  "outputs": [
    { "name": "summary.txt", "content": "A test project readme." }
  ],
  "environment": {
    "os": "darwin",
    "runtime": "go1.23",
    "tool_versions": { "read_file": "1.0" }
  }
}
```

### Storage Layout

```
.ctx/
├── config.json       # Store metadata
├── objects/           # Content-addressed blob storage (SHA-256)
│   ├── ab/            # First two hex chars of hash
│   │   └── cdef…      # Blob file (remaining hash chars)
│   └── …
├── packs/             # Pack manifest registry
│   └── <hash>         # Pack manifest files
└── graph/             # Context graph (token optimization)
    ├── manifests/     # JSONL metadata (commits, paths)
    └── snapshots/     # Per-commit file/symbol snapshots
```

### Design Principles

- **Content-addressed storage** — every blob stored by SHA-256 hash; same content is never stored twice
- **Canonical JSON serialization** — deterministic hashing via recursive key sorting
- **Atomic writes** — blobs written to temp files, then renamed atomically to prevent corruption
- **Hash prefix resolution** — short prefixes (e.g., `a1b2`) resolve automatically; ambiguity is detected and reported
- **Zero external dependencies** — only the Go standard library and Cobra for CLI; no databases, no cloud services

## Project Status

### Shipped

- [x] Content-addressed blob store (SHA-256)
- [x] Context pack creation, inspection, and listing
- [x] Structured diff / decision drift detection
- [x] Step-by-step replay with fidelity reporting
- [x] Artifact provenance verification
- [x] Pack forking for mutable drafts
- [x] Shell completion (bash, zsh, fish, powershell)
- [x] Git commit indexing and context graphs
- [x] File-level delta computation
- [x] Token-optimized context pack generation
- [x] Token savings metrics and benchmarking
- [x] Cross-platform releases (Linux, macOS, Windows; amd64, arm64)

### Planned

- [ ] Remote pack sharing (push/pull to server)
- [ ] Cryptographic pack signing and verification
- [ ] Web UI for pack inspection
- [ ] IDE extensions (VS Code, JetBrains)

## Used By

- **[OpenRudder](https://github.com/scalefirstai/openrudder)** — Open source framework for change detection in microservices applications to build Ambient Agents. ContextSubstrate provides the execution substrate that makes OpenRudder's agent runs reproducible, auditable, and contestable.

## Non-Goals

`ctx` is **infrastructure, not a workflow**. It deliberately does not:

- **Orchestrate agents** — `ctx` captures and analyzes execution; it does not run agents
- **Host models** — no inference, training, or model management
- **Sync files** — not a replacement for git, rsync, or cloud storage
- **Monitor in real-time** — designed for post-hoc analysis, not live dashboards

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, coding guidelines, and how to submit changes.

```bash
make build      # Compile the ctx binary
make test       # Run tests with race detector
make lint       # Run golangci-lint
make coverage   # Generate coverage report
```

## License

[MIT](LICENSE) — see [LICENSE](LICENSE) for details.
