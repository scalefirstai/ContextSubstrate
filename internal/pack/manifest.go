package pack

import (
	"fmt"
	"time"
)

type Pack struct {
	Version      string      `json:"version"`
	Hash         string      `json:"hash"`
	Created      time.Time   `json:"created"`
	Model        Model       `json:"model"`
	SystemPrompt string      `json:"system_prompt"`
	Prompts      []Prompt    `json:"prompts"`
	Inputs       []Input     `json:"inputs"`
	Steps        []Step      `json:"steps"`
	Outputs      []Output    `json:"outputs"`
	Environment  Environment `json:"environment"`
	Parent       string      `json:"parent,omitempty"`
}

type Model struct {
	Identifier string                 `json:"identifier"`
	Parameters map[string]interface{} `json:"parameters"`
}

type Prompt struct {
	Role       string `json:"role"`
	ContentRef string `json:"content_ref"`
}

type Input struct {
	Name       string `json:"name"`
	ContentRef string `json:"content_ref"`
	Size       int64  `json:"size"`
}

type Step struct {
	Index         int                    `json:"index"`
	Type          string                 `json:"type"`
	Tool          string                 `json:"tool"`
	Parameters    map[string]interface{} `json:"parameters"`
	OutputRef     string                 `json:"output_ref"`
	Deterministic bool                   `json:"deterministic"`
	Timestamp     time.Time              `json:"timestamp"`
}

type Output struct {
	Name        string `json:"name"`
	ContentRef  string `json:"content_ref"`
	ContextPack string `json:"context_pack,omitempty"`
}

type Environment struct {
	OS           string            `json:"os"`
	Runtime      string            `json:"runtime"`
	ToolVersions map[string]string `json:"tool_versions"`
}

// Validate checks that all required fields are present in the pack manifest.
func (p *Pack) Validate() error {
	var missing []string

	if p.Version == "" {
		missing = append(missing, "version")
	}
	if p.Created.IsZero() {
		missing = append(missing, "created")
	}
	if p.Model.Identifier == "" {
		missing = append(missing, "model.identifier")
	}
	if p.SystemPrompt == "" {
		missing = append(missing, "system_prompt")
	}
	if p.Environment.OS == "" {
		missing = append(missing, "environment.os")
	}
	if p.Environment.Runtime == "" {
		missing = append(missing, "environment.runtime")
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required fields: %v", missing)
	}
	return nil
}
