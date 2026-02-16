## Context

ContextSubstrate is a greenfield Go CLI tool (`ctx`) that provides reproducibility, debugging, and contestability infrastructure for AI agent execution. There is no existing codebase — this design covers the foundational architecture for all five core capabilities: context packs, replay, drift detection, contestable outputs, and context sharing.

The primary constraint is that the system must work entirely locally with no cloud dependencies, use a Git-compatible filesystem layout, and rely only on the Go standard library plus minimal dependencies.

## Goals / Non-Goals

**Goals:**

- Define a storage layout that is Git-friendly and content-addressable
- Establish the Context Pack format as the central data structure everything else builds on
- Design a CLI structure that maps cleanly to the five capabilities
- Keep the architecture simple enough for a single developer to implement incrementally

**Non-Goals:**

- Agent orchestration or execution — ContextSubstrate only captures and replays, it does not run agents
- Cloud storage, remote sync, or network protocols
- Plugin/extension system (MCP adapter is explicitly out of scope for v0.1)
- GUI or web interface
- Opinionated agent framework integration

## Decisions

### 1. Storage Layout: Git-like Object Store

**Decision**: Use a `.ctx/` directory with a content-addressed object store, similar to Git's `.git/objects/` layout.

```
.ctx/
  objects/          # content-addressed blobs (SHA-256, first 2 chars as subdir)
    ab/
      cdef1234...   # blob content
  packs/            # symlinks or index pointing to pack manifest objects
  refs/             # named references (latest, tags)
  config.json       # local configuration
```

**Rationale**: Developers already understand Git's layout. Content-addressed storage gives us deduplication for free — identical inputs across runs share the same objects. The 2-char prefix subdirectory prevents filesystem issues with large numbers of files.

**Alternatives considered**:
- SQLite database: Simpler queries but less transparent, harder to inspect manually, not Git-friendly
- Flat directory of JSON files: Simpler but no deduplication, doesn't scale

### 2. Context Pack Format: JSON Manifest + Blob References

**Decision**: A Context Pack is a JSON manifest file that references content-addressed blobs. The manifest itself is also content-addressed.

```json
{
  "version": "0.1",
  "hash": "sha256:...",
  "created": "2026-02-10T...",
  "model": {
    "identifier": "claude-opus-4-6",
    "parameters": { "temperature": 0, "max_tokens": 4096 }
  },
  "system_prompt": "sha256:...",
  "prompts": [
    { "role": "user", "content_ref": "sha256:..." }
  ],
  "inputs": [
    { "name": "file.py", "content_ref": "sha256:...", "size": 1234 }
  ],
  "steps": [
    {
      "index": 0,
      "type": "tool_call",
      "tool": "read_file",
      "parameters": { "path": "src/main.py" },
      "output_ref": "sha256:...",
      "deterministic": true,
      "timestamp": "2026-02-10T..."
    }
  ],
  "outputs": [
    { "name": "result.md", "content_ref": "sha256:...", "context_pack": "sha256:..." }
  ],
  "environment": {
    "os": "darwin",
    "runtime": "go1.22",
    "tool_versions": { "ctx": "0.1.0" }
  }
}
```

**Rationale**: JSON is human-readable and inspectable. Blob references keep the manifest small while large content (files, outputs) is stored once in the object store. The manifest hash is computed over the canonical JSON (sorted keys, no extra whitespace).

**Alternatives considered**:
- Protobuf: More compact but not human-readable, adds a build dependency
- YAML: Ambiguous parsing semantics, less suitable for canonical hashing
- Tar/zip bundle: Harder to inspect, harder to deduplicate across packs

### 3. CLI Framework: Cobra

**Decision**: Use `github.com/spf13/cobra` for CLI structure.

**Rationale**: Cobra is the de facto standard for Go CLIs (used by kubectl, gh, docker). Provides subcommand routing, flag parsing, help generation, and shell completions out of the box. The team won't need to learn a custom framework.

**Alternatives considered**:
- `flag` stdlib: Too basic for a multi-subcommand CLI, no built-in help generation
- `urfave/cli`: Viable but less ecosystem adoption than cobra

### 4. CLI Command Structure

**Decision**: Top-level `ctx` binary with subcommands mapping to capabilities:

| Command | Capability | Description |
|---|---|---|
| `ctx pack` | context-packs | Create/finalize a context pack |
| `ctx show <hash>` | context-packs | Inspect a pack's contents |
| `ctx replay <hash>` | deterministic-replay | Re-execute a captured run |
| `ctx diff <a> <b>` | drift-detection | Compare two packs |
| `ctx verify <artifact>` | contestable-outputs | Verify artifact provenance |
| `ctx fork <hash>` | context-sharing | Derive a new pack from existing |
| `ctx log` | context-sharing | List packs (like `git log`) |
| `ctx init` | — | Initialize `.ctx/` in current directory |

**Rationale**: Each command maps to exactly one capability. Verbs are familiar to Git users (`diff`, `log`, `show`, `init`).

### 5. Replay Execution Model: Step-by-Step with Drift Reporting

**Decision**: Replay walks the `steps` array in order, re-executing each tool call. After each step, compare the actual output hash against the recorded output hash. Accumulate a fidelity report.

Replay fidelity levels:
- **exact**: All step outputs match recorded hashes
- **degraded**: Some steps diverged but execution completed (e.g., timestamps differ, formatting changed)
- **failed**: A step could not execute (missing tool, missing input, error)

Non-deterministic steps (marked `"deterministic": false`) are expected to diverge and do not downgrade fidelity.

**Rationale**: Step-by-step comparison gives precise drift localization. The three-level fidelity model is simple enough to be useful without requiring complex scoring.

**Alternatives considered**:
- Output-only comparison (just check final result): Loses the ability to identify *where* drift occurred
- Probabilistic matching (fuzzy diff): Too complex for v0.1, can be added later

### 6. Artifact Provenance: Sidecar Metadata Files

**Decision**: When an agent produces an artifact, provenance is stored in a `.ctx.json` sidecar file alongside the artifact (e.g., `result.md` gets `result.md.ctx.json`).

```json
{
  "context_pack": "sha256:...",
  "inputs": ["sha256:...", "sha256:..."],
  "tools": ["read_file", "write_file"],
  "confidence": "high",
  "notes": "Generated from template with no manual edits"
}
```

**Rationale**: Sidecar files don't modify the original artifact, are easy to `.gitignore` if unwanted, and can be discovered by convention. Embedding metadata in the artifact itself would require format-specific injection (different for .md, .py, .json, etc.).

**Alternatives considered**:
- Embedded metadata (comments/headers in artifact): Format-dependent, may break parsers
- Central registry file: Single point of failure, merge conflicts in teams

### 7. Diff Algorithm: Structured Section Comparison

**Decision**: `ctx diff` compares two pack manifests section by section, producing typed drift entries:

- **prompt_drift**: System prompt or user prompts differ
- **tool_drift**: Different tools called or different call order
- **param_drift**: Same tool called with different parameters
- **reasoning_drift**: Intermediate step outputs diverge
- **output_drift**: Final outputs differ

Output format: JSON (machine-readable) by default, `--human` flag for readable summary.

**Rationale**: Typed drift categories make it actionable — a developer can immediately see "the prompt changed" vs. "the model chose a different tool." Section-by-section comparison is straightforward to implement.

### 8. Hashing: SHA-256 with `sha256:` Prefix

**Decision**: All content addressing uses SHA-256. Hashes are represented as `sha256:<hex>` strings. The `ctx://` URI scheme is `ctx://<sha256-hex>` (omitting the prefix for brevity in URIs).

**Rationale**: SHA-256 is the standard for content addressing (Docker, Git SHA-256, OCI). The prefix allows future algorithm migration without ambiguity.

## Risks / Trade-offs

**[Large packs with many files]** → Content-addressed deduplication mitigates storage growth. For v0.1, accept that very large runs (thousands of files) may be slow to pack. Optimize later with parallel hashing.

**[Replay requires tool availability]** → If a tool used in the original run isn't available during replay, replay fails. Mitigation: clear error messages identifying which tool is missing. Future: tool version pinning / containerized replay.

**[Non-deterministic LLM outputs]** → LLM calls are inherently non-deterministic even with temperature=0. Mitigation: mark LLM steps as `"deterministic": false` so replay expects divergence. This is a fundamental constraint, not a bug.

**[JSON canonical hashing]** → JSON serialization must be deterministic for hash stability. Mitigation: use sorted keys and no trailing whitespace. Go's `encoding/json` with `json.Marshal` on sorted struct fields provides this.

**[Sidecar file proliferation]** → Every artifact gets a `.ctx.json` file. Mitigation: these are small, can be `.gitignore`d, and `ctx verify` can discover them by convention. Not a blocker for v0.1.

## Open Questions

- Should `ctx pack` capture execution interactively (watching an agent run) or accept a completed execution log as input? For v0.1, likely log-based input is simpler.
- Should the `.ctx/` directory live at project root or in a user-global location (`~/.ctx/`)? Leaning project-local for Git compatibility, with `ctx init` establishing it.
- What is the minimum viable diff output for `--human` mode? Plain text summary vs. colorized terminal output.
