package cache

import (
	"encoding/json"
	"testing"
)

func TestPutAndGet(t *testing.T) {
	storeRoot := t.TempDir() + "/.ctx"

	entry := &CacheEntry{
		ArtifactType: "summary",
		ScopeType:    "file",
		ScopeID:      "main.go",
		ContentHash:  "abc123def456",
		Model:        "gpt-4",
		Payload:      json.RawMessage(`{"text":"hello world"}`),
		TokensIn:     100,
		TokensOut:    50,
	}

	if err := Put(storeRoot, entry); err != nil {
		t.Fatalf("Put: %v", err)
	}

	got, err := Get(storeRoot, "abc123def456", "summary")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got == nil {
		t.Fatal("expected cache hit, got nil")
	}
	if got.ContentHash != "abc123def456" {
		t.Errorf("ContentHash: got %q, want %q", got.ContentHash, "abc123def456")
	}
	if got.Model != "gpt-4" {
		t.Errorf("Model: got %q, want %q", got.Model, "gpt-4")
	}
	if got.TokensIn != 100 {
		t.Errorf("TokensIn: got %d, want %d", got.TokensIn, 100)
	}
}

func TestGetMiss(t *testing.T) {
	storeRoot := t.TempDir() + "/.ctx"

	got, err := Get(storeRoot, "nonexistent", "summary")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Error("expected nil for cache miss")
	}
}

func TestPutReplace(t *testing.T) {
	storeRoot := t.TempDir() + "/.ctx"

	entry := &CacheEntry{
		ArtifactType: "summary",
		ScopeType:    "file",
		ScopeID:      "main.go",
		ContentHash:  "abc123",
		Payload:      json.RawMessage(`{"v":1}`),
	}
	Put(storeRoot, entry)

	// Update same entry
	entry.Payload = json.RawMessage(`{"v":2}`)
	Put(storeRoot, entry)

	// Should have only one entry
	entries, err := List(storeRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after replace, got %d", len(entries))
	}

	// Verify it was updated
	got, _ := Get(storeRoot, "abc123", "summary")
	if string(got.Payload) != `{"v":2}` {
		t.Errorf("payload not updated: %s", got.Payload)
	}
}

func TestInvalidate(t *testing.T) {
	storeRoot := t.TempDir() + "/.ctx"

	// Add 3 entries
	for _, hash := range []string{"hash1", "hash2", "hash3"} {
		Put(storeRoot, &CacheEntry{
			ArtifactType: "summary",
			ScopeID:      hash,
			ContentHash:  hash,
			Payload:      json.RawMessage(`{}`),
		})
	}

	entries, _ := List(storeRoot)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// Invalidate 2 of them
	removed, err := Invalidate(storeRoot, []string{"hash1", "hash3"})
	if err != nil {
		t.Fatalf("Invalidate: %v", err)
	}
	if removed != 2 {
		t.Errorf("removed: got %d, want 2", removed)
	}

	// Should have 1 left
	entries, _ = List(storeRoot)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry after invalidation, got %d", len(entries))
	}
	if entries[0].ContentHash != "hash2" {
		t.Errorf("wrong entry remaining: %s", entries[0].ContentHash)
	}
}

func TestInvalidateNone(t *testing.T) {
	storeRoot := t.TempDir() + "/.ctx"

	removed, err := Invalidate(storeRoot, []string{"nonexistent"})
	if err != nil {
		t.Fatalf("Invalidate: %v", err)
	}
	if removed != 0 {
		t.Errorf("removed: got %d, want 0", removed)
	}
}

func TestList(t *testing.T) {
	storeRoot := t.TempDir() + "/.ctx"

	// Empty list
	entries, err := List(storeRoot)
	if err != nil {
		t.Fatalf("List empty: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}

	// Add some entries
	Put(storeRoot, &CacheEntry{
		ArtifactType: "a",
		ScopeID:      "1",
		ContentHash:  "h1",
		Payload:      json.RawMessage(`{}`),
	})
	Put(storeRoot, &CacheEntry{
		ArtifactType: "b",
		ScopeID:      "2",
		ContentHash:  "h2",
		Payload:      json.RawMessage(`{}`),
	})

	entries, err = List(storeRoot)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}
