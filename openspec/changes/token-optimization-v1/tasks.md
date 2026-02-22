## Phase 1: Context Graph Store + Change Detection + Indexing

- [x] 1.1 Define JSONL record types in `internal/graph/records.go`: CommitRecord, PathRecord, FileSnapshot, SymbolRecord, RegionRecord, ImportEdge, CallEdge
- [x] 1.2 Implement JSONL read/write utilities in `internal/graph/jsonl.go`: AppendRecord, ReadRecords (generic), WriteRecords
- [x] 1.3 Implement graph directory initialization in `internal/graph/init.go`: InitGraph creates `.ctx/graph/manifests/` and `.ctx/graph/snapshots/`
- [x] 1.4 Extend `store.InitStore` to call `graph.InitGraph` during store initialization
- [x] 1.5 Implement git change detection in `internal/index/detect.go`: DetectChanges, ListFilesAtCommit, GetCommitInfo, GetHeadSHA, GetRepoRoot
- [x] 1.6 Implement file-level indexing in `internal/index/index.go`: IndexCommit, IndexRange with language detection, content hashing, LOC counting
- [x] 1.7 Implement delta computation in `internal/delta/delta.go`: ComputeDelta compares indexed snapshots, DeltaReport with JSON/Human output
- [x] 1.8 Wire `ctx index` command with `--commit` flag in `cmd/ctx/commands.go`
- [x] 1.9 Wire `ctx delta` command with `--base`, `--head`, `--human` flags in `cmd/ctx/commands.go`
- [x] 1.10 Write unit tests for graph package: JSONL round-trip, directory init, path helpers
- [x] 1.11 Write unit tests for index package: change detection, file listing, commit info, indexing, language detection
- [x] 1.12 Write unit tests for delta package: delta computation, self-delta, JSON/Human output
- [x] 1.13 Write integration test for init → index → delta end-to-end workflow
- [x] 1.14 Create OpenSpec specs for `context-graph`, `change-detection`, `incremental-indexing`

## Phase 2: Symbol Extraction + Cache + Pack Generation

- [x] 2.1 Implement regex-based symbol extraction for Go, TypeScript, Python in `internal/index/symbols.go`
- [x] 2.2 Implement import edge extraction in `internal/index/edges.go`
- [x] 2.3 Implement call edge extraction (grep-based) in `internal/index/edges.go`
- [x] 2.4 Implement cache layer with content-hash invalidation in `internal/cache/cache.go`
- [x] 2.5 Implement pack generator with token budgeting in `internal/optimize/generate.go`
- [x] 2.6 Wire `ctx optimize` command with `--task`, `--commit`, `--token-cap`, `--include-tests`, `--human` flags
- [x] 2.7 Write tests for symbol extraction, cache, pack generation
- [x] 2.8 Create OpenSpec specs for `symbol-graph`, `context-cache`, `optimized-packs`

## Phase 3: Telemetry + Metrics + Benchmarking

- [x] 3.1 Implement run/metrics recording in `internal/telemetry/telemetry.go`: Run, RunMetrics, RecordRun, GetMetrics, GetRuns
- [x] 3.2 Implement baseline estimation model in `internal/telemetry/telemetry.go`: EstimateBaseline
- [x] 3.3 Implement ROI computation in `internal/telemetry/telemetry.go`: ComputeROI, ROISummary
- [x] 3.4 Wire `ctx metrics` and `ctx benchmark` commands with `--limit` and `--commits` flags
- [x] 3.5 Write tests for telemetry: record/retrieve, ROI, formatting, run ID generation
- [x] 3.6 Create OpenSpec specs for `token-telemetry`, `benchmarking`
