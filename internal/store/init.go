package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/contextsubstrate/ctx/internal/graph"
)

const StoreDirName = ".ctx"

type Config struct {
	Version string `json:"version"`
}

// InitStore creates a .ctx/ directory with the required subdirectory structure.
// Returns the path to the created store root.
func InitStore(dir string) (string, error) {
	root := filepath.Join(dir, StoreDirName)

	// Check if already exists
	if _, err := os.Stat(root); err == nil {
		return "", fmt.Errorf("context store already initialized at %s", root)
	}

	// Create directory structure
	dirs := []string{
		root,
		filepath.Join(root, "objects"),
		filepath.Join(root, "packs"),
		filepath.Join(root, "refs"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return "", fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	// Create config.json
	cfg := Config{Version: "0.1"}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling config: %w", err)
	}
	configPath := filepath.Join(root, "config.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return "", fmt.Errorf("writing config: %w", err)
	}

	// Initialize context graph directories
	if err := graph.InitGraph(root); err != nil {
		return "", fmt.Errorf("initializing graph: %w", err)
	}

	return root, nil
}
