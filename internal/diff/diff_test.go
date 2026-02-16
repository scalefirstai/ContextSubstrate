package diff

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

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

func createPack(t *testing.T, root string, sysPrompt string, prompts []pack.LogPrompt, steps []pack.LogStep, outputs []pack.LogOutput) *pack.Pack {
	t.Helper()
	log := &pack.ExecutionLog{
		Model:        pack.LogModel{Identifier: "test-model", Parameters: map[string]interface{}{}},
		SystemPrompt: sysPrompt,
		Prompts:      prompts,
		Inputs:       []pack.LogInput{},
		Steps:        steps,
		Outputs:      outputs,
		Environment:  pack.LogEnvironment{OS: "darwin", Runtime: "go1.22", ToolVersions: map[string]string{}},
	}
	p, err := pack.CreatePack(root, log)
	if err != nil {
		t.Fatalf("CreatePack failed: %v", err)
	}
	return p
}

func defaultSteps() []pack.LogStep {
	return []pack.LogStep{
		{Index: 0, Type: "tool_call", Tool: "read_file", Parameters: map[string]interface{}{"path": "a.txt"}, Output: "content", Deterministic: true},
	}
}

func defaultOutputs() []pack.LogOutput {
	return []pack.LogOutput{{Name: "result.txt", Content: "result"}}
}

func defaultPrompts() []pack.LogPrompt {
	return []pack.LogPrompt{{Role: "user", Content: "do something"}}
}

func TestDiffIdenticalPacks(t *testing.T) {
	root := setupTestStore(t)
	p := createPack(t, root, "system prompt", defaultPrompts(), defaultSteps(), defaultOutputs())

	report, err := Diff(root, p.Hash, p.Hash)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	if report.HasDrift {
		t.Error("expected no drift for identical packs")
	}
	if len(report.Entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(report.Entries))
	}
}

func TestDiffPromptDrift(t *testing.T) {
	root := setupTestStore(t)
	a := createPack(t, root, "prompt A", defaultPrompts(), defaultSteps(), defaultOutputs())
	b := createPack(t, root, "prompt B", defaultPrompts(), defaultSteps(), defaultOutputs())

	report, err := Diff(root, a.Hash, b.Hash)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	if !report.HasDrift {
		t.Error("expected drift")
	}

	found := false
	for _, e := range report.Entries {
		if e.Type == PromptDrift {
			found = true
		}
	}
	if !found {
		t.Error("expected prompt_drift entry")
	}
}

func TestDiffUserPromptDrift(t *testing.T) {
	root := setupTestStore(t)
	a := createPack(t, root, "sys", []pack.LogPrompt{{Role: "user", Content: "query A"}}, defaultSteps(), defaultOutputs())
	b := createPack(t, root, "sys", []pack.LogPrompt{{Role: "user", Content: "query B"}}, defaultSteps(), defaultOutputs())

	report, err := Diff(root, a.Hash, b.Hash)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	found := false
	for _, e := range report.Entries {
		if e.Type == PromptDrift && e.StepIndex == 0 {
			found = true
		}
	}
	if !found {
		t.Error("expected prompt_drift for user prompt at index 0")
	}
}

func TestDiffToolDrift(t *testing.T) {
	root := setupTestStore(t)
	stepsA := []pack.LogStep{{Index: 0, Type: "tool_call", Tool: "read_file", Parameters: map[string]interface{}{}, Output: "out", Deterministic: true}}
	stepsB := []pack.LogStep{{Index: 0, Type: "tool_call", Tool: "write_file", Parameters: map[string]interface{}{}, Output: "out", Deterministic: true}}

	a := createPack(t, root, "sys", defaultPrompts(), stepsA, defaultOutputs())
	b := createPack(t, root, "sys", defaultPrompts(), stepsB, defaultOutputs())

	report, err := Diff(root, a.Hash, b.Hash)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	found := false
	for _, e := range report.Entries {
		if e.Type == ToolDrift {
			found = true
		}
	}
	if !found {
		t.Error("expected tool_drift entry")
	}
}

func TestDiffParamDrift(t *testing.T) {
	root := setupTestStore(t)
	stepsA := []pack.LogStep{{Index: 0, Type: "tool_call", Tool: "read_file", Parameters: map[string]interface{}{"path": "a.txt"}, Output: "out", Deterministic: true}}
	stepsB := []pack.LogStep{{Index: 0, Type: "tool_call", Tool: "read_file", Parameters: map[string]interface{}{"path": "b.txt"}, Output: "out", Deterministic: true}}

	a := createPack(t, root, "sys", defaultPrompts(), stepsA, defaultOutputs())
	b := createPack(t, root, "sys", defaultPrompts(), stepsB, defaultOutputs())

	report, err := Diff(root, a.Hash, b.Hash)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	found := false
	for _, e := range report.Entries {
		if e.Type == ParamDrift {
			found = true
		}
	}
	if !found {
		t.Error("expected param_drift entry")
	}
}

func TestDiffReasoningDrift(t *testing.T) {
	root := setupTestStore(t)
	stepsA := []pack.LogStep{{Index: 0, Type: "tool_call", Tool: "read_file", Parameters: map[string]interface{}{"path": "a.txt"}, Output: "output A", Deterministic: true}}
	stepsB := []pack.LogStep{{Index: 0, Type: "tool_call", Tool: "read_file", Parameters: map[string]interface{}{"path": "a.txt"}, Output: "output B", Deterministic: true}}

	a := createPack(t, root, "sys", defaultPrompts(), stepsA, defaultOutputs())
	b := createPack(t, root, "sys", defaultPrompts(), stepsB, defaultOutputs())

	report, err := Diff(root, a.Hash, b.Hash)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	found := false
	for _, e := range report.Entries {
		if e.Type == ReasoningDrift {
			found = true
		}
	}
	if !found {
		t.Error("expected reasoning_drift entry")
	}
}

func TestDiffOutputDrift(t *testing.T) {
	root := setupTestStore(t)
	a := createPack(t, root, "sys", defaultPrompts(), defaultSteps(), []pack.LogOutput{{Name: "result.txt", Content: "result A"}})
	b := createPack(t, root, "sys", defaultPrompts(), defaultSteps(), []pack.LogOutput{{Name: "result.txt", Content: "result B"}})

	report, err := Diff(root, a.Hash, b.Hash)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	found := false
	for _, e := range report.Entries {
		if e.Type == OutputDrift {
			found = true
		}
	}
	if !found {
		t.Error("expected output_drift entry")
	}
}

func TestDiffStepCountMismatch(t *testing.T) {
	root := setupTestStore(t)
	stepsA := []pack.LogStep{
		{Index: 0, Type: "tool_call", Tool: "read_file", Parameters: map[string]interface{}{}, Output: "out", Deterministic: true},
		{Index: 1, Type: "tool_call", Tool: "write_file", Parameters: map[string]interface{}{}, Output: "out2", Deterministic: true},
	}
	stepsB := []pack.LogStep{
		{Index: 0, Type: "tool_call", Tool: "read_file", Parameters: map[string]interface{}{}, Output: "out", Deterministic: true},
	}

	a := createPack(t, root, "sys", defaultPrompts(), stepsA, defaultOutputs())
	b := createPack(t, root, "sys", defaultPrompts(), stepsB, defaultOutputs())

	report, err := Diff(root, a.Hash, b.Hash)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	// Should report the extra step as removed
	found := false
	for _, e := range report.Entries {
		if e.Type == ToolDrift && e.StepIndex == 1 {
			found = true
		}
	}
	if !found {
		t.Error("expected tool_drift for removed step")
	}
}

func TestDiffJSONOutput(t *testing.T) {
	root := setupTestStore(t)
	a := createPack(t, root, "sys A", defaultPrompts(), defaultSteps(), defaultOutputs())
	b := createPack(t, root, "sys B", defaultPrompts(), defaultSteps(), defaultOutputs())

	report, err := Diff(root, a.Hash, b.Hash)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	data, err := report.JSON()
	if err != nil {
		t.Fatalf("JSON failed: %v", err)
	}

	var parsed DriftReport
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON output not parseable: %v", err)
	}
	if !parsed.HasDrift {
		t.Error("expected has_drift=true in JSON")
	}
}

func TestDiffHumanOutput(t *testing.T) {
	root := setupTestStore(t)
	a := createPack(t, root, "sys A", defaultPrompts(), defaultSteps(), defaultOutputs())
	b := createPack(t, root, "sys B", defaultPrompts(), defaultSteps(), defaultOutputs())

	report, err := Diff(root, a.Hash, b.Hash)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	human := report.Human()
	if human == "" {
		t.Error("expected non-empty human output")
	}
}

func TestDiffHumanNoDrift(t *testing.T) {
	root := setupTestStore(t)
	p := createPack(t, root, "sys", defaultPrompts(), defaultSteps(), defaultOutputs())

	report, err := Diff(root, p.Hash, p.Hash)
	if err != nil {
		t.Fatalf("Diff failed: %v", err)
	}

	human := report.Human()
	if human != "No differences found.\n" {
		t.Errorf("expected 'No differences found.', got %q", human)
	}
}

func TestDiffNonExistentPack(t *testing.T) {
	root := setupTestStore(t)
	fakeHash := "sha256:0000000000000000000000000000000000000000000000000000000000000000"
	p := createPack(t, root, "sys", defaultPrompts(), defaultSteps(), defaultOutputs())

	_, err := Diff(root, p.Hash, fakeHash)
	if err == nil {
		t.Error("expected error for non-existent pack")
	}
	_ = store.ShortHash(fakeHash, 12) // just to use the import
}
