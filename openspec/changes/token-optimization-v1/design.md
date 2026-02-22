## Context

ContextSubstrate is a Go CLI tool (`ctx`) that provides reproducibility and debugging infrastructure for AI agent execution. The existing system stores immutable Context Packs in a `.ctx/` directory using content-addressed blobs. This design extends the store with a JSONL-based context graph that tracks file-level state per commit, enabling incremental context and token optimization.

## Goals / Non-Goals

**Goals:**

- Define a JSONL storage format for commit-scoped file snapshots and metadata
- Integrate with Git to detect file changes between commits
- Provide file-level indexing with language detection and content hashing
- Enable delta computation between indexed commits
- Keep the design extensible for Phase 2 (symbols, edges, cache)

**Non-Goals:**

- AST-level symbol extraction (Phase 2)
- Import/call edge graphs (Phase 2)
- Token counting or budget optimization (Phase 2)
- Telemetry or metrics collection (Phase 3)
- Support for non-Git version control systems

## Decisions

### 1. JSONL Storage Format

**Decision**: Use JSONL (JSON Lines) for all graph data, one record per line.

JSONL was chosen over SQLite or custom binary formats because:
- Append-friendly: new records can be added without rewriting the file
- Human-readable and debuggable with standard tools (`jq`, `grep`)
- Git-friendly: line-based diffs work naturally
- No external dependencies: Go's standard library handles JSON natively
- Deterministic: files can be sorted and reproduced identically

### 2. Storage Layout

**Decision**: Store graph data under `.ctx/graph/` with manifests (global) and snapshots (per-commit).

```
.ctx/graph/
  manifests/
    commits.jsonl       # Commit identity records (append-only)
    paths.jsonl         # Stable path identity records (append-only)
  snapshots/
    <commit_sha>/
      files.jsonl       # File snapshots for this commit
      symbols.jsonl     # (Phase 2) Symbol definitions
      regions.jsonl     # (Phase 2) Text span regions
      edges.imports.jsonl   # (Phase 2) Import edges
      edges.calls.jsonl     # (Phase 2) Call edges
```

Manifests are append-only global files. Snapshots are per-commit directories containing the full file state at that commit. This design supports efficient incremental indexing — only new commits need new snapshot directories.

### 3. Path Identity via Content Hash

**Decision**: Use a deterministic hash of the file path as the PathID.

PathIDs are SHA-256 hashes (truncated to 128 bits) of the file path string. This provides:
- Stable identity across commits even if content changes
- No need for sequential ID generation or coordination
- Deterministic and reproducible

### 4. Git Integration via Subprocess

**Decision**: Shell out to `git` commands rather than using a Git library.

This avoids adding a heavy dependency (e.g., go-git) and works with any Git installation. Commands used: `diff-tree`, `ls-tree`, `show`, `rev-parse`, `log`, `rev-list`.

### 5. Language Detection via Extension

**Decision**: Detect programming language from file extension (Phase 1). Tree-sitter or AST-based detection deferred to Phase 2.

Simple extension-based detection covers the most common languages and avoids adding dependencies in Phase 1.

## Record Types

All records share a `type` field for polymorphic deserialization:

- `commit`: Git commit metadata (SHA, parent, author, message, timestamp)
- `path`: Stable file path identity (path ID, repo path, first/last seen commits)
- `file_snapshot`: File state at a commit (content hash, language, LOC, byte size, binary/generated flags)
- `symbol`: (Phase 2) Symbol definition (function, class, type)
- `region`: (Phase 2) Text span within a file
- `import_edge`: (Phase 2) File-level import dependency
- `call_edge`: (Phase 2) Symbol-level call dependency

## CLI Commands

- `ctx index [--commit SHA]`: Index HEAD or a specific commit into the context graph
- `ctx delta --base SHA --head SHA [--human]`: Compute and display changes between two indexed commits
