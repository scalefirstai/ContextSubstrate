package sharing

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/contextsubstrate/ctx/internal/pack"
	"github.com/contextsubstrate/ctx/internal/store"
)

// Fork creates a mutable draft derived from an existing pack.
func Fork(storeRoot string, sourceHash string) (string, error) {
	p, err := pack.LoadPack(storeRoot, sourceHash)
	if err != nil {
		return "", err
	}

	// Create drafts directory if needed
	draftsDir := filepath.Join(storeRoot, "drafts")
	if err := os.MkdirAll(draftsDir, 0755); err != nil {
		return "", fmt.Errorf("creating drafts directory: %w", err)
	}

	// Create draft with parent reference
	p.Parent = p.Hash
	p.Hash = "" // Will be computed on finalization

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", fmt.Errorf("serializing draft: %w", err)
	}

	// Use parent short hash as draft name
	draftName := store.ShortHash(p.Parent, 12) + ".draft.json"
	draftPath := filepath.Join(draftsDir, draftName)

	if err := os.WriteFile(draftPath, data, 0644); err != nil {
		return "", fmt.Errorf("writing draft: %w", err)
	}

	return draftPath, nil
}

// FinalizeDraft converts a mutable draft into an immutable pack.
func FinalizeDraft(storeRoot string, draftPath string) (*pack.Pack, error) {
	data, err := os.ReadFile(draftPath)
	if err != nil {
		return nil, fmt.Errorf("reading draft: %w", err)
	}

	var p pack.Pack
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing draft: %w", err)
	}

	if p.Parent == "" {
		return nil, fmt.Errorf("draft has no parent reference")
	}

	// Clear hash so canonical serialization is clean
	p.Hash = ""

	// Re-serialize as canonical JSON and store as blob
	canonicalData, err := json.Marshal(&p)
	if err != nil {
		return nil, fmt.Errorf("serializing pack: %w", err)
	}

	hash, err := store.WriteBlob(storeRoot, canonicalData)
	if err != nil {
		return nil, fmt.Errorf("storing pack: %w", err)
	}
	p.Hash = hash

	// Register
	if err := pack.RegisterPack(storeRoot, hash); err != nil {
		// May already be registered if same content
		if !os.IsExist(err) {
			return nil, fmt.Errorf("registering pack: %w", err)
		}
	}

	// Remove draft
	os.Remove(draftPath)

	return &p, nil
}
