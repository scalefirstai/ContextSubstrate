# v0.1.0 — Initial Release

**ContextSubstrate** — Reproducible, debuggable, contestable AI agent execution.

`ctx` is an execution substrate for AI agents that makes their work reproducible, debuggable, and contestable using developer-native primitives — files, hashes, diffs, and CLI workflows.

## Highlights

This is the first public release of ContextSubstrate, delivering all five core capabilities:

### Context Packs
Immutable, content-addressed snapshots of AI agent executions. Each pack captures inputs, steps, outputs, and metadata, identified by a SHA-256 hash for tamper-evidence and deduplication.

### Deterministic Replay
Re-execute recorded agent runs step-by-step with fidelity tracking (exact, degraded, or failed). Verify that agent behavior is reproducible across environments.

### Decision Drift Detection
Structured diff between any two context packs. Detects prompt drift, step drift, and output drift with both human-readable and JSON output formats.

### Contestable Outputs
Sidecar metadata (`.ctx.json`) links artifacts back to the agent run that produced them. Verify provenance of any output against the context store.

### Context Sharing
Fork existing packs into mutable drafts for iteration. List and discover packs in the local store with parent tracking for full lineage.

## Commands

| Command | Description |
|---------|-------------|
| `ctx init` | Initialize a `.ctx/` store |
| `ctx pack <log-file>` | Create a context pack from an execution log |
| `ctx show <hash>` | Inspect a context pack |
| `ctx log` | List all context packs |
| `ctx diff <hash-a> <hash-b>` | Compare two packs for drift |
| `ctx replay <hash>` | Replay an agent run step-by-step |
| `ctx verify <artifact>` | Verify artifact provenance |
| `ctx fork <hash>` | Fork a pack into a mutable draft |
| `ctx completion <shell>` | Generate shell completions |

## Installation

```bash
# From source
go install github.com/contextsubstrate/ctx/cmd/ctx@latest

# Or download a binary from the Releases page
```

## What's Included

- 9 CLI commands covering the full pack lifecycle
- Content-addressed blob store with SHA-256 integrity
- Canonical JSON serialization for deterministic hashing
- Comprehensive test suite (unit + integration)
- CI/CD pipeline (GitHub Actions)
- Cross-platform release builds (Linux, macOS, Windows — amd64, arm64)
- Shell completions for bash, zsh, fish, and PowerShell

## What's Next

- Remote pack sharing (push/pull)
- Pack signing and verification
- Web UI for pack inspection
- IDE extensions

## Links

- [Documentation](https://github.com/scalefirstai/ContextSubstrate#readme)
- [Contributing Guide](https://github.com/scalefirstai/ContextSubstrate/blob/main/CONTRIBUTING.md)
- [OpenRudder](https://github.com/scalefirstai/openrudder) — uses ContextSubstrate for agent observability
