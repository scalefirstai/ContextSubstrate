## 1. Project Setup

- [x] 1.1 Initialize Go module (`go mod init`) and set up directory structure: `cmd/ctx/`, `internal/store/`, `internal/pack/`, `internal/replay/`, `internal/diff/`, `internal/verify/`, `internal/sharing/`
- [x] 1.2 Add Cobra dependency and create root command (`cmd/ctx/main.go`, `cmd/ctx/root.go`) with version flag
- [x] 1.3 Create stub subcommands for all CLI commands: `init`, `pack`, `show`, `replay`, `diff`, `verify`, `fork`, `log`

## 2. Object Store and Hashing

- [x] 2.1 Implement SHA-256 content hashing with `sha256:` prefix format (`internal/store/hash.go`)
- [x] 2.2 Implement blob storage: write content to `.ctx/objects/<2-char-prefix>/<rest-of-hash>` with deduplication (skip if exists) and immutability (reject overwrite) (`internal/store/blob.go`)
- [x] 2.3 Implement blob retrieval: read by hash, verify integrity on read (`internal/store/blob.go`)
- [x] 2.4 Implement `ctx init` command: create `.ctx/` directory with `objects/`, `packs/`, `refs/` subdirectories and `config.json`; error if already exists (`internal/store/init.go`)
- [x] 2.5 Write tests for hashing, blob storage, blob retrieval, and init

## 3. Context Pack Creation

- [x] 3.1 Define Go structs for pack manifest: `Pack`, `Model`, `Prompt`, `Input`, `Step`, `Output`, `Environment` matching the JSON schema from design (`internal/pack/manifest.go`)
- [x] 3.2 Implement execution log parser: read JSON log file, validate required fields, produce error listing missing/invalid fields (`internal/pack/parse.go`)
- [x] 3.3 Implement pack creation: extract content from log, store blobs, build manifest with content refs, compute canonical manifest hash (sorted keys), store manifest as blob (`internal/pack/create.go`)
- [x] 3.4 Implement `ctx pack <log-file>` command: parse log, create pack, register in `.ctx/packs/`, output `ctx://<sha256>` URI
- [x] 3.5 Implement `ctx show <hash>` command: load manifest, resolve blob summaries (name, size), print human-readable output; error if hash not found
- [x] 3.6 Write tests for log parsing (valid and invalid), pack creation, manifest hashing determinism, and show command

## 4. Deterministic Replay

- [x] 4.1 Define replay report structs: `ReplayReport`, `StepResult` (matched/diverged/failed), fidelity level enum (exact/degraded/failed) (`internal/replay/report.go`)
- [x] 4.2 Implement step executor: re-execute a single tool call, capture output, hash and compare against recorded output ref (`internal/replay/executor.go`)
- [x] 4.3 Implement replay orchestrator: walk steps in order, handle deterministic vs non-deterministic steps, accumulate fidelity report, detect dependency drift (tool version, missing input) (`internal/replay/replay.go`)
- [x] 4.4 Implement `ctx replay <hash>` command: load pack, run orchestrator, print fidelity report with step details; error if hash not found
- [x] 4.5 Write tests for replay with exact match, degraded fidelity, failed step, non-deterministic step handling, and drift identification

## 5. Drift Detection

- [x] 5.1 Define drift types and report structs: `DriftEntry` with type enum (prompt_drift, tool_drift, param_drift, reasoning_drift, output_drift), `DriftReport` (`internal/diff/types.go`)
- [x] 5.2 Implement prompt comparison: compare system_prompt refs and prompts arrays between two packs (`internal/diff/prompt.go`)
- [x] 5.3 Implement step comparison: align by index, detect tool drift, param drift, reasoning drift; handle mismatched step counts as additions/removals (`internal/diff/steps.go`)
- [x] 5.4 Implement output comparison: compare output arrays between two packs (`internal/diff/output.go`)
- [x] 5.5 Implement `ctx diff <a> <b>` command: load both packs, run all comparisons, output JSON drift report by default; support `--human` flag for plain-text summary; error if hash not found
- [x] 5.6 Write tests for each drift type, identical packs (no drift), mismatched step counts, JSON output format, and human-readable output

## 6. Contestable Outputs

- [x] 6.1 Define sidecar metadata struct: `context_pack`, `inputs`, `tools`, `confidence`, `notes` fields (`internal/verify/sidecar.go`)
- [x] 6.2 Implement sidecar generation: given a pack and output artifact, create `<artifact>.ctx.json` sidecar file (`internal/verify/sidecar.go`)
- [x] 6.3 Implement `ctx verify <artifact>` command: locate sidecar, validate pack reference exists in store, hash artifact content and compare against pack's output `content_ref`, display provenance info with pack hash for replay/show; handle missing sidecar and broken provenance
- [x] 6.4 Add sidecar generation to `ctx pack` command: after packing, generate sidecar files for each output artifact listed in the manifest
- [x] 6.5 Write tests for sidecar generation, verify with valid/missing/broken provenance, content integrity check (match and mismatch), and confidence metadata display

## 7. Context Sharing

- [x] 7.1 Implement `ctx fork <hash>` command: load existing pack, create mutable draft copy with `parent` field set to source hash, write draft to staging area; error if hash not found (`internal/sharing/fork.go`)
- [x] 7.2 Implement draft finalization: finalize a forked draft into a new immutable pack with lineage preserved in `parent` field
- [x] 7.3 Implement `ctx log` command: scan `.ctx/packs/`, load each manifest, display hash/date/model/step-count sorted by creation date newest-first; handle empty store (`internal/sharing/log.go`)
- [x] 7.4 Write tests for fork (valid, non-existent), draft finalization with lineage, log listing (populated and empty stores), and manifest portability (no absolute paths)

## 8. Integration and CLI Polish

- [x] 8.1 Add store discovery: walk up directory tree to find nearest `.ctx/` (similar to how Git finds `.git/`); all commands except `init` should use this
- [x] 8.2 Add consistent error formatting across all commands (non-zero exit codes, human-readable error messages)
- [x] 8.3 Write end-to-end integration test: init → pack → show → replay → diff → verify → fork → log
- [x] 8.4 Add shell completion support via Cobra's built-in completion generation
