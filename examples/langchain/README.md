# Capturing LangChain with ctx

## Overview

[LangChain](https://github.com/langchain-ai/langchain) is a framework for building applications powered by large language models. LangChain agents can use tools (search engines, file readers, APIs) to complete multi-step tasks autonomously.

This example demonstrates how to capture a LangChain agent execution as an immutable, content-addressed context pack using `ctx`. The included `execution.json` contains a realistic log of an agent performing a research task: searching the web, reading sources, and summarizing findings into a report.

## Prerequisites

- `ctx` CLI installed and on your PATH
- Python 3.10+ with `langchain` and `langchain-openai` installed
- An OpenAI API key (for the agent itself; `ctx` does not require one)

## Capturing the Execution Log

LangChain provides callback hooks that let you intercept every step of an agent run. To produce a JSON execution log compatible with `ctx`, instrument your agent with a custom callback handler that records tool calls, inputs, outputs, and timestamps.

A minimal approach:

```python
from langchain.agents import AgentExecutor, create_openai_tools_agent
from langchain_openai import ChatOpenAI
from langchain.callbacks.base import BaseCallbackHandler
import json, datetime

class CtxLogger(BaseCallbackHandler):
    def __init__(self):
        self.steps = []
        self.index = 0

    def on_tool_start(self, serialized, input_str, **kwargs):
        self.current_tool = serialized.get("name", "unknown")
        self.current_input = input_str

    def on_tool_end(self, output, **kwargs):
        self.steps.append({
            "index": self.index,
            "type": "tool_call",
            "tool": self.current_tool,
            "parameters": {"input": self.current_input},
            "output": output,
            "deterministic": False,
            "timestamp": datetime.datetime.utcnow().isoformat() + "Z"
        })
        self.index += 1

logger = CtxLogger()
# Pass `callbacks=[logger]` when invoking your agent
# After the run, write logger.steps into the full execution log JSON
```

The included `execution.json` file shows the complete schema that `ctx pack` expects.

## Packing and Inspecting

Once you have an execution log, pack it into an immutable context pack:

```bash
ctx pack langchain-research-run execution.json
```

Inspect the resulting pack:

```bash
ctx inspect langchain-research-run
```

This prints the content hash, model parameters, number of steps, inputs, and outputs recorded in the pack.

## Replaying

Replay the captured execution to verify that deterministic steps produce the same outputs:

```bash
ctx replay langchain-research-run
```

The replay engine re-executes deterministic tool calls and compares their outputs against the recorded values. Non-deterministic steps (like LLM generations or live web searches) are skipped by default.

## Comparing Runs

If you capture multiple runs of the same agent task (for example, after changing the prompt or model temperature), you can diff them:

```bash
ctx diff langchain-research-run langchain-research-run-v2
```

This shows a structured comparison of inputs, outputs, tool call sequences, and any divergences between the two runs. It is useful for prompt regression testing and evaluating the impact of model parameter changes.
