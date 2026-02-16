package verify

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/contextsubstrate/ctx/internal/pack"
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

func createTestPack(t *testing.T, root string) *pack.Pack {
	t.Helper()
	log := &pack.ExecutionLog{
		Model:        pack.LogModel{Identifier: "test-model", Parameters: map[string]interface{}{}},
		SystemPrompt: "test prompt",
		Prompts:      []pack.LogPrompt{{Role: "user", Content: "test"}},
		Inputs:       []pack.LogInput{{Name: "input.txt", Content: "input data"}},
		Steps: []pack.LogStep{
			{Index: 0, Type: "tool_call", Tool: "write_file", Parameters: map[string]interface{}{}, Output: "result content", Deterministic: true},
		},
		Outputs:     []pack.LogOutput{{Name: "result.txt", Content: "result content"}},
		Environment: pack.LogEnvironment{OS: "darwin", Runtime: "go1.22", ToolVersions: map[string]string{}},
	}
	p, err := pack.CreatePack(root, log)
	if err != nil {
		t.Fatalf("CreatePack failed: %v", err)
	}
	return p
}

func TestSidecarRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ctx.json")

	meta := &SidecarMetadata{
		ContextPack: "sha256:abc123",
		Inputs:      []string{"sha256:input1"},
		Tools:       []string{"write_file"},
		Confidence:  "high",
		Notes:       "test notes",
	}

	if err := WriteSidecar(path, meta); err != nil {
		t.Fatalf("WriteSidecar failed: %v", err)
	}

	loaded, err := ReadSidecar(path)
	if err != nil {
		t.Fatalf("ReadSidecar failed: %v", err)
	}

	if loaded.ContextPack != meta.ContextPack {
		t.Errorf("ContextPack mismatch: %s vs %s", loaded.ContextPack, meta.ContextPack)
	}
	if loaded.Confidence != "high" {
		t.Errorf("Confidence mismatch: %s", loaded.Confidence)
	}
	if loaded.Notes != "test notes" {
		t.Errorf("Notes mismatch: %s", loaded.Notes)
	}
}

func TestGenerateSidecars(t *testing.T) {
	root := setupTestStore(t)
	p := createTestPack(t, root)

	outputDir := t.TempDir()
	// Create the artifact file
	os.WriteFile(filepath.Join(outputDir, "result.txt"), []byte("result content"), 0644)

	count, err := GenerateSidecars(p, outputDir)
	if err != nil {
		t.Fatalf("GenerateSidecars failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 sidecar, got %d", count)
	}

	sidecarPath := filepath.Join(outputDir, "result.txt.ctx.json")
	if _, err := os.Stat(sidecarPath); err != nil {
		t.Errorf("sidecar file not found: %v", err)
	}

	meta, err := ReadSidecar(sidecarPath)
	if err != nil {
		t.Fatalf("ReadSidecar failed: %v", err)
	}
	if meta.ContextPack != p.Hash {
		t.Errorf("sidecar pack hash mismatch: %s vs %s", meta.ContextPack, p.Hash)
	}
}

func TestVerifyValidProvenance(t *testing.T) {
	root := setupTestStore(t)
	p := createTestPack(t, root)

	outputDir := t.TempDir()
	artifactPath := filepath.Join(outputDir, "result.txt")
	os.WriteFile(artifactPath, []byte("result content"), 0644)
	GenerateSidecars(p, outputDir)

	result, err := Verify(root, artifactPath)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if result.PackHash != p.Hash {
		t.Errorf("pack hash mismatch: %s vs %s", result.PackHash, p.Hash)
	}
	if !result.ContentMatch {
		t.Error("expected content to match")
	}
}

func TestVerifyNoSidecar(t *testing.T) {
	root := setupTestStore(t)
	artifactPath := filepath.Join(t.TempDir(), "orphan.txt")
	os.WriteFile(artifactPath, []byte("no provenance"), 0644)

	_, err := Verify(root, artifactPath)
	if err == nil {
		t.Error("expected error for missing sidecar")
	}
}

func TestVerifyBrokenProvenance(t *testing.T) {
	root := setupTestStore(t)

	outputDir := t.TempDir()
	artifactPath := filepath.Join(outputDir, "result.txt")
	os.WriteFile(artifactPath, []byte("content"), 0644)

	// Write sidecar with non-existent pack
	meta := &SidecarMetadata{
		ContextPack: "sha256:0000000000000000000000000000000000000000000000000000000000000000",
		Inputs:      []string{},
		Tools:       []string{},
	}
	WriteSidecar(SidecarPath(artifactPath), meta)

	_, err := Verify(root, artifactPath)
	if err == nil {
		t.Error("expected error for broken provenance")
	}
}

func TestVerifyModifiedContent(t *testing.T) {
	root := setupTestStore(t)
	p := createTestPack(t, root)

	outputDir := t.TempDir()
	artifactPath := filepath.Join(outputDir, "result.txt")
	// Write DIFFERENT content than what the pack recorded
	os.WriteFile(artifactPath, []byte("modified content"), 0644)
	GenerateSidecars(p, outputDir)

	result, err := Verify(root, artifactPath)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if result.ContentMatch {
		t.Error("expected content NOT to match")
	}
}

func TestVerifyWithConfidence(t *testing.T) {
	root := setupTestStore(t)
	p := createTestPack(t, root)

	outputDir := t.TempDir()
	artifactPath := filepath.Join(outputDir, "result.txt")
	os.WriteFile(artifactPath, []byte("result content"), 0644)

	// Create sidecar with confidence
	meta := &SidecarMetadata{
		ContextPack: p.Hash,
		Inputs:      []string{},
		Tools:       []string{"write_file"},
		Confidence:  "low",
		Notes:       "Multiple valid approaches exist",
	}
	WriteSidecar(SidecarPath(artifactPath), meta)

	result, err := Verify(root, artifactPath)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if result.Confidence != "low" {
		t.Errorf("expected confidence 'low', got %s", result.Confidence)
	}
	if result.Notes != "Multiple valid approaches exist" {
		t.Errorf("expected notes, got %s", result.Notes)
	}
}

func TestVerifyWithoutConfidence(t *testing.T) {
	root := setupTestStore(t)
	p := createTestPack(t, root)

	outputDir := t.TempDir()
	artifactPath := filepath.Join(outputDir, "result.txt")
	os.WriteFile(artifactPath, []byte("result content"), 0644)
	GenerateSidecars(p, outputDir)

	result, err := Verify(root, artifactPath)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if result.Confidence != "" {
		t.Errorf("expected empty confidence, got %s", result.Confidence)
	}
}

func TestFormatVerifyResult(t *testing.T) {
	result := &VerifyResult{
		ArtifactPath: "result.txt",
		PackHash:     "sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
		PackCreated:  "2026-01-01 00:00:00 UTC",
		Tools:        []string{"write_file"},
		ContentMatch: true,
	}

	output := FormatVerifyResult(result)
	if output == "" {
		t.Error("expected non-empty output")
	}
}
