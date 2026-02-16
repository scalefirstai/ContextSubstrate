package sharing

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/contextsubstrate/ctx/internal/pack"
	"github.com/contextsubstrate/ctx/internal/store"
)

type PackSummary struct {
	Hash    string
	Created string
	Model   string
	Steps   int
	Parent  string
}

// ListPacks lists all finalized packs in the store, sorted by creation date (newest first).
func ListPacks(storeRoot string, limit int) ([]PackSummary, error) {
	packsDir := filepath.Join(storeRoot, "packs")
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading packs directory: %w", err)
	}

	var summaries []PackSummary
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ref := "sha256:" + entry.Name()
		if !store.ValidateHash(ref) {
			continue
		}

		p, err := pack.LoadPack(storeRoot, ref)
		if err != nil {
			continue // Skip corrupted packs
		}

		summaries = append(summaries, PackSummary{
			Hash:    p.Hash,
			Created: p.Created.Format("2006-01-02 15:04:05"),
			Model:   p.Model.Identifier,
			Steps:   len(p.Steps),
			Parent:  p.Parent,
		})
	}

	// Sort by created date (newest first)
	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Created > summaries[j].Created
	})

	if limit > 0 && len(summaries) > limit {
		summaries = summaries[:limit]
	}

	return summaries, nil
}

// FormatPackList produces human-readable output for a list of packs.
func FormatPackList(summaries []PackSummary) string {
	if len(summaries) == 0 {
		return "No context packs found.\n"
	}

	var s string
	for _, p := range summaries {
		parent := ""
		if p.Parent != "" {
			parent = fmt.Sprintf(" (forked from %s)", store.ShortHash(p.Parent, 12))
		}
		s += fmt.Sprintf("%s  %s  %s  %d steps%s\n",
			store.ShortHash(p.Hash, 12),
			p.Created,
			p.Model,
			p.Steps,
			parent,
		)
	}
	return s
}
