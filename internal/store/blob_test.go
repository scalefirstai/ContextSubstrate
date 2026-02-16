package store

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func setupTestStore(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	root := filepath.Join(dir, ".ctx")
	os.MkdirAll(filepath.Join(root, "objects"), 0755)
	os.MkdirAll(filepath.Join(root, "packs"), 0755)
	os.MkdirAll(filepath.Join(root, "refs"), 0755)
	return root
}

func TestWriteAndReadBlob(t *testing.T) {
	root := setupTestStore(t)
	data := []byte("hello world")

	ref, err := WriteBlob(root, data)
	if err != nil {
		t.Fatalf("WriteBlob failed: %v", err)
	}
	if !ValidateHash(ref) {
		t.Errorf("invalid hash returned: %s", ref)
	}

	got, err := ReadBlob(root, ref)
	if err != nil {
		t.Fatalf("ReadBlob failed: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("expected %q, got %q", data, got)
	}
}

func TestWriteBlobDirectoryStructure(t *testing.T) {
	root := setupTestStore(t)
	data := []byte("test content")

	ref, err := WriteBlob(root, data)
	if err != nil {
		t.Fatalf("WriteBlob failed: %v", err)
	}

	_, hexStr, _ := ParseHash(ref)
	expectedPath := filepath.Join(root, "objects", hexStr[:2], hexStr[2:])
	if _, err := os.Stat(expectedPath); err != nil {
		t.Errorf("expected blob at %s, got error: %v", expectedPath, err)
	}
}

func TestWriteBlobDeduplication(t *testing.T) {
	root := setupTestStore(t)
	data := []byte("duplicate content")

	ref1, err := WriteBlob(root, data)
	if err != nil {
		t.Fatalf("first WriteBlob failed: %v", err)
	}
	ref2, err := WriteBlob(root, data)
	if err != nil {
		t.Fatalf("second WriteBlob failed: %v", err)
	}
	if ref1 != ref2 {
		t.Errorf("expected same hash, got %s and %s", ref1, ref2)
	}
}

func TestReadBlobNotFound(t *testing.T) {
	root := setupTestStore(t)
	fakeHash := "sha256:0000000000000000000000000000000000000000000000000000000000000000"

	_, err := ReadBlob(root, fakeHash)
	if err == nil {
		t.Error("expected error for missing blob")
	}
}

func TestReadBlobCorruption(t *testing.T) {
	root := setupTestStore(t)
	data := []byte("original content")

	ref, err := WriteBlob(root, data)
	if err != nil {
		t.Fatalf("WriteBlob failed: %v", err)
	}

	// Tamper with the blob
	path, _ := blobPath(root, ref)
	os.Chmod(path, 0644) // make writable
	os.WriteFile(path, []byte("tampered"), 0444)

	_, err = ReadBlob(root, ref)
	if err == nil {
		t.Error("expected integrity error for tampered blob")
	}
}

func TestBlobExists(t *testing.T) {
	root := setupTestStore(t)
	data := []byte("exists check")

	ref, _ := WriteBlob(root, data)

	if !BlobExists(root, ref) {
		t.Error("expected blob to exist")
	}

	fakeHash := "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	if BlobExists(root, fakeHash) {
		t.Error("expected blob not to exist")
	}
}
