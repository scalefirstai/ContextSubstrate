package pack

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// ExecutionLog represents the raw input format before conversion to a Pack manifest.
type ExecutionLog struct {
	Model        LogModel        `json:"model"`
	SystemPrompt string          `json:"system_prompt"`
	Prompts      []LogPrompt     `json:"prompts"`
	Inputs       []LogInput      `json:"inputs"`
	Steps        []LogStep       `json:"steps"`
	Outputs      []LogOutput     `json:"outputs"`
	Environment  LogEnvironment  `json:"environment"`
}

type LogModel struct {
	Identifier string                 `json:"identifier"`
	Parameters map[string]interface{} `json:"parameters"`
}

type LogPrompt struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type LogInput struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type LogStep struct {
	Index         int                    `json:"index"`
	Type          string                 `json:"type"`
	Tool          string                 `json:"tool"`
	Parameters    map[string]interface{} `json:"parameters"`
	Output        string                 `json:"output"`
	Deterministic bool                   `json:"deterministic"`
	Timestamp     time.Time              `json:"timestamp"`
}

type LogOutput struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

type LogEnvironment struct {
	OS           string            `json:"os"`
	Runtime      string            `json:"runtime"`
	ToolVersions map[string]string `json:"tool_versions"`
}

// ParseExecutionLog reads and parses a JSON execution log from a file path.
func ParseExecutionLog(path string) (*ExecutionLog, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening log file: %w", err)
	}
	defer f.Close()
	return ParseExecutionLogReader(f)
}

// ParseExecutionLogReader reads and parses a JSON execution log from a reader.
func ParseExecutionLogReader(r io.Reader) (*ExecutionLog, error) {
	var log ExecutionLog
	decoder := json.NewDecoder(r)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&log); err != nil {
		return nil, fmt.Errorf("parsing execution log: %w", err)
	}

	if err := validateLog(&log); err != nil {
		return nil, err
	}

	return &log, nil
}

func validateLog(log *ExecutionLog) error {
	var missing []string

	if log.Model.Identifier == "" {
		missing = append(missing, "model.identifier")
	}
	if log.SystemPrompt == "" {
		missing = append(missing, "system_prompt")
	}
	if log.Environment.OS == "" {
		missing = append(missing, "environment.os")
	}
	if log.Environment.Runtime == "" {
		missing = append(missing, "environment.runtime")
	}

	for i, step := range log.Steps {
		if step.Tool == "" {
			missing = append(missing, fmt.Sprintf("steps[%d].tool", i))
		}
		if step.Type == "" {
			missing = append(missing, fmt.Sprintf("steps[%d].type", i))
		}
	}

	for i, output := range log.Outputs {
		if output.Name == "" {
			missing = append(missing, fmt.Sprintf("outputs[%d].name", i))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("invalid execution log: missing required fields: %v", missing)
	}
	return nil
}
