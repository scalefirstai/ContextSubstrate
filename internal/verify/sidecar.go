package verify

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/contextsubstrate/ctx/internal/pack"
	"github.com/contextsubstrate/ctx/internal/store"
)

type SidecarMetadata struct {
	ContextPack string   `json:"context_pack"`
	Inputs      []string `json:"inputs"`
	Tools       []string `json:"tools"`
	Confidence  string   `json:"confidence,omitempty"`
	Notes       string   `json:"notes,omitempty"`
}

// SidecarPath returns the sidecar metadata file path for an artifact.
func SidecarPath(artifactPath string) string {
	return artifactPath + ".ctx.json"
}

// ReadSidecar reads and parses a sidecar metadata file.
func ReadSidecar(path string) (*SidecarMetadata, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("no provenance metadata found for artifact")
		}
		return nil, fmt.Errorf("reading sidecar: %w", err)
	}

	var meta SidecarMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("parsing sidecar: %w", err)
	}
	return &meta, nil
}

// WriteSidecar writes a sidecar metadata file.
func WriteSidecar(path string, meta *SidecarMetadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling sidecar: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// GenerateSidecars creates sidecar metadata files for all outputs in a pack.
func GenerateSidecars(p *pack.Pack, outputDir string) (int, error) {
	// Collect tool names from steps
	toolSet := make(map[string]bool)
	for _, step := range p.Steps {
		toolSet[step.Tool] = true
	}
	tools := make([]string, 0, len(toolSet))
	for t := range toolSet {
		tools = append(tools, t)
	}

	// Collect input refs
	inputRefs := make([]string, len(p.Inputs))
	for i, inp := range p.Inputs {
		inputRefs[i] = inp.ContentRef
	}

	count := 0
	for _, out := range p.Outputs {
		sidecarPath := SidecarPath(filepath.Join(outputDir, out.Name))
		meta := &SidecarMetadata{
			ContextPack: p.Hash,
			Inputs:      inputRefs,
			Tools:       tools,
		}
		if err := WriteSidecar(sidecarPath, meta); err != nil {
			return count, fmt.Errorf("writing sidecar for %s: %w", out.Name, err)
		}
		count++
	}

	return count, nil
}

// VerifyArtifact checks an artifact's provenance and content integrity.
type VerifyResult struct {
	ArtifactPath    string
	PackHash        string
	PackCreated     string
	Tools           []string
	ContentMatch    bool
	ContentExpected string
	ContentActual   string
	Confidence      string
	Notes           string
}

// Verify checks an artifact's provenance against the context store.
func Verify(storeRoot string, artifactPath string) (*VerifyResult, error) {
	// Read sidecar
	sidecarPath := SidecarPath(artifactPath)
	meta, err := ReadSidecar(sidecarPath)
	if err != nil {
		return nil, err
	}

	// Validate pack exists
	p, err := pack.LoadPack(storeRoot, meta.ContextPack)
	if err != nil {
		return nil, fmt.Errorf("provenance broken: referenced pack not found (%s)", store.ShortHash(meta.ContextPack, 12))
	}

	result := &VerifyResult{
		ArtifactPath: artifactPath,
		PackHash:     p.Hash,
		PackCreated:  p.Created.Format("2006-01-02 15:04:05 UTC"),
		Tools:        meta.Tools,
		Confidence:   meta.Confidence,
		Notes:        meta.Notes,
	}

	// Check content integrity
	artifactData, err := os.ReadFile(artifactPath)
	if err != nil {
		return result, fmt.Errorf("reading artifact: %w", err)
	}
	actualHash := store.HashContent(artifactData)

	// Find matching output by name
	baseName := filepath.Base(artifactPath)
	for _, out := range p.Outputs {
		if out.Name == baseName {
			result.ContentExpected = out.ContentRef
			result.ContentActual = actualHash
			result.ContentMatch = (actualHash == out.ContentRef)
			break
		}
	}

	return result, nil
}

// FormatVerifyResult produces a human-readable verification summary.
func FormatVerifyResult(r *VerifyResult) string {
	var s string
	s += fmt.Sprintf("Artifact:  %s\n", r.ArtifactPath)
	s += fmt.Sprintf("Pack:      %s\n", store.ShortHash(r.PackHash, 12))
	s += fmt.Sprintf("Created:   %s\n", r.PackCreated)

	if len(r.Tools) > 0 {
		s += fmt.Sprintf("Tools:     %v\n", r.Tools)
	}

	if r.ContentExpected != "" {
		if r.ContentMatch {
			s += "Integrity: verified\n"
		} else {
			s += fmt.Sprintf("Integrity: modified (expected %s, actual %s)\n",
				store.ShortHash(r.ContentExpected, 12),
				store.ShortHash(r.ContentActual, 12))
		}
	}

	if r.Confidence != "" {
		s += fmt.Sprintf("Confidence: %s\n", r.Confidence)
	}
	if r.Notes != "" {
		s += fmt.Sprintf("Notes:     %s\n", r.Notes)
	}

	s += fmt.Sprintf("\nTo inspect: ctx show %s\n", store.ShortHash(r.PackHash, 12))
	s += fmt.Sprintf("To replay:  ctx replay %s\n", store.ShortHash(r.PackHash, 12))

	return s
}
