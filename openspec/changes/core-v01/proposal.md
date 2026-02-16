## Why

AI agent outputs are opaque — when an agent produces a result, there's no standard way to reproduce the run, understand why it made the decisions it did, or challenge its outputs with evidence. Teams can't share, diff, or audit agent work using the tools they already trust. ContextSubstrate solves this by providing infrastructure that makes agent execution reproducible, debuggable, and contestable using developer-native primitives (files, hashes, diffs, CLI).

## What Changes

This is the initial implementation of ContextSubstrate v0.1. All capabilities are new:

- **Context Packs**: Immutable, content-addressed bundles that fully capture an agent run (prompts, inputs, tool calls, model params, execution order, environment metadata). Referenced via `ctx://<sha256>`.
- **Deterministic Replay**: Re-run an agent execution exactly as it originally occurred. Flag non-deterministic steps, identify dependency drift on replay failure.
- **Decision Drift Detection**: Structured diff between two Context Packs — identify prompt drift, tool choice drift, parameter drift, reasoning divergence, and output divergence.
- **Contestable Outputs**: Every produced artifact is traceable back to the Context Pack that produced it. Artifacts carry provenance metadata and can be verified.
- **Context Sharing**: Context Packs are referenced by hash, shareable across teams, forkable, and comparable — no shared repo or environment required.
- **CLI (`ctx`)**: Go-based command-line interface exposing all capabilities (`ctx replay`, `ctx diff`, `ctx verify`, `ctx fork`).

## Capabilities

### New Capabilities

- `context-packs`: Immutable, versioned, content-addressed bundles that capture the full state of an agent run (prompts, inputs, tool calls, model config, execution trace, environment metadata).
- `deterministic-replay`: Re-execute a captured agent run from a Context Pack, detecting dependency drift and reporting replay fidelity (exact, degraded, failed).
- `drift-detection`: Compare two Context Packs to produce structured and human-readable diffs of decision points — prompt, tool choice, parameter, reasoning, and output divergence.
- `contestable-outputs`: Artifact provenance tracking — every output carries metadata linking it back to its producing Context Pack, supporting verification and challenge.
- `context-sharing`: Hash-based sharing, forking, and comparison of Context Packs across teams and environments without platform coupling.

### Modified Capabilities

None — this is a greenfield implementation.

## Impact

- **New codebase**: Go CLI project with module, build, and test infrastructure
- **New storage format**: JSON-based Context Pack format with content-addressed blob storage in a Git-compatible filesystem layout
- **New CLI commands**: `ctx pack`, `ctx replay`, `ctx diff`, `ctx verify`, `ctx fork`
- **Dependencies**: Go standard library, SHA-256 hashing, JSON serialization. No cloud or external service dependencies.
- **No breaking changes**: Greenfield project, no existing users or APIs affected
