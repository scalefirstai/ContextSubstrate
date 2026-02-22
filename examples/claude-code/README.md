# Capturing Claude Code with ctx

## Overview

[Claude Code](https://docs.anthropic.com/en/docs/claude-code) is Anthropic's agentic coding tool that operates directly in your terminal. It can read and edit files, run shell commands, search codebases, and iteratively fix bugs or build features. A typical Claude Code session involves multiple rounds of file reading, code editing, and test execution.

This example demonstrates how to capture a Claude Code coding session as an immutable, content-addressed context pack using `ctx`. The included `execution.json` records a session where Claude Code diagnoses and fixes a bug in a Go HTTP handler, then verifies the fix by running tests.

## Prerequisites

- `ctx` CLI installed and on your PATH
- Claude Code installed (`npm install -g @anthropic-ai/claude-code`)
- A project with source code and tests

## Capturing the Execution Log

Claude Code can output a structured log of its session when run in non-interactive mode. To capture an execution log suitable for `ctx`:

```bash
# Run Claude Code with JSON output logging
claude -p "Fix the nil pointer panic in handlers/user.go" \
  --output-format json \
  > execution.json
```

Alternatively, you can use the Claude Code SDK to programmatically capture sessions:

```javascript
import { claude } from "@anthropic-ai/claude-code";

const result = await claude({
  prompt: "Fix the nil pointer panic in handlers/user.go",
  options: { outputFormat: "json" }
});

// Transform result into ctx execution log format
// and write to execution.json
```

The resulting JSON must conform to the `ctx` execution log schema. See the included `execution.json` for the expected structure.

## Packing and Inspecting

Pack the captured session:

```bash
ctx pack claude-code-bugfix execution.json
```

Inspect the pack to review the full session:

```bash
ctx inspect claude-code-bugfix
```

This shows the content hash, model used, files read and written, tools invoked (read_file, edit_file, bash), and the final outputs.

## Replaying

Replay the captured session to verify deterministic steps:

```bash
ctx replay claude-code-bugfix
```

In a Claude Code session, several steps are deterministic and can be verified during replay:

- **File reads** should return the same content (assuming the repository is at the same commit).
- **File edits** apply known diffs and produce predictable results.
- **Test runs** on deterministic code should produce the same pass/fail results.

Non-deterministic steps (the LLM reasoning that decides which files to read or what edits to make) are recorded but not re-executed during replay.

## Comparing Runs

Compare two sessions that tackle the same bug or feature to understand different approaches:

```bash
ctx diff claude-code-bugfix claude-code-bugfix-v2
```

This is valuable for understanding how different prompts, models, or code states lead to different fix strategies. The diff shows which files were read, what edits were applied, and whether tests passed in each run.
