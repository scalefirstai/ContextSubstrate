## Why

AI agent runs consume significant tokens by re-scanning entire repositories on every invocation, even when only a few files have changed. ContextSubstrate currently captures and analyzes agent execution (packs, replay, diff, verify, fork), but has no mechanism to reduce token consumption by leveraging knowledge of what changed between commits. Token optimization through incremental context is a critical cost-reduction feature for teams running agents at scale.

## What Changes

This change introduces token optimization as a first-class feature via three new capabilities:

- **Context Graph Store**: A JSONL-based graph storage layer that maintains file snapshots, path identities, and commit metadata. Stored under `.ctx/graph/` with deterministic, append-friendly JSONL files.
- **Change Detection**: Git-integrated change detection using `diff-tree` that identifies files added, modified, and deleted between any two commits.
- **Incremental Indexing**: File-level indexing that computes content hashes, detects languages, and tracks file metadata per commit. Only changed files need re-processing on subsequent runs.
- **Delta Computation**: Commit-scoped delta reports that compare indexed snapshots to identify exactly what changed, enabling downstream consumers to focus context on affected files only.

## Capabilities

### New Capabilities

- `context-graph`: JSONL-based graph store for file snapshots, path identities, commit records, and (future) symbol/edge records. Stored under `.ctx/graph/`.
- `change-detection`: Git diff-tree integration to detect file-level changes between commits, producing structured ChangeSets.
- `incremental-indexing`: Per-commit file indexing that snapshots file metadata (size, LOC, language, content hash) and maintains stable path identities across commits.
- `delta-computation`: Compare two indexed commits to produce a DeltaReport of changed, added, and deleted files.

### Modified Capabilities

- `ctx init`: Now creates `.ctx/graph/manifests/` and `.ctx/graph/snapshots/` subdirectories in addition to existing store structure.

## Impact

- **New CLI commands**: `ctx index`, `ctx delta`
- **New packages**: `internal/graph/`, `internal/index/`, `internal/delta/`
- **Modified packages**: `internal/store/` (init extended)
- **Storage**: New `.ctx/graph/` directory with JSONL files per commit

## Phases

This is Phase 1 of 3:
- **Phase 1** (this change): Context graph store, change detection, file-level indexing, delta computation
- **Phase 2** (future): Symbol extraction, cache layer, optimized pack generation
- **Phase 3** (future): Token telemetry, metrics, benchmarking
