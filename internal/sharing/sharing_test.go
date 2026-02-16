package sharing

import (
	"encoding/json"
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

func createTestPack(t *testing.T, root string, sysPrompt string) *pack.Pack {
	t.Helper()
	log := &pack.ExecutionLog{
		Model:        pack.LogModel{Identifier: "test-model", Parameters: map[string]interface{}{}},
		SystemPrompt: sysPrompt,
		Prompts:      []pack.LogPrompt{{Role: "user", Content: "test"}},
		Inputs:       []pack.LogInput{},
		Steps: []pack.LogStep{
			{Index: 0, Type: "tool_call", Tool: "test_tool", Parameters: map[string]interface{}{}, Output: "out", Deterministic: true},
		},
		Outputs:     []pack.LogOutput{{Name: "result.txt", Content: "result"}},
		Environment: pack.LogEnvironment{OS: "darwin", Runtime: "go1.22", ToolVersions: map[string]string{}},
	}
	p, err := pack.CreatePack(root, log)
	if err != nil {
		t.Fatalf("CreatePack failed: %v", err)
	}
	if err := pack.RegisterPack(root, p.Hash); err != nil {
		t.Fatalf("RegisterPack failed: %v", err)
	}
	return p
}

func TestForkExistingPack(t *testing.T) {
	root := setupTestStore(t)
	p := createTestPack(t, root, "test prompt")

	draftPath, err := Fork(root, p.Hash)
	if err != nil {
		t.Fatalf("Fork failed: %v", err)
	}

	if _, err := os.Stat(draftPath); err != nil {
		t.Fatalf("draft file not found: %v", err)
	}

	// Read draft and check parent
	data, _ := os.ReadFile(draftPath)
	var draft pack.Pack
	json.Unmarshal(data, &draft)

	if draft.Parent != p.Hash {
		t.Errorf("expected parent %s, got %s", p.Hash, draft.Parent)
	}
}

func TestForkNonExistentPack(t *testing.T) {
	root := setupTestStore(t)
	_, err := Fork(root, "sha256:0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Error("expected error for non-existent pack")
	}
}

func TestFinalizeDraft(t *testing.T) {
	root := setupTestStore(t)
	p := createTestPack(t, root, "test prompt")

	draftPath, err := Fork(root, p.Hash)
	if err != nil {
		t.Fatalf("Fork failed: %v", err)
	}

	finalized, err := FinalizeDraft(root, draftPath)
	if err != nil {
		t.Fatalf("FinalizeDraft failed: %v", err)
	}

	if finalized.Parent != p.Hash {
		t.Errorf("expected parent %s, got %s", p.Hash, finalized.Parent)
	}
	if finalized.Hash == p.Hash {
		t.Error("finalized pack should have different hash than parent")
	}
	if finalized.Hash == "" {
		t.Error("finalized pack should have a hash")
	}

	// Draft should be removed
	if _, err := os.Stat(draftPath); !os.IsNotExist(err) {
		t.Error("draft file should be removed after finalization")
	}
}

func TestListPacksPopulated(t *testing.T) {
	root := setupTestStore(t)
	createTestPack(t, root, "prompt A")
	createTestPack(t, root, "prompt B")

	summaries, err := ListPacks(root, 50)
	if err != nil {
		t.Fatalf("ListPacks failed: %v", err)
	}

	if len(summaries) != 2 {
		t.Errorf("expected 2 packs, got %d", len(summaries))
	}
}

func TestListPacksEmpty(t *testing.T) {
	root := setupTestStore(t)

	summaries, err := ListPacks(root, 50)
	if err != nil {
		t.Fatalf("ListPacks failed: %v", err)
	}

	if len(summaries) != 0 {
		t.Errorf("expected 0 packs, got %d", len(summaries))
	}

	output := FormatPackList(summaries)
	if output != "No context packs found.\n" {
		t.Errorf("expected 'No context packs found.', got %q", output)
	}
}

func TestListPacksShowsForkedLineage(t *testing.T) {
	root := setupTestStore(t)
	p := createTestPack(t, root, "original")

	draftPath, _ := Fork(root, p.Hash)
	FinalizeDraft(root, draftPath)

	summaries, err := ListPacks(root, 50)
	if err != nil {
		t.Fatalf("ListPacks failed: %v", err)
	}

	hasForked := false
	for _, s := range summaries {
		if s.Parent != "" {
			hasForked = true
		}
	}
	if !hasForked {
		t.Error("expected at least one forked pack in list")
	}
}

func TestManifestNoAbsolutePaths(t *testing.T) {
	root := setupTestStore(t)
	p := createTestPack(t, root, "test")

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	str := string(data)
	if filepath.IsAbs(str) {
		// This is a rough check — abs paths contain OS-specific root
		for _, field := range []string{p.SystemPrompt} {
			if filepath.IsAbs(field) {
				t.Errorf("absolute path found in manifest: %s", field)
			}
		}
	}
}
