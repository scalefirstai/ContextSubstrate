package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitStore(t *testing.T) {
	dir := t.TempDir()

	root, err := InitStore(dir)
	if err != nil {
		t.Fatalf("InitStore failed: %v", err)
	}

	// Check directories exist
	for _, sub := range []string{"objects", "packs", "refs"} {
		path := filepath.Join(root, sub)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected directory %s: %v", sub, err)
		} else if !info.IsDir() {
			t.Errorf("expected %s to be a directory", sub)
		}
	}

	// Check config.json
	configPath := filepath.Join(root, "config.json")
	if _, err := os.Stat(configPath); err != nil {
		t.Errorf("expected config.json: %v", err)
	}
}

func TestInitStoreAlreadyExists(t *testing.T) {
	dir := t.TempDir()

	_, err := InitStore(dir)
	if err != nil {
		t.Fatalf("first InitStore failed: %v", err)
	}

	_, err = InitStore(dir)
	if err == nil {
		t.Error("expected error for already initialized store")
	}
}
