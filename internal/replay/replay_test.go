package replay

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/contextsubstrate/ctx/internal/pack"
	"github.com/contextsubstrate/ctx/internal/store"
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

func createTestPack(t *testing.T, root string, steps []pack.LogStep) *pack.Pack {
	t.Helper()
	log := &pack.ExecutionLog{
		Model:        pack.LogModel{Identifier: "test-model", Parameters: map[string]interface{}{}},
		SystemPrompt: "test prompt",
		Prompts:      []pack.LogPrompt{{Role: "user", Content: "test"}},
		Inputs:       []pack.LogInput{},
		Steps:        steps,
		Outputs:      []pack.LogOutput{{Name: "out.txt", Content: "result"}},
		Environment:  pack.LogEnvironment{OS: "darwin", Runtime: "go1.22", ToolVersions: map[string]string{}},
	}
	p, err := pack.CreatePack(root, log)
	if err != nil {
		t.Fatalf("CreatePack failed: %v", err)
	}
	return p
}

func TestReplayExactFidelity(t *testing.T) {
	root := setupTestStore(t)

	// Create a file that the read_file tool can read
	testFile := filepath.Join(t.TempDir(), "test.txt")
	content := []byte("hello world")
	os.WriteFile(testFile, content, 0644)

	contentHash := store.HashContent(content)

	p := createTestPack(t, root, []pack.LogStep{
		{
			Index:         0,
			Type:          "tool_call",
			Tool:          "read_file",
			Parameters:    map[string]interface{}{"path": testFile},
			Output:        string(content),
			Deterministic: true,
			Timestamp:     time.Now(),
		},
	})

	report, err := Replay(root, p.Hash)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	if report.Fidelity != FidelityExact {
		t.Errorf("expected exact fidelity, got %s", report.Fidelity)
	}
	if len(report.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(report.Steps))
	}
	if report.Steps[0].Status != StepMatched {
		t.Errorf("expected matched, got %s", report.Steps[0].Status)
	}
	_ = contentHash
}

func TestReplayDegradedFidelity(t *testing.T) {
	root := setupTestStore(t)

	// Create a file with different content than what the pack recorded
	testFile := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(testFile, []byte("changed content"), 0644)

	p := createTestPack(t, root, []pack.LogStep{
		{
			Index:         0,
			Type:          "tool_call",
			Tool:          "read_file",
			Parameters:    map[string]interface{}{"path": testFile},
			Output:        "original content",
			Deterministic: true,
			Timestamp:     time.Now(),
		},
	})

	report, err := Replay(root, p.Hash)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	if report.Fidelity != FidelityDegraded {
		t.Errorf("expected degraded fidelity, got %s", report.Fidelity)
	}
	if report.Steps[0].Status != StepDiverged {
		t.Errorf("expected diverged, got %s", report.Steps[0].Status)
	}
}

func TestReplayFailedFidelity(t *testing.T) {
	root := setupTestStore(t)

	p := createTestPack(t, root, []pack.LogStep{
		{
			Index:         0,
			Type:          "tool_call",
			Tool:          "unavailable_tool",
			Parameters:    map[string]interface{}{},
			Output:        "output",
			Deterministic: true,
			Timestamp:     time.Now(),
		},
	})

	report, err := Replay(root, p.Hash)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	if report.Fidelity != FidelityFailed {
		t.Errorf("expected failed fidelity, got %s", report.Fidelity)
	}
	if report.Steps[0].Status != StepFailed {
		t.Errorf("expected failed, got %s", report.Steps[0].Status)
	}
	if report.Steps[0].Reason == "" {
		t.Error("expected reason for failed step")
	}
}

func TestReplayNonDeterministicStepDiverged(t *testing.T) {
	root := setupTestStore(t)

	testFile := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(testFile, []byte("actual output"), 0644)

	p := createTestPack(t, root, []pack.LogStep{
		{
			Index:         0,
			Type:          "tool_call",
			Tool:          "read_file",
			Parameters:    map[string]interface{}{"path": testFile},
			Output:        "different output",
			Deterministic: false, // non-deterministic
			Timestamp:     time.Now(),
		},
	})

	report, err := Replay(root, p.Hash)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	// Non-deterministic divergence should NOT downgrade fidelity
	if report.Fidelity != FidelityExact {
		t.Errorf("expected exact fidelity (non-det divergence doesn't downgrade), got %s", report.Fidelity)
	}
	if report.Steps[0].Status != StepDiverged {
		t.Errorf("expected diverged status, got %s", report.Steps[0].Status)
	}
}

func TestReplayNonDeterministicStepMatched(t *testing.T) {
	root := setupTestStore(t)

	testFile := filepath.Join(t.TempDir(), "test.txt")
	content := []byte("same content")
	os.WriteFile(testFile, content, 0644)

	p := createTestPack(t, root, []pack.LogStep{
		{
			Index:         0,
			Type:          "tool_call",
			Tool:          "read_file",
			Parameters:    map[string]interface{}{"path": testFile},
			Output:        string(content),
			Deterministic: false,
			Timestamp:     time.Now(),
		},
	})

	report, err := Replay(root, p.Hash)
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}

	if report.Fidelity != FidelityExact {
		t.Errorf("expected exact, got %s", report.Fidelity)
	}
	if report.Steps[0].Status != StepMatched {
		t.Errorf("expected matched, got %s", report.Steps[0].Status)
	}
}

func TestReplayNonExistentPack(t *testing.T) {
	root := setupTestStore(t)
	_, err := Replay(root, "sha256:0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Error("expected error for non-existent pack")
	}
}

func TestReplayReportSummary(t *testing.T) {
	report := &ReplayReport{
		PackHash: "sha256:abc123",
		Fidelity: FidelityExact,
		Steps: []StepResult{
			{Index: 0, Tool: "read_file", Status: StepMatched, Deterministic: true},
		},
		StartTime: time.Now(),
		EndTime:   time.Now(),
	}

	summary := report.Summary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}

func TestReplayReportJSON(t *testing.T) {
	report := &ReplayReport{
		PackHash: "sha256:abc123",
		Fidelity: FidelityExact,
		Steps:    []StepResult{},
	}

	data, err := report.JSON()
	if err != nil {
		t.Fatalf("JSON failed: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty JSON")
	}
}
