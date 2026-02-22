package delta

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/contextsubstrate/ctx/internal/graph"
)

// DeltaReport describes the changes between two indexed commits.
type DeltaReport struct {
	Base               string   `json:"base"`
	Head               string   `json:"head"`
	FilesChanged       []string `json:"files_changed,omitempty"`
	FilesAdded         []string `json:"files_added,omitempty"`
	FilesDeleted       []string `json:"files_deleted,omitempty"`
	SymbolsInvalidated []string `json:"symbols_invalidated,omitempty"`
}

// JSON returns the delta report as formatted JSON.
func (r *DeltaReport) JSON() ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// Human returns a human-readable representation of the delta report.
func (r *DeltaReport) Human() string {
	var b strings.Builder

	fmt.Fprintf(&b, "Delta: %s..%s\n", shortSHA(r.Base), shortSHA(r.Head))
	fmt.Fprintf(&b, "───────────────────────────────────\n")

	totalChanges := len(r.FilesChanged) + len(r.FilesAdded) + len(r.FilesDeleted)
	fmt.Fprintf(&b, "Files affected: %d\n", totalChanges)

	if len(r.FilesAdded) > 0 {
		fmt.Fprintf(&b, "\nAdded (%d):\n", len(r.FilesAdded))
		for _, f := range r.FilesAdded {
			fmt.Fprintf(&b, "  + %s\n", f)
		}
	}

	if len(r.FilesDeleted) > 0 {
		fmt.Fprintf(&b, "\nDeleted (%d):\n", len(r.FilesDeleted))
		for _, f := range r.FilesDeleted {
			fmt.Fprintf(&b, "  - %s\n", f)
		}
	}

	if len(r.FilesChanged) > 0 {
		fmt.Fprintf(&b, "\nModified (%d):\n", len(r.FilesChanged))
		for _, f := range r.FilesChanged {
			fmt.Fprintf(&b, "  ~ %s\n", f)
		}
	}

	if len(r.SymbolsInvalidated) > 0 {
		fmt.Fprintf(&b, "\nSymbols invalidated (%d):\n", len(r.SymbolsInvalidated))
		for _, s := range r.SymbolsInvalidated {
			fmt.Fprintf(&b, "  ! %s\n", s)
		}
	}

	if totalChanges == 0 {
		fmt.Fprintf(&b, "\nNo changes detected.\n")
	}

	return b.String()
}

// IsEmpty returns true if no changes were detected.
func (r *DeltaReport) IsEmpty() bool {
	return len(r.FilesChanged) == 0 && len(r.FilesAdded) == 0 && len(r.FilesDeleted) == 0
}

// ComputeDelta compares two indexed commits and produces a delta report.
// Both commits must have been previously indexed.
func ComputeDelta(storeRoot, baseSHA, headSHA string) (*DeltaReport, error) {
	// Read file snapshots for both commits
	baseFiles, err := graph.ReadRecords[graph.FileSnapshot](graph.FilesPath(storeRoot, baseSHA))
	if err != nil {
		return nil, fmt.Errorf("reading base files (%s): %w", baseSHA[:8], err)
	}

	headFiles, err := graph.ReadRecords[graph.FileSnapshot](graph.FilesPath(storeRoot, headSHA))
	if err != nil {
		return nil, fmt.Errorf("reading head files (%s): %w", headSHA[:8], err)
	}

	// Build maps by PathID
	baseMap := make(map[string]graph.FileSnapshot, len(baseFiles))
	for _, f := range baseFiles {
		baseMap[f.PathID] = f
	}

	headMap := make(map[string]graph.FileSnapshot, len(headFiles))
	for _, f := range headFiles {
		headMap[f.PathID] = f
	}

	// Load path records to resolve PathID → path
	paths, err := graph.ReadRecords[graph.PathRecord](graph.PathsPath(storeRoot))
	if err != nil {
		return nil, fmt.Errorf("reading paths: %w", err)
	}
	pathLookup := make(map[string]string, len(paths))
	for _, p := range paths {
		pathLookup[p.PathID] = p.Path
	}

	report := &DeltaReport{
		Base: baseSHA,
		Head: headSHA,
	}

	// Find added and changed files
	for pathID, headFile := range headMap {
		path := pathLookup[pathID]
		if path == "" {
			path = pathID // fallback
		}

		baseFile, exists := baseMap[pathID]
		if !exists {
			report.FilesAdded = append(report.FilesAdded, path)
		} else if baseFile.ContentSHA256 != headFile.ContentSHA256 {
			report.FilesChanged = append(report.FilesChanged, path)
		}
	}

	// Find deleted files
	for pathID := range baseMap {
		if _, exists := headMap[pathID]; !exists {
			path := pathLookup[pathID]
			if path == "" {
				path = pathID
			}
			report.FilesDeleted = append(report.FilesDeleted, path)
		}
	}

	// Sort all slices for deterministic output
	sort.Strings(report.FilesChanged)
	sort.Strings(report.FilesAdded)
	sort.Strings(report.FilesDeleted)

	// Phase 1: SymbolsInvalidated is populated by matching changed file PathIDs
	// against symbol records. For now, leave empty (symbol extraction is Phase 2).

	return report, nil
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}
