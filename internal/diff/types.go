package diff

import (
	"encoding/json"
	"fmt"
	"strings"
)

type DriftType string

const (
	PromptDrift    DriftType = "prompt_drift"
	ToolDrift      DriftType = "tool_drift"
	ParamDrift     DriftType = "param_drift"
	ReasoningDrift DriftType = "reasoning_drift"
	OutputDrift    DriftType = "output_drift"
)

type DriftEntry struct {
	Type        DriftType   `json:"type"`
	Description string      `json:"description"`
	StepIndex   int         `json:"step_index,omitempty"`
	PackA       interface{} `json:"pack_a,omitempty"`
	PackB       interface{} `json:"pack_b,omitempty"`
}

type DriftReport struct {
	PackHashA string       `json:"pack_hash_a"`
	PackHashB string       `json:"pack_hash_b"`
	Entries   []DriftEntry `json:"entries"`
	HasDrift  bool         `json:"has_drift"`
}

// JSON returns the report as JSON bytes.
func (r *DriftReport) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// Human returns a human-readable summary of the drift report.
func (r *DriftReport) Human() string {
	if !r.HasDrift {
		return "No differences found.\n"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Comparing %s vs %s\n\n", r.PackHashA, r.PackHashB))
	b.WriteString(fmt.Sprintf("%d difference(s) found:\n\n", len(r.Entries)))

	for i, e := range r.Entries {
		b.WriteString(fmt.Sprintf("  %d. [%s] %s\n", i+1, e.Type, e.Description))
	}

	return b.String()
}
