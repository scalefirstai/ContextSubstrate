package replay

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type FidelityLevel string

const (
	FidelityExact    FidelityLevel = "exact"
	FidelityDegraded FidelityLevel = "degraded"
	FidelityFailed   FidelityLevel = "failed"
)

type StepStatus string

const (
	StepMatched  StepStatus = "matched"
	StepDiverged StepStatus = "diverged"
	StepFailed   StepStatus = "failed"
)

type StepResult struct {
	Index         int        `json:"index"`
	Tool          string     `json:"tool"`
	Status        StepStatus `json:"status"`
	ExpectedHash  string     `json:"expected_hash,omitempty"`
	ActualHash    string     `json:"actual_hash,omitempty"`
	Deterministic bool       `json:"deterministic"`
	Reason        string     `json:"reason,omitempty"`
}

type DriftEntry struct {
	Type        string `json:"type"`
	Description string `json:"description"`
	Expected    string `json:"expected,omitempty"`
	Actual      string `json:"actual,omitempty"`
}

type ReplayReport struct {
	PackHash  string        `json:"pack_hash"`
	Fidelity  FidelityLevel `json:"fidelity"`
	Steps     []StepResult  `json:"steps"`
	Drift     []DriftEntry  `json:"drift,omitempty"`
	StartTime time.Time     `json:"start_time"`
	EndTime   time.Time     `json:"end_time"`
}

// Summary returns a human-readable summary of the replay report.
func (r *ReplayReport) Summary() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("Replay: %s\n", r.PackHash))
	b.WriteString(fmt.Sprintf("Fidelity: %s\n", r.Fidelity))
	b.WriteString(fmt.Sprintf("Duration: %s\n\n", r.EndTime.Sub(r.StartTime)))

	b.WriteString(fmt.Sprintf("Steps (%d):\n", len(r.Steps)))
	for _, s := range r.Steps {
		var icon string
		switch s.Status {
		case StepMatched:
			icon = "✓"
		case StepDiverged:
			if !s.Deterministic {
				icon = "≈"
			} else {
				icon = "≠"
			}
		case StepFailed:
			icon = "✗"
		}

		detail := ""
		if s.Status == StepDiverged && !s.Deterministic {
			detail = " (expected, non-deterministic)"
		}
		if s.Reason != "" {
			detail = fmt.Sprintf(" (%s)", s.Reason)
		}

		b.WriteString(fmt.Sprintf("  %s [%d] %s %s%s\n", icon, s.Index, s.Tool, s.Status, detail))
	}

	if len(r.Drift) > 0 {
		b.WriteString(fmt.Sprintf("\nDrift (%d):\n", len(r.Drift)))
		for _, d := range r.Drift {
			b.WriteString(fmt.Sprintf("  %s: %s\n", d.Type, d.Description))
		}
	}

	return b.String()
}

// JSON returns the report as JSON bytes.
func (r *ReplayReport) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}
