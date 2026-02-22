package graph

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAppendAndReadRecords(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	// Append records
	r1 := CommitRecord{
		Type:       TypeCommit,
		Repo:       "/tmp/repo",
		SHA:        "abc123",
		ParentSHA:  "def456",
		Author:     "Test <test@example.com>",
		Message:    "initial commit",
		AuthoredAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	r2 := CommitRecord{
		Type:       TypeCommit,
		Repo:       "/tmp/repo",
		SHA:        "def456",
		Author:     "Test <test@example.com>",
		Message:    "second commit",
		AuthoredAt: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	if err := AppendRecord(path, r1); err != nil {
		t.Fatalf("AppendRecord 1: %v", err)
	}
	if err := AppendRecord(path, r2); err != nil {
		t.Fatalf("AppendRecord 2: %v", err)
	}

	// Read back
	records, err := ReadRecords[CommitRecord](path)
	if err != nil {
		t.Fatalf("ReadRecords: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].SHA != "abc123" {
		t.Errorf("first record SHA: got %q, want %q", records[0].SHA, "abc123")
	}
	if records[1].SHA != "def456" {
		t.Errorf("second record SHA: got %q, want %q", records[1].SHA, "def456")
	}
}

func TestReadRecordsNonExistent(t *testing.T) {
	records, err := ReadRecords[CommitRecord]("/nonexistent/path.jsonl")
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("expected empty slice, got %d records", len(records))
	}
}

func TestWriteRecords(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "write.jsonl")

	records := []any{
		PathRecord{Type: TypePath, PathID: "p1", Repo: "/repo", Path: "main.go", FirstSeenCommit: "aaa"},
		PathRecord{Type: TypePath, PathID: "p2", Repo: "/repo", Path: "util.go", FirstSeenCommit: "aaa"},
	}

	if err := WriteRecords(path, records); err != nil {
		t.Fatalf("WriteRecords: %v", err)
	}

	// Read back
	result, err := ReadRecords[PathRecord](path)
	if err != nil {
		t.Fatalf("ReadRecords: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 records, got %d", len(result))
	}
	if result[0].Path != "main.go" {
		t.Errorf("first path: got %q, want %q", result[0].Path, "main.go")
	}
	if result[1].Path != "util.go" {
		t.Errorf("second path: got %q, want %q", result[1].Path, "util.go")
	}
}

func TestWriteRecordsOverwrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "overwrite.jsonl")

	// Write initial records
	initial := []any{
		PathRecord{Type: TypePath, PathID: "p1", Path: "old.go"},
		PathRecord{Type: TypePath, PathID: "p2", Path: "old2.go"},
	}
	if err := WriteRecords(path, initial); err != nil {
		t.Fatalf("WriteRecords initial: %v", err)
	}

	// Overwrite with fewer records
	replacement := []any{
		PathRecord{Type: TypePath, PathID: "p3", Path: "new.go"},
	}
	if err := WriteRecords(path, replacement); err != nil {
		t.Fatalf("WriteRecords replacement: %v", err)
	}

	result, err := ReadRecords[PathRecord](path)
	if err != nil {
		t.Fatalf("ReadRecords: %v", err)
	}
	if len(result) != 1 {
		t.Fatalf("expected 1 record, got %d", len(result))
	}
	if result[0].Path != "new.go" {
		t.Errorf("path: got %q, want %q", result[0].Path, "new.go")
	}
}

func TestAppendCreatesDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "dir", "test.jsonl")

	rec := FileSnapshot{Type: TypeFileSnapshot, Commit: "abc", PathID: "p1"}
	if err := AppendRecord(path, rec); err != nil {
		t.Fatalf("AppendRecord with nested dirs: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("file not created: %v", err)
	}
}

func TestInitGraph(t *testing.T) {
	dir := t.TempDir()
	storeRoot := filepath.Join(dir, ".ctx")
	os.MkdirAll(storeRoot, 0755)

	if err := InitGraph(storeRoot); err != nil {
		t.Fatalf("InitGraph: %v", err)
	}

	// Check directories exist
	for _, sub := range []string{"graph", "graph/manifests", "graph/snapshots"} {
		path := filepath.Join(storeRoot, sub)
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected directory %s: %v", sub, err)
		} else if !info.IsDir() {
			t.Errorf("expected %s to be a directory", sub)
		}
	}
}

func TestInitGraphIdempotent(t *testing.T) {
	dir := t.TempDir()
	storeRoot := filepath.Join(dir, ".ctx")
	os.MkdirAll(storeRoot, 0755)

	if err := InitGraph(storeRoot); err != nil {
		t.Fatalf("first InitGraph: %v", err)
	}
	if err := InitGraph(storeRoot); err != nil {
		t.Fatalf("second InitGraph should be idempotent: %v", err)
	}
}

func TestPathHelpers(t *testing.T) {
	root := "/tmp/.ctx"

	if got := CommitsPath(root); got != "/tmp/.ctx/graph/manifests/commits.jsonl" {
		t.Errorf("CommitsPath: got %q", got)
	}
	if got := PathsPath(root); got != "/tmp/.ctx/graph/manifests/paths.jsonl" {
		t.Errorf("PathsPath: got %q", got)
	}
	if got := FilesPath(root, "abc123"); got != "/tmp/.ctx/graph/snapshots/abc123/files.jsonl" {
		t.Errorf("FilesPath: got %q", got)
	}
	if got := SnapshotDir(root, "abc123"); got != "/tmp/.ctx/graph/snapshots/abc123" {
		t.Errorf("SnapshotDir: got %q", got)
	}
}

func TestFileSnapshotRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "files.jsonl")

	snap := FileSnapshot{
		Type:          TypeFileSnapshot,
		Commit:        "abc123",
		PathID:        "p1",
		BlobOID:       "blob123",
		ContentSHA256: "deadbeef",
		Language:      "go",
		ByteSize:      1024,
		LOC:           42,
		IsGenerated:   false,
		IsBinary:      false,
	}

	if err := AppendRecord(path, snap); err != nil {
		t.Fatalf("AppendRecord: %v", err)
	}

	records, err := ReadRecords[FileSnapshot](path)
	if err != nil {
		t.Fatalf("ReadRecords: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1, got %d", len(records))
	}
	if records[0].Language != "go" {
		t.Errorf("language: got %q, want %q", records[0].Language, "go")
	}
	if records[0].LOC != 42 {
		t.Errorf("LOC: got %d, want %d", records[0].LOC, 42)
	}
}
