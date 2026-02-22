# Capturing CrewAI with ctx

## Overview

[CrewAI](https://github.com/crewAIInc/crewAI) is a framework for orchestrating autonomous AI agents that collaborate on tasks. A "crew" consists of multiple agents, each with a defined role, goal, and set of tools. CrewAI coordinates these agents to complete complex workflows such as research, content creation, and data analysis.

This example demonstrates how to capture a CrewAI multi-agent workflow as a context pack using `ctx`. The included `execution.json` records a two-agent crew (a Researcher and a Writer) collaborating on a content creation task.

## Prerequisites

- `ctx` CLI installed and on your PATH
- Python 3.10+ with `crewai` installed
- An OpenAI API key (for CrewAI agent execution)

## Capturing the Execution Log

CrewAI exposes callbacks and event hooks that allow you to intercept agent actions. To produce a `ctx`-compatible execution log, attach a logging callback to your crew:

```python
from crewai import Agent, Task, Crew, Process
import json, datetime

execution_log = {
    "model": {"identifier": "gpt-4", "parameters": {"temperature": 0.0}},
    "system_prompt": "",
    "prompts": [],
    "inputs": [],
    "steps": [],
    "outputs": [],
    "environment": {}
}

step_index = 0

def log_step(agent_name, tool_name, params, output, deterministic=False):
    global step_index
    execution_log["steps"].append({
        "index": step_index,
        "type": "tool_call",
        "tool": tool_name,
        "parameters": {"agent": agent_name, **params},
        "output": output,
        "deterministic": deterministic,
        "timestamp": datetime.datetime.utcnow().isoformat() + "Z"
    })
    step_index += 1

# Wire log_step into your custom tools and crew callbacks
# After the crew finishes, serialize execution_log to execution.json
```

The included `execution.json` shows a complete captured workflow.

## Packing and Inspecting

Pack the captured execution log:

```bash
ctx pack crewai-content-workflow execution.json
```

Inspect the pack to review its contents:

```bash
ctx inspect crewai-content-workflow
```

This displays the content hash, the full sequence of tool calls across all agents, and the final outputs.

## Replaying

Replay deterministic steps in the captured workflow:

```bash
ctx replay crewai-content-workflow
```

In a multi-agent workflow, non-deterministic steps (LLM reasoning, live web searches) are skipped during replay. Deterministic steps such as file writes and template rendering are re-executed and verified against recorded outputs.

## Comparing Runs

Compare two runs of the same crew to understand how changes affect output:

```bash
ctx diff crewai-content-workflow crewai-content-workflow-v2
```

This is particularly useful for CrewAI workflows because you can see how changing an agent's role description, goal, or tool set affects the overall crew output. The diff highlights differences in tool call sequences, intermediate results, and final outputs.
