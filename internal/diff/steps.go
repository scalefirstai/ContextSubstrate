package diff

import (
	"fmt"
	"reflect"

	"github.com/contextsubstrate/ctx/internal/pack"
	"github.com/contextsubstrate/ctx/internal/store"
)

// CompareSteps compares execution steps between two packs, aligned by index.
func CompareSteps(a *pack.Pack, b *pack.Pack) []DriftEntry {
	var entries []DriftEntry

	minLen := len(a.Steps)
	if len(b.Steps) < minLen {
		minLen = len(b.Steps)
	}

	for i := 0; i < minLen; i++ {
		sa := a.Steps[i]
		sb := b.Steps[i]

		// Tool drift
		if sa.Tool != sb.Tool {
			entries = append(entries, DriftEntry{
				Type:        ToolDrift,
				Description: fmt.Sprintf("Step %d: different tool", i),
				StepIndex:   i,
				PackA:       sa.Tool,
				PackB:       sb.Tool,
			})
			continue // If tools differ, no point comparing params or output
		}

		// Param drift
		if !reflect.DeepEqual(sa.Parameters, sb.Parameters) {
			entries = append(entries, DriftEntry{
				Type:        ParamDrift,
				Description: fmt.Sprintf("Step %d: %s called with different parameters", i, sa.Tool),
				StepIndex:   i,
				PackA:       sa.Parameters,
				PackB:       sb.Parameters,
			})
		}

		// Reasoning drift (output divergence at same step)
		if sa.OutputRef != sb.OutputRef {
			entries = append(entries, DriftEntry{
				Type:        ReasoningDrift,
				Description: fmt.Sprintf("Step %d: %s produced different output", i, sa.Tool),
				StepIndex:   i,
				PackA:       store.ShortHash(sa.OutputRef, 12),
				PackB:       store.ShortHash(sb.OutputRef, 12),
			})
		}
	}

	// Handle mismatched step counts
	if len(a.Steps) > len(b.Steps) {
		for i := minLen; i < len(a.Steps); i++ {
			entries = append(entries, DriftEntry{
				Type:        ToolDrift,
				Description: fmt.Sprintf("Step %d: %s removed in pack B", i, a.Steps[i].Tool),
				StepIndex:   i,
				PackA:       a.Steps[i].Tool,
			})
		}
	} else if len(b.Steps) > len(a.Steps) {
		for i := minLen; i < len(b.Steps); i++ {
			entries = append(entries, DriftEntry{
				Type:        ToolDrift,
				Description: fmt.Sprintf("Step %d: %s added in pack B", i, b.Steps[i].Tool),
				StepIndex:   i,
				PackB:       b.Steps[i].Tool,
			})
		}
	}

	return entries
}
