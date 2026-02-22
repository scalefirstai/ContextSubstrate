Below is a **focused, implementation-ready design spec** for adding **Token Optimization as a First-Class Feature** in ContextSubstrate.

This is designed to differentiate from Continue/Aider/LlamaIndex by making:

> **Token savings measurable, reproducible, and commit-scoped.**

---

# ContextSubstrate

# Token Optimization & Incremental Context Spec (v1)

---

# 1. Objective

Reduce LLM token consumption by eliminating redundant repository scanning and re-summarization.

Enable:

* Commit-scoped incremental context reuse
* Deterministic ContextPack generation
* Token savings as a measurable KPI
* Cache hit rate visibility
* Cost/latency ROI reporting

---

# 2. Core Principles

1. **Context is versioned**
2. **Only deltas are recomputed**
3. **Context is addressable**
4. **Token efficiency is observable**
5. **ContextPacks are reproducible**

---

# 3. System Architecture

```
Git Repo
   ↓
Change Detector (commit diff)
   ↓
Incremental Parser (AST + symbols)
   ↓
Context Graph Store
   ↓
Context Cache Layer
   ↓
ContextPack Generator
   ↓
LLM
   ↓
Telemetry Collector
```

---

# 4. Components

---

## 4.1 Commit-Aware Change Detector

### Input:

* Base commit SHA
* Head commit SHA

### Output:

```json
{
  "files_changed": [],
  "files_added": [],
  "files_deleted": []
}
```

---

## 4.2 Incremental Symbol Graph Builder

Leverages:

* Tree-sitter for AST
* Language-aware symbol extraction

### Stores:

* file_digest
* symbol_id
* references
* dependency edges

### Invalidation Strategy:

If file_digest changes:

* Recompute only that file
* Update dependent symbols

---

## 4.3 Context Graph Store

Graph schema:

Nodes:

* File
* Symbol
* Module
* ADR
* Commit

Edges:

* DEFINES
* CALLS
* IMPORTS
* DEPENDS_ON
* CONSTRAINED_BY

Backends (pluggable):

* SQLite
* Dolt
* Postgres
* Embedded graph DB

---

## 4.4 Context Cache Layer

Stores:

| Artifact       | Key         |
| -------------- | ----------- |
| File summary   | file_digest |
| Symbol summary | symbol_hash |
| Module summary | module_hash |
| ADR summary    | adr_hash    |

Cache invalidates only when hash changes.

---

## 4.5 ContextPack Generator

Given:

* Task prompt
* Commit SHA

Produces minimal payload:

```json
{
  "commit": "abc123",
  "files": [],
  "symbols": [],
  "snippets": [],
  "adr_constraints": [],
  "rationale_trace": []
}
```

Rules:

* Top-K relevant symbols
* Include only impacted modules
* Inject architectural constraints
* Never exceed configurable token cap

---

# 5. Token Optimization Engine

---

## 5.1 Baseline Model (Estimated)

Estimate cost of full rescan:

```
T_rescan_est =
  (avg_tokens_per_file × total_files)
+ (exploration_overhead × repo_size_factor)
```

Calibrated from cold-start runs.

---

## 5.2 Delta Context Actual

Measured:

* Tokens sent to LLM
* Tokens returned
* Tokens used for summaries

---

## 5.3 Derived Metrics

Per run:

```
TokensSaved = T_rescan_est - T_delta_actual
SavingsPct = TokensSaved / T_rescan_est
CostSaved = TokensSaved × model_token_price
```

---

## 5.4 Telemetry Schema

```json
{
  "run_id": "uuid",
  "repo": "org/repo",
  "base_commit": "abc123",
  "head_commit": "def456",
  "tokens": {
    "baseline_estimate": 182000,
    "delta_actual": 24000,
    "saved": 158000,
    "savings_pct": 0.868
  },
  "cache": {
    "hit_rate": 0.94,
    "files_invalidated": 7,
    "symbols_invalidated": 34
  },
  "latency_ms": 1820
}
```

Exportable to:

* CLI
* MCP server
* Prometheus endpoint

---

# 6. CLI Interface (Go)

```
cs index
cs delta --base abc --head def
cs pack --task "Add feature X"
cs metrics
cs benchmark
```

---

# 7. MCP Server Interface (TypeScript)

Endpoints:

* `context.delta`
* `context.pack`
* `context.metrics`
* `context.invalidate`
* `context.roi`

Agents like Claude Code or Codex call:

```json
{
  "method": "context.pack",
  "params": {
    "commit": "abc123",
    "task": "refactor payment module"
  }
}
```

---

# 8. Benchmarking Mode

Simulate:

1. Cold run (full rescan)
2. Warm run (ContextSubstrate)
3. Multi-commit incremental runs

Outputs:

* Token savings over 30 commits
* Cost savings projection
* Latency reduction

---

# 9. Enterprise Extensions (Phase 2)

* Budget guardrails (max tokens per task)
* Context drift detection
* ADR enforcement injection
* Per-team token efficiency dashboards
* SLA-aware escalation

---

# 10. Success Metrics

| KPI               | Target        |
| ----------------- | ------------- |
| Cache Hit Rate    | >90%          |
| Tokens Saved      | >70%          |
| ContextPack Size  | <25% baseline |
| Latency Reduction | >50%          |

---

# 11. Non-Goals (v1)

* Competing with IDE chat UX tools
* General-purpose RAG
* Memory for conversations (beads-style)

This is **repo lifecycle optimization**, not chat memory.

---

# 12. Differentiation

Unlike:

* Continue
* Aider
* LlamaIndex

ContextSubstrate:

* Is commit-scoped
* Measures token ROI
* Uses incremental invalidation
* Injects architectural constraints
* Produces reproducible ContextPacks

---

# 13. Strategic Positioning

ContextSubstrate becomes:

> The cost-control and determinism layer for AI coding systems.

Not another coding assistant.

---
Below is a **repo-native storage schema** that supports:

* **commit-scoped incremental invalidation**
* **symbol + reference graph**
* **cacheable summaries / embeddings**
* **deterministic ContextPacks**
* **token ROI telemetry**

It’s written as **relational tables + graph edges** (edge tables). Works in SQLite/Postgres/Dolt.

---

# 1) Identity + Versioning

## `repo`

* `repo_id` (PK, text/uuid)
* `name` (text) — `org/repo`
* `default_branch` (text)
* `created_at` (ts)

## `commit`

* `repo_id` (FK)
* `commit_sha` (PK, text)
* `parent_sha` (text, nullable)
* `author` (text)
* `authored_at` (ts)
* `message` (text)

**Index**

* `(repo_id, authored_at)`
* `(repo_id, parent_sha)`

---

# 2) Files + Snapshots (incremental change detection)

## `path`

Stable path identity (useful across renames if you detect them later).

* `path_id` (PK, uuid)
* `repo_id` (FK)
* `path` (text) — `src/foo/bar.ts`
* `first_seen_commit` (text FK->commit.commit_sha)
* `last_seen_commit` (text FK->commit.commit_sha, nullable)

Unique: `(repo_id, path)`

## `file_snapshot`

A file “as of commit” with a digest for invalidation.

* `repo_id` (FK)
* `commit_sha` (FK)
* `path_id` (FK)
* `blob_oid` (text) — git blob id (or content hash)
* `content_sha256` (text) — computed hash (portable beyond git)
* `byte_size` (int)
* `loc` (int)
* `language` (text)
* `is_generated` (bool)
* `is_binary` (bool)

PK: `(repo_id, commit_sha, path_id)`

**Index**

* `(repo_id, path_id, commit_sha)`
* `(repo_id, commit_sha, language)`

### Why both `blob_oid` and `content_sha256`?

* `blob_oid` is fast + aligns with git.
* `content_sha256` is backend-agnostic and stable if you import non-git snapshots.

---

# 3) Symbols (definitions) + Regions (precise snippets)

## `symbol`

Represents a stable “thing” (function/class/type/etc.) within a commit snapshot.

* `repo_id` (FK)
* `commit_sha` (FK)
* `symbol_id` (PK, uuid)
* `kind` (text) — `function|class|method|type|module|const|…`
* `name` (text) — local name
* `fqname` (text) — fully qualified (best-effort)
* `visibility` (text) — `public|private|protected|internal|unknown`
* `language` (text)
* `path_id` (FK)
* `def_region_id` (FK->region.region_id) — definition span
* `signature` (text, nullable)
* `docstring` (text, nullable)
* `symbol_hash` (text) — hash(signature + AST subtree + docstring + kind + fqname)

Unique: `(repo_id, commit_sha, fqname, kind)` (if possible)

**Index**

* `(repo_id, commit_sha, fqname)`
* `(repo_id, commit_sha, name)`
* `(repo_id, commit_sha, path_id)`

## `region`

Reusable address for text spans (for deterministic snippet packaging).

* `repo_id` (FK)
* `commit_sha` (FK)
* `region_id` (PK, uuid)
* `path_id` (FK)
* `start_line` (int)
* `start_col` (int)
* `end_line` (int)
* `end_col` (int)
* `region_hash` (text) — hash of the extracted content (or AST node id)
* `purpose` (text) — `def|callsite|import|adr|snippet|test|…`

**Index**

* `(repo_id, commit_sha, path_id, start_line)`

> Regions let ContextPacks ship *exact* file slices without rescan.

---

# 4) Graph edges (core dependency graph)

All edges are commit-scoped so they can be invalidated by digest changes.

## `edge_symbol_calls`

* `repo_id`
* `commit_sha`
* `from_symbol_id` (FK symbol)
* `to_symbol_id` (FK symbol, nullable if unresolved)
* `to_external_ref` (text, nullable) — e.g., `std::vector` / npm package symbol
* `call_region_id` (FK region)
* `call_type` (text) — `direct|dynamic|reflective|unknown`
* `confidence` (real 0..1)

PK: `(repo_id, commit_sha, from_symbol_id, call_region_id)`

## `edge_symbol_refs`

General references (read/write/type-usage)

* `repo_id`
* `commit_sha`
* `from_symbol_id`
* `to_symbol_id` (nullable)
* `ref_kind` (text) — `reads|writes|type_uses|decorates|extends|implements`
* `ref_region_id` (FK region)
* `confidence` (real)

## `edge_file_imports`

File/module import graph

* `repo_id`
* `commit_sha`
* `from_path_id`
* `to_path_id` (nullable if external)
* `to_external_module` (text, nullable) — `react`, `numpy`, `java.util.*`
* `import_region_id` (FK region)

PK: `(repo_id, commit_sha, from_path_id, import_region_id)`

## `edge_symbol_defined_in_file`

Often derivable, but storing it speeds queries.

* `repo_id`
* `commit_sha`
* `symbol_id`
* `path_id`

PK: `(repo_id, commit_sha, symbol_id)`

---

# 5) ADRs + Architecture Constraints (enterprise hook)

## `adr`

* `repo_id`
* `adr_id` (PK, uuid)
* `title` (text)
* `status` (text) — `proposed|accepted|deprecated|superseded`
* `date` (date, nullable)
* `owner` (text, nullable)
* `source_path_id` (FK path, nullable) — where ADR lives
* `tags` (json/text)

## `adr_snapshot`

ADR content versioned by commit (like file snapshots)

* `repo_id`
* `commit_sha`
* `adr_id`
* `content_sha256`
* `summary_cache_key` (text, nullable)

PK: `(repo_id, commit_sha, adr_id)`

## `adr_rule`

Machine-enforceable constraints (Architecture-as-code)

* `repo_id`
* `adr_id`
* `rule_id` (PK, uuid)
* `rule_type` (text) — `forbid_dependency|require_boundary|layering|naming|security|…`
* `selector` (json) — target set (paths/symbols/tags)
* `constraint` (json) — the rule itself
* `severity` (text) — `warn|error`
* `rationale` (text)

## `edge_adr_constrains`

* `repo_id`
* `commit_sha`
* `adr_id`
* `target_type` (text) — `path|symbol|module`
* `target_id` (uuid/text) — `path_id` or `symbol_id`
* `confidence` (real)

PK: `(repo_id, commit_sha, adr_id, target_type, target_id)`

---

# 6) Cacheable artifacts (summaries, embeddings, “compiled context”)

## `artifact`

Generic cache table for any derived object.

* `artifact_id` (PK, uuid)
* `repo_id`
* `commit_sha`
* `artifact_type` (text) — `file_summary|symbol_summary|module_summary|contextpack|adr_summary|…`
* `scope_type` (text) — `path|symbol|adr|repo|query`
* `scope_id` (text/uuid) — e.g. `path_id` or `symbol_id`
* `content_hash` (text) — hash of input data that produced this artifact
* `model` (text, nullable) — model used to produce summary (if any)
* `created_at` (ts)
* `payload_json` (json/text) — summary text, structured summary, etc.
* `token_count_in` (int, nullable)
* `token_count_out` (int, nullable)

Unique: `(repo_id, commit_sha, artifact_type, scope_type, scope_id, content_hash)`

## `embedding`

(If you do vector search; in SQLite store externally or via extension.)

* `artifact_id` (FK artifact)
* `embedding_model` (text)
* `vector` (blob / vector type)
* `dim` (int)

Unique: `(artifact_id, embedding_model)`

> Key idea: **artifact cache invalidates by `content_hash`**, not by time.

---

# 7) ContextPacks (deterministic, reproducible payloads)

## `contextpack`

* `repo_id`
* `commit_sha`
* `pack_id` (PK, uuid)
* `task_hash` (text) — hash(normalized task prompt + settings)
* `policy_json` (json) — token caps, include_tests, safety, etc.
* `created_at` (ts)
* `estimated_tokens` (int)
* `actual_tokens_sent` (int, nullable)

Unique: `(repo_id, commit_sha, task_hash)`

## `contextpack_item`

Ordered items included in a pack.

* `pack_id` (FK)
* `ordinal` (int)
* `item_type` (text) — `region|symbol|adr|file_summary|module_summary`
* `ref_id` (uuid/text) — region_id / symbol_id / adr_id / artifact_id
* `why` (text) — short rationale (token-minimal)
* `priority` (real)

PK: `(pack_id, ordinal)`

---

# 8) Token ROI telemetry (what makes “token optimization” provable)

## `run`

* `repo_id`
* `run_id` (PK, uuid)
* `base_commit` (text)
* `head_commit` (text)
* `agent` (text) — `claude-code|codex|cursor|custom`
* `task_hash` (text)
* `started_at` (ts)
* `ended_at` (ts, nullable)
* `status` (text) — `ok|fail|cancel`

## `run_metrics`

* `run_id` (PK/FK)
* `baseline_rescan_est_tokens` (int)
* `delta_context_tokens` (int)
* `tokens_saved` (int)
* `savings_pct` (real)
* `cache_hit_rate` (real)
* `files_invalidated` (int)
* `symbols_invalidated` (int)
* `latency_ms` (int)

---

# 9) Minimal graph edge set (if you want v1 smaller)

If you want the smallest set that still delivers token savings:

**Tables you must keep:**

* `repo`, `commit`
* `path`, `file_snapshot`
* `region`, `symbol`
* `edge_file_imports`, `edge_symbol_calls` *(or just imports at first)*
* `artifact` *(summaries)*
* `contextpack`, `contextpack_item`
* `run`, `run_metrics`

That’s enough to do:

* diff → invalidate → reuse summaries → build ContextPack → measure savings.

---

# 10) Query patterns this schema enables

### A) “What changed and what does it impact?”

* diff commits → list changed `path_id`
* join to `edge_file_imports` (reverse) + `edge_symbol_calls` (reverse)
* return impacted symbols/regions

### B) “Build the smallest ContextPack for task X”

* task → retrieve top symbols (via embeddings on `artifact` summaries)
* expand 1–2 hops in edges (imports/calls)
* add ADR rules via `edge_adr_constrains`
* emit ordered `contextpack_item`

### C) Invalidate only what’s necessary

* changed file_snapshot where `content_sha256` differs
* recompute symbols/regions in those files
* update `symbol_hash` and dependent edges

---

If you want, I can also provide:

* **DDL (SQLite/Postgres)** for this schema
* a **“graph materialization” view layer** (views for calls/imports reverse edges)
* the **invalidation algorithm** that updates only affected symbols + edges based on `content_sha256` and `symbol_hash`


Yes — you *can* establish the graph using **JSONL files committed to git**, and it can work surprisingly well for ContextSubstrate’s “commit-scoped, reproducible ContextPacks” goal. The trade is **query performance + incremental maintenance complexity** compared to a DB.

Here’s a practical design that keeps it sane.

---

## When JSONL-in-git is a good fit

* You want **fully portable** artifacts (clone repo → you have the graph)
* You want **auditability** (diffs show graph changes)
* You expect **single-user / small-team** scale, or you can tolerate slower queries
* You primarily need: *“what changed?”, “what depends on it?”, “build contextpack”* — not heavy analytics

When it starts hurting:

* Very large monorepos + frequent updates
* Need sub-second reverse lookups without building indexes
* Multi-writer concurrency, merges, and conflict resolution

---

## Minimal JSONL layout (commit-scoped)

Store under a reserved folder, e.g.:

```
.contextsubstrate/
  manifests/
    commits.jsonl
    paths.jsonl
  snapshots/
    <commit_sha>/
      files.jsonl
      symbols.jsonl
      regions.jsonl
      edges.calls.jsonl
      edges.imports.jsonl
      adrs.jsonl
      artifacts.summaries.jsonl
  packs/
    <commit_sha>/
      <task_hash>.json
  telemetry/
    runs.jsonl
```

### Record types (schema-by-convention)

**commits.jsonl**

```json
{"type":"commit","repo":"org/repo","sha":"abc","parent":"def","authored_at":"...","message":"..."}
```

**paths.jsonl**

```json
{"type":"path","path_id":"uuid","path":"src/foo.ts","first_seen":"abc","last_seen":null}
```

**files.jsonl**

```json
{"type":"file_snapshot","commit":"abc","path_id":"uuid","blob_oid":"...","content_sha256":"...","loc":120,"language":"ts"}
```

**symbols.jsonl**

```json
{"type":"symbol","commit":"abc","symbol_id":"uuid","path_id":"uuid","kind":"function","fqname":"pkg.mod.fn","def_region_id":"uuid","symbol_hash":"..."}
```

**edges.imports.jsonl**

```json
{"type":"edge_file_imports","commit":"abc","from_path_id":"u1","to_path_id":"u2","to_external_module":null,"import_region_id":"r1"}
```

**edges.calls.jsonl**

```json
{"type":"edge_symbol_calls","commit":"abc","from_symbol_id":"s1","to_symbol_id":"s2","to_external_ref":null,"call_region_id":"r9","confidence":0.92}
```

---

## The key problem: reverse edges

JSONL is naturally append-only and forward-scannable. Reverse queries like:

* “who calls this symbol?”
* “who imports this file?”

…require scanning *unless you create indexes*.

### Option A (recommended): build local ephemeral indexes

Keep JSONL as the source of truth in git, but **materialize fast indexes locally** (SQLite/duckdb/rocksdb) on-demand:

* On checkout / commit change:

  * read new/changed JSONL
  * update local index files under `.contextsubstrate/cache/` (gitignored)

This gives you:

* portability + auditability in git
* DB-level query speed locally
* no merge conflicts on index files

### Option B: store reverse edge JSONL too

For each edge stream, also write a reverse stream:

* `edges.calls.rev.jsonl` (to → from)
* `edges.imports.rev.jsonl` (to → from)

This doubles edge storage and makes updates more annoying (must keep forward and reverse consistent), but lets you do faster targeted scans by grepping on `to_symbol_id` / `to_path_id`.

### Option C: shard JSONL by key

Instead of one huge file, shard by prefix:

```
edges/
  calls/
    ab.jsonl   # to_symbol_id starts with "ab"
    ac.jsonl
```

This makes reverse scans cheaper (open only a shard), but increases file count.

---

## Git realities you should plan for

### 1) Size growth

Graph edges can be big. Git *can* store them, but you’ll want:

* **compressed JSONL** (`.jsonl.zst`) or
* keep only “current commit snapshots” + periodic checkpoints

Git won’t diff compressed well, but it will store efficiently.

### 2) Merge conflicts

If multiple people/CI write the same JSONL files, conflicts happen.

Mitigations:

* Make graph generation a **CI artifact** (single writer)
* Or write per-commit folder only (no shared “latest” file)
* Or avoid editing existing lines; append-only per commit is safest

### 3) Determinism

If you want clean diffs and reproducible packs:

* sort edges deterministically
* stable IDs (path_id, symbol_id) derived from hashes (see below)

---

## Stable IDs without a DB

If you want graph to be reproducible across machines:

* `path_id = sha1(repo_id + ":" + path)`
* `symbol_id = sha1(commit_scope? + fqname + kind + def_span + content_hash)`

Two approaches:

1. **Commit-scoped symbols** (simpler): symbol_id changes when definition changes.
2. **Stable symbol identity** (harder): symbol_id tries to remain stable across refactors.

For v1, commit-scoped is totally fine and aligns with invalidation.

---

## Practical recommendation for ContextSubstrate

Use a **hybrid**:

* **JSONL in git** = canonical, portable “Context Graph Snapshot”
* **Local SQLite** (gitignored) = query acceleration / reverse edges / joins

This gives you the best of both worlds:

* enterprise audit + reproducibility
* fast ContextPack generation
* minimal pain with reverse dependencies

---

## What this enables as a product story

“Your context graph is code.”

* it versions with commits
* it’s reviewable in PRs
* it produces deterministic ContextPacks
* token savings can be computed per commit and audited

---

