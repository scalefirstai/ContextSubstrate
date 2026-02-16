package store

import (
	"fmt"
	"os"
	"path/filepath"
)

// blobPath returns the filesystem path for a given hash reference within the store root.
// Uses 2-char prefix subdirectory: objects/ab/cdef1234...
func blobPath(root string, ref string) (string, error) {
	_, hexStr, err := ParseHash(ref)
	if err != nil {
		return "", err
	}
	if len(hexStr) < 2 {
		return "", fmt.Errorf("hash hex too short: %q", hexStr)
	}
	return filepath.Join(root, "objects", hexStr[:2], hexStr[2:]), nil
}

// WriteBlob stores content in the object store. Returns the content hash.
// Deduplication: if a blob with the same hash already exists, the write is skipped.
// Immutability: existing blobs are never overwritten.
func WriteBlob(root string, data []byte) (string, error) {
	ref := HashContent(data)

	path, err := blobPath(root, ref)
	if err != nil {
		return "", fmt.Errorf("computing blob path: %w", err)
	}

	// Deduplication: skip if already exists
	if _, err := os.Stat(path); err == nil {
		return ref, nil
	}

	// Create parent directory
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("creating blob directory: %w", err)
	}

	// Write atomically: write to temp file then rename
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0444); err != nil {
		return "", fmt.Errorf("writing blob: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return "", fmt.Errorf("finalizing blob: %w", err)
	}

	return ref, nil
}

// ReadBlob reads a blob from the object store and verifies its integrity.
func ReadBlob(root string, ref string) ([]byte, error) {
	path, err := blobPath(root, ref)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("blob not found: %s", ShortHash(ref, 12))
		}
		return nil, fmt.Errorf("reading blob: %w", err)
	}

	// Verify integrity
	actual := HashContent(data)
	if actual != ref {
		return nil, fmt.Errorf("blob integrity check failed: expected %s, got %s", ShortHash(ref, 12), ShortHash(actual, 12))
	}

	return data, nil
}

// BlobExists checks whether a blob exists in the object store without reading it.
func BlobExists(root string, ref string) bool {
	path, err := blobPath(root, ref)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
