package diff

import (
	"fmt"

	"github.com/contextsubstrate/ctx/internal/pack"
	"github.com/contextsubstrate/ctx/internal/store"
)

// ComparePrompts compares system prompts and user prompts between two packs.
func ComparePrompts(a *pack.Pack, b *pack.Pack) []DriftEntry {
	var entries []DriftEntry

	// Compare system prompts
	if a.SystemPrompt != b.SystemPrompt {
		entries = append(entries, DriftEntry{
			Type:        PromptDrift,
			Description: "System prompts differ",
			PackA:       store.ShortHash(a.SystemPrompt, 12),
			PackB:       store.ShortHash(b.SystemPrompt, 12),
		})
	}

	// Compare user prompts by index
	minLen := len(a.Prompts)
	if len(b.Prompts) < minLen {
		minLen = len(b.Prompts)
	}

	for i := 0; i < minLen; i++ {
		if a.Prompts[i].ContentRef != b.Prompts[i].ContentRef {
			entries = append(entries, DriftEntry{
				Type:        PromptDrift,
				Description: fmt.Sprintf("Prompt %d content differs (role: %s)", i, a.Prompts[i].Role),
				StepIndex:   i,
				PackA:       store.ShortHash(a.Prompts[i].ContentRef, 12),
				PackB:       store.ShortHash(b.Prompts[i].ContentRef, 12),
			})
		}
		if a.Prompts[i].Role != b.Prompts[i].Role {
			entries = append(entries, DriftEntry{
				Type:        PromptDrift,
				Description: fmt.Sprintf("Prompt %d role changed", i),
				PackA:       a.Prompts[i].Role,
				PackB:       b.Prompts[i].Role,
			})
		}
	}

	// Handle mismatched lengths
	if len(a.Prompts) > len(b.Prompts) {
		entries = append(entries, DriftEntry{
			Type:        PromptDrift,
			Description: fmt.Sprintf("Pack A has %d extra prompt(s)", len(a.Prompts)-len(b.Prompts)),
		})
	} else if len(b.Prompts) > len(a.Prompts) {
		entries = append(entries, DriftEntry{
			Type:        PromptDrift,
			Description: fmt.Sprintf("Pack B has %d extra prompt(s)", len(b.Prompts)-len(a.Prompts)),
		})
	}

	return entries
}
