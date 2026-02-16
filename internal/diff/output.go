package diff

import (
	"fmt"

	"github.com/contextsubstrate/ctx/internal/pack"
	"github.com/contextsubstrate/ctx/internal/store"
)

// CompareOutputs compares final output artifacts between two packs.
func CompareOutputs(a *pack.Pack, b *pack.Pack) []DriftEntry {
	var entries []DriftEntry

	mapA := make(map[string]string) // name -> content_ref
	for _, o := range a.Outputs {
		mapA[o.Name] = o.ContentRef
	}

	mapB := make(map[string]string)
	for _, o := range b.Outputs {
		mapB[o.Name] = o.ContentRef
	}

	// Check shared outputs
	for name, refA := range mapA {
		refB, ok := mapB[name]
		if !ok {
			entries = append(entries, DriftEntry{
				Type:        OutputDrift,
				Description: fmt.Sprintf("Output %q removed in pack B", name),
				PackA:       store.ShortHash(refA, 12),
			})
			continue
		}
		if refA != refB {
			entries = append(entries, DriftEntry{
				Type:        OutputDrift,
				Description: fmt.Sprintf("Output %q content differs", name),
				PackA:       store.ShortHash(refA, 12),
				PackB:       store.ShortHash(refB, 12),
			})
		}
	}

	// Check outputs only in B
	for name, refB := range mapB {
		if _, ok := mapA[name]; !ok {
			entries = append(entries, DriftEntry{
				Type:        OutputDrift,
				Description: fmt.Sprintf("Output %q added in pack B", name),
				PackB:       store.ShortHash(refB, 12),
			})
		}
	}

	return entries
}
