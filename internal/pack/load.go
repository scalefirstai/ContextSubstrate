package pack

import (
	"encoding/json"
	"fmt"

	"github.com/contextsubstrate/ctx/internal/store"
)

// LoadPack loads a pack manifest from the object store by its hash reference.
// Accepts full hashes, short hex prefixes, and ctx:// URIs.
func LoadPack(storeRoot string, ref string) (*Pack, error) {
	normalized, err := store.ResolveHash(storeRoot, ref)
	if err != nil {
		return nil, fmt.Errorf("invalid hash: %w", err)
	}

	data, err := store.ReadBlob(storeRoot, normalized)
	if err != nil {
		return nil, fmt.Errorf("pack not found: %s", store.ShortHash(normalized, 12))
	}

	var p Pack
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing pack manifest: %w", err)
	}

	// The manifest is stored without the hash field; set it from the blob identity
	p.Hash = normalized

	return &p, nil
}

// FormatPack produces a human-readable summary of a pack.
func FormatPack(p *Pack) string {
	var s string
	s += fmt.Sprintf("Pack:    %s\n", store.ShortHash(p.Hash, 12))
	s += fmt.Sprintf("Created: %s\n", p.Created.Format("2006-01-02 15:04:05 UTC"))
	s += fmt.Sprintf("Model:   %s\n", p.Model.Identifier)
	if p.Parent != "" {
		s += fmt.Sprintf("Parent:  %s\n", store.ShortHash(p.Parent, 12))
	}
	s += fmt.Sprintf("\nSystem Prompt: %s\n", store.ShortHash(p.SystemPrompt, 12))

	if len(p.Inputs) > 0 {
		s += fmt.Sprintf("\nInputs (%d):\n", len(p.Inputs))
		for _, inp := range p.Inputs {
			s += fmt.Sprintf("  %s (%d bytes)\n", inp.Name, inp.Size)
		}
	}

	if len(p.Steps) > 0 {
		s += fmt.Sprintf("\nSteps (%d):\n", len(p.Steps))
		for _, step := range p.Steps {
			det := "deterministic"
			if !step.Deterministic {
				det = "non-deterministic"
			}
			s += fmt.Sprintf("  [%d] %s %s (%s)\n", step.Index, step.Type, step.Tool, det)
		}
	}

	if len(p.Outputs) > 0 {
		s += fmt.Sprintf("\nOutputs (%d):\n", len(p.Outputs))
		for _, out := range p.Outputs {
			s += fmt.Sprintf("  %s\n", out.Name)
		}
	}

	s += fmt.Sprintf("\nEnvironment: %s / %s\n", p.Environment.OS, p.Environment.Runtime)
	if len(p.Environment.ToolVersions) > 0 {
		s += "Tool Versions:\n"
		for k, v := range p.Environment.ToolVersions {
			s += fmt.Sprintf("  %s: %s\n", k, v)
		}
	}

	return s
}
