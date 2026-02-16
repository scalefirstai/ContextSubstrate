Perfect — here’s a **tight, implementation-ready spec** for **ContextSubstrate**, scoped **only** to the core 5 features we identified. No fluff, no platform creep.

---

# ContextSubstrate — Core Spec (v0.1)

## Purpose

ContextSubstrate is an **execution substrate for AI agents** that makes their work **reproducible, debuggable, and contestable**, using artifacts developers already trust (files, hashes, diffs, CLI workflows).

---

## Core Feature 1: Context Packs (Reproducible Execution Units)

### Description

A **Context Pack** is an immutable, versioned bundle that fully captures an agent run.

### Includes

* Prompt(s) and system instructions
* Input files (or content-addressed references)
* Tool calls + parameters
* Model identifier + parameters
* Execution order + timestamps
* Environment metadata (OS, runtime, tool versions)

### Guarantees

* Content-addressable (hash-based)
* Immutable once finalized
* Portable across machines

### Output

```text
ctx://<sha256>
```

---

## Core Feature 2: Deterministic Replay

### Description

Ability to **re-run an agent execution exactly as it originally occurred**.

### Requirements

* Same inputs + tool versions must yield same intermediate steps
* Non-deterministic steps must be explicitly flagged
* Replay failures must identify *which dependency drifted*

### CLI

```bash
ctx replay <context-hash>
```

### Output

* Re-executed run
* Replay status: exact | degraded | failed

---

## Core Feature 3: Decision Drift Detection

### Description

Compare two Context Packs and identify **where and why decisions diverged**.

### Drift Types

* Prompt drift
* Tool choice drift
* Parameter drift
* Intermediate reasoning divergence
* Output divergence

### CLI

```bash
ctx diff <ctx-hash-a> <ctx-hash-b>
```

### Output

* Structured diff (machine-readable)
* Human-readable summary of *decision points that changed*

---

## Core Feature 4: Contestable Outputs (Traceable Artifacts)

### Description

Every produced artifact must be **traceable back to the context that produced it**.

### Artifact Metadata

* Context Pack hash
* Supporting inputs
* Tools involved
* Confidence / uncertainty notes (optional but first-class)

### Verification

```bash
ctx verify <artifact>
```

### Guarantee

> Any output can be challenged, reviewed, or replayed.

---

## Core Feature 5: Context Hashes as a Shareable Primitive

### Description

Context Packs are referenced and shared **by hash**, not by environment or platform.

### Properties

* Shareable across teams
* Forkable (derive a new pack from an existing one)
* Comparable (via diff)

### CLI

```bash
ctx fork <context-hash>
```

### Social Contract

> “If you can share the hash, you can reproduce the work.”

---

## Non-Goals (Explicit)

* No agent orchestration
* No prompt IDE
* No vector search
* No cloud dependency
* No opinionated agent framework

ContextSubstrate is **infrastructure**, not a workflow.

---

## Initial Implementation Targets

* **CLI**: Go
* **Pack format**: JSON + content-addressed blobs
* **Storage**: local filesystem (Git-compatible layout)
* **Extensibility**: optional MCP adapter later (out of scope for v0.1)

---

## Success Criteria

* A developer can reproduce an agent run from last week
* Two agent runs can be meaningfully diffed
* An output can be challenged with evidence
* Context can be shared without sharing a repo or environment

---

If you want next:

* I can turn this into a **GitHub README**
* Or a **CLI command reference**
* Or a **“Why this exists”** section tuned for Reddit / Hacker News

Just say the word.
