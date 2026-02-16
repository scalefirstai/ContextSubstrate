package replay

import (
	"fmt"
	"runtime"
	"time"

	"github.com/contextsubstrate/ctx/internal/pack"
	"github.com/contextsubstrate/ctx/internal/store"
)

// Replay re-executes an agent run from a Context Pack and produces a fidelity report.
func Replay(storeRoot string, packHash string) (*ReplayReport, error) {
	p, err := pack.LoadPack(storeRoot, packHash)
	if err != nil {
		return nil, err
	}

	report := &ReplayReport{
		PackHash:  p.Hash,
		StartTime: time.Now(),
	}

	executors := DefaultExecutors()

	// Check environment drift
	report.Drift = checkEnvironmentDrift(p)

	// Check input availability
	for _, input := range p.Inputs {
		if !store.BlobExists(storeRoot, input.ContentRef) {
			report.Drift = append(report.Drift, DriftEntry{
				Type:        "missing_input",
				Description: fmt.Sprintf("input %q not found in store", input.Name),
				Expected:    store.ShortHash(input.ContentRef, 12),
			})
		}
	}

	// Execute steps
	hasFailed := false
	hasDiverged := false

	for i := range p.Steps {
		result := ExecuteStep(storeRoot, &p.Steps[i], executors)
		report.Steps = append(report.Steps, *result)

		switch result.Status {
		case StepFailed:
			hasFailed = true
		case StepDiverged:
			if result.Deterministic {
				hasDiverged = true
			}
		}
	}

	// Compute fidelity
	switch {
	case hasFailed:
		report.Fidelity = FidelityFailed
	case hasDiverged:
		report.Fidelity = FidelityDegraded
	default:
		report.Fidelity = FidelityExact
	}

	report.EndTime = time.Now()
	return report, nil
}

func checkEnvironmentDrift(p *pack.Pack) []DriftEntry {
	var drift []DriftEntry

	currentOS := runtime.GOOS
	if p.Environment.OS != "" && p.Environment.OS != currentOS {
		drift = append(drift, DriftEntry{
			Type:        "environment",
			Description: "OS changed",
			Expected:    p.Environment.OS,
			Actual:      currentOS,
		})
	}

	return drift
}
