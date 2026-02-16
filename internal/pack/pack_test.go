package pack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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

func sampleLog() *ExecutionLog {
	return &ExecutionLog{
		Model: LogModel{
			Identifier: "claude-opus-4-6",
			Parameters: map[string]interface{}{"temperature": float64(0)},
		},
		SystemPrompt: "You are a helpful assistant.",
		Prompts: []LogPrompt{
			{Role: "user", Content: "Write hello world."},
		},
		Inputs: []LogInput{
			{Name: "main.go", Content: "package main\n"},
		},
		Steps: []LogStep{
			{
				Index:         0,
				Type:          "tool_call",
				Tool:          "write_file",
				Parameters:    map[string]interface{}{"path": "main.go"},
				Output:        "package main\n\nfunc main() {}\n",
				Deterministic: true,
			},
		},
		Outputs: []LogOutput{
			{Name: "main.go", Content: "package main\n\nfunc main() {}\n"},
		},
		Environment: LogEnvironment{
			OS:           "darwin",
			Runtime:      "go1.22",
			ToolVersions: map[string]string{"ctx": "0.1.0"},
		},
	}
}

func TestCreatePack(t *testing.T) {
	root := setupTestStore(t)
	log := sampleLog()

	p, err := CreatePack(root, log)
	if err != nil {
		t.Fatalf("CreatePack failed: %v", err)
	}

	if p.Version != "0.1" {
		t.Errorf("expected version 0.1, got %s", p.Version)
	}
	if !store.ValidateHash(p.Hash) {
		t.Errorf("invalid pack hash: %s", p.Hash)
	}
	if p.Model.Identifier != "claude-opus-4-6" {
		t.Errorf("expected model claude-opus-4-6, got %s", p.Model.Identifier)
	}
	if !store.ValidateHash(p.SystemPrompt) {
		t.Errorf("system prompt should be a hash ref, got %s", p.SystemPrompt)
	}
	if len(p.Prompts) != 1 {
		t.Errorf("expected 1 prompt, got %d", len(p.Prompts))
	}
	if len(p.Inputs) != 1 {
		t.Errorf("expected 1 input, got %d", len(p.Inputs))
	}
	if p.Inputs[0].Size != 13 {
		t.Errorf("expected input size 13, got %d", p.Inputs[0].Size)
	}
	if len(p.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(p.Steps))
	}
	if len(p.Outputs) != 1 {
		t.Errorf("expected 1 output, got %d", len(p.Outputs))
	}
}

func TestCreatePackManifestFields(t *testing.T) {
	root := setupTestStore(t)
	p, err := CreatePack(root, sampleLog())
	if err != nil {
		t.Fatalf("CreatePack failed: %v", err)
	}

	if err := p.Validate(); err != nil {
		t.Errorf("pack validation failed: %v", err)
	}
}

func TestCreatePackBlobsStored(t *testing.T) {
	root := setupTestStore(t)
	p, err := CreatePack(root, sampleLog())
	if err != nil {
		t.Fatalf("CreatePack failed: %v", err)
	}

	// Check that all referenced blobs exist
	if !store.BlobExists(root, p.SystemPrompt) {
		t.Error("system prompt blob not found")
	}
	for _, prompt := range p.Prompts {
		if !store.BlobExists(root, prompt.ContentRef) {
			t.Errorf("prompt blob not found: %s", prompt.ContentRef)
		}
	}
	for _, input := range p.Inputs {
		if !store.BlobExists(root, input.ContentRef) {
			t.Errorf("input blob not found: %s", input.ContentRef)
		}
	}
	for _, step := range p.Steps {
		if step.OutputRef != "" && !store.BlobExists(root, step.OutputRef) {
			t.Errorf("step output blob not found: %s", step.OutputRef)
		}
	}
}

func TestCreatePackDeterministicHash(t *testing.T) {
	// Two packs from the same log should produce the same hash
	// (as long as Created timestamp is the same, which we can't control here)
	// Instead, verify that loading a pack produces consistent data
	root := setupTestStore(t)
	p, err := CreatePack(root, sampleLog())
	if err != nil {
		t.Fatalf("CreatePack failed: %v", err)
	}

	loaded, err := LoadPack(root, p.Hash)
	if err != nil {
		t.Fatalf("LoadPack failed: %v", err)
	}

	if loaded.Hash != p.Hash {
		t.Errorf("loaded hash %s != created hash %s", loaded.Hash, p.Hash)
	}
	if loaded.Model.Identifier != p.Model.Identifier {
		t.Errorf("model mismatch after load")
	}
}

func TestCreatePackDeduplication(t *testing.T) {
	root := setupTestStore(t)
	log := sampleLog()

	// Create two packs with same content
	p1, err := CreatePack(root, log)
	if err != nil {
		t.Fatalf("first CreatePack failed: %v", err)
	}
	p2, err := CreatePack(root, log)
	if err != nil {
		t.Fatalf("second CreatePack failed: %v", err)
	}

	// Content blobs should be reused (same refs)
	if p1.SystemPrompt != p2.SystemPrompt {
		t.Error("system prompt blobs not deduplicated")
	}
	if p1.Inputs[0].ContentRef != p2.Inputs[0].ContentRef {
		t.Error("input blobs not deduplicated")
	}
}

func TestLoadPackNotFound(t *testing.T) {
	root := setupTestStore(t)
	_, err := LoadPack(root, "sha256:0000000000000000000000000000000000000000000000000000000000000000")
	if err == nil {
		t.Error("expected error for non-existent pack")
	}
}

func TestRegisterPack(t *testing.T) {
	root := setupTestStore(t)
	p, err := CreatePack(root, sampleLog())
	if err != nil {
		t.Fatalf("CreatePack failed: %v", err)
	}

	if err := RegisterPack(root, p.Hash); err != nil {
		t.Fatalf("RegisterPack failed: %v", err)
	}

	_, hex, _ := store.ParseHash(p.Hash)
	path := filepath.Join(root, "packs", hex)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("pack not registered: %v", err)
	}
}

func TestParseExecutionLogValid(t *testing.T) {
	json := `{
		"model": {"identifier": "test-model", "parameters": {}},
		"system_prompt": "test prompt",
		"prompts": [{"role": "user", "content": "hello"}],
		"inputs": [],
		"steps": [{"index": 0, "type": "tool_call", "tool": "test", "parameters": {}, "output": "out", "deterministic": true, "timestamp": "2026-01-01T00:00:00Z"}],
		"outputs": [{"name": "result.txt", "content": "result"}],
		"environment": {"os": "linux", "runtime": "go1.22", "tool_versions": {}}
	}`

	log, err := ParseExecutionLogReader(strings.NewReader(json))
	if err != nil {
		t.Fatalf("ParseExecutionLogReader failed: %v", err)
	}
	if log.Model.Identifier != "test-model" {
		t.Errorf("unexpected model: %s", log.Model.Identifier)
	}
}

func TestParseExecutionLogMissingFields(t *testing.T) {
	json := `{
		"model": {"identifier": "", "parameters": {}},
		"system_prompt": "",
		"prompts": [],
		"inputs": [],
		"steps": [],
		"outputs": [],
		"environment": {"os": "", "runtime": "", "tool_versions": {}}
	}`

	_, err := ParseExecutionLogReader(strings.NewReader(json))
	if err == nil {
		t.Fatal("expected error for missing fields")
	}
	// Should report multiple missing fields
	errStr := err.Error()
	if !strings.Contains(errStr, "model.identifier") {
		t.Errorf("expected model.identifier in error, got: %s", errStr)
	}
	if !strings.Contains(errStr, "system_prompt") {
		t.Errorf("expected system_prompt in error, got: %s", errStr)
	}
}

func TestParseExecutionLogMalformed(t *testing.T) {
	_, err := ParseExecutionLogReader(strings.NewReader("not json"))
	if err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
