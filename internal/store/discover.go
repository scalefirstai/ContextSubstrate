package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// DiscoverStore walks up from the current directory to find the nearest .ctx/ directory.
func DiscoverStore() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	return DiscoverStoreFrom(dir)
}

// DiscoverStoreFrom walks up from the given directory to find the nearest .ctx/ directory.
func DiscoverStoreFrom(dir string) (string, error) {
	for {
		candidate := filepath.Join(dir, StoreDirName)
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			return "", fmt.Errorf("no context store found (run 'ctx init' to create one)")
		}
		dir = parent
	}
}
