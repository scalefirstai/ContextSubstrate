package replay

import (
	"fmt"
	"os"

	"github.com/contextsubstrate/ctx/internal/pack"
	"github.com/contextsubstrate/ctx/internal/store"
)

// ToolExecutor executes a tool call and returns the output.
type ToolExecutor func(tool string, params map[string]interface{}) ([]byte, error)

// DefaultExecutors returns the built-in tool executors for v0.1.
func DefaultExecutors() map[string]ToolExecutor {
	return map[string]ToolExecutor{
		"read_file": func(tool string, params map[string]interface{}) ([]byte, error) {
			path, ok := params["path"].(string)
			if !ok {
				return nil, fmt.Errorf("read_file: missing or invalid 'path' parameter")
			}
			return os.ReadFile(path)
		},
	}
}

// ExecuteStep re-executes a single tool call and compares the output.
func ExecuteStep(storeRoot string, step *pack.Step, executors map[string]ToolExecutor) *StepResult {
	result := &StepResult{
		Index:         step.Index,
		Tool:          step.Tool,
		Deterministic: step.Deterministic,
		ExpectedHash:  step.OutputRef,
	}

	executor, ok := executors[step.Tool]
	if !ok {
		result.Status = StepFailed
		result.Reason = fmt.Sprintf("tool not available: %s", step.Tool)
		return result
	}

	output, err := executor(step.Tool, step.Parameters)
	if err != nil {
		result.Status = StepFailed
		result.Reason = fmt.Sprintf("execution error: %v", err)
		return result
	}

	actualHash := store.HashContent(output)
	result.ActualHash = actualHash

	if actualHash == step.OutputRef {
		result.Status = StepMatched
	} else {
		result.Status = StepDiverged
	}

	return result
}
