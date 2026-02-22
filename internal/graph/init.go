package graph

import (
	"fmt"
	"os"
	"path/filepath"
)

// GraphDir is the subdirectory name within .ctx/ for the context graph.
const GraphDir = "graph"

// InitGraph creates the .ctx/graph/ directory structure for the context graph store.
// storeRoot is the path to the .ctx/ directory.
func InitGraph(storeRoot string) error {
	graphRoot := filepath.Join(storeRoot, GraphDir)

	dirs := []string{
		graphRoot,
		filepath.Join(graphRoot, "manifests"),
		filepath.Join(graphRoot, "snapshots"),
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("creating graph directory %s: %w", d, err)
		}
	}

	return nil
}

// SnapshotDir returns the path to the snapshot directory for a given commit SHA.
func SnapshotDir(storeRoot, commitSHA string) string {
	return filepath.Join(storeRoot, GraphDir, "snapshots", commitSHA)
}

// ManifestsDir returns the path to the manifests directory.
func ManifestsDir(storeRoot string) string {
	return filepath.Join(storeRoot, GraphDir, "manifests")
}

// CommitsPath returns the path to the commits manifest JSONL file.
func CommitsPath(storeRoot string) string {
	return filepath.Join(ManifestsDir(storeRoot), "commits.jsonl")
}

// PathsPath returns the path to the paths manifest JSONL file.
func PathsPath(storeRoot string) string {
	return filepath.Join(ManifestsDir(storeRoot), "paths.jsonl")
}

// FilesPath returns the path to the files snapshot JSONL for a commit.
func FilesPath(storeRoot, commitSHA string) string {
	return filepath.Join(SnapshotDir(storeRoot, commitSHA), "files.jsonl")
}

// SymbolsPath returns the path to the symbols snapshot JSONL for a commit.
func SymbolsPath(storeRoot, commitSHA string) string {
	return filepath.Join(SnapshotDir(storeRoot, commitSHA), "symbols.jsonl")
}

// RegionsPath returns the path to the regions snapshot JSONL for a commit.
func RegionsPath(storeRoot, commitSHA string) string {
	return filepath.Join(SnapshotDir(storeRoot, commitSHA), "regions.jsonl")
}

// ImportEdgesPath returns the path to the import edges JSONL for a commit.
func ImportEdgesPath(storeRoot, commitSHA string) string {
	return filepath.Join(SnapshotDir(storeRoot, commitSHA), "edges.imports.jsonl")
}

// CallEdgesPath returns the path to the call edges JSONL for a commit.
func CallEdgesPath(storeRoot, commitSHA string) string {
	return filepath.Join(SnapshotDir(storeRoot, commitSHA), "edges.calls.jsonl")
}
