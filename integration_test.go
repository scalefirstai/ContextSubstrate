package integration_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/contextsubstrate/ctx/internal/delta"
	ctxdiff "github.com/contextsubstrate/ctx/internal/diff"
	"github.com/contextsubstrate/ctx/internal/graph"
	"github.com/contextsubstrate/ctx/internal/index"
	"github.com/contextsubstrate/ctx/internal/pack"
	"github.com/contextsubstrate/ctx/internal/replay"
	"github.com/contextsubstrate/ctx/internal/sharing"
	"github.com/contextsubstrate/ctx/internal/store"
	"github.com/contextsubstrate/ctx/internal/verify"
)

// TestEndToEnd exercises the full workflow: init → pack → show → replay → diff → verify → fork → log.
func TestEndToEnd(t *testing.T) {
	dir := t.TempDir()

	// === 1. Init ===
	root, err := store.InitStore(dir)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "objects")); err != nil {
		t.Fatalf("init: objects dir missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "packs")); err != nil {
		t.Fatalf("init: packs dir missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "config.json")); err != nil {
		t.Fatalf("init: config.json missing: %v", err)
	}
	// Verify graph directories created by init
	if _, err := os.Stat(filepath.Join(root, "graph", "manifests")); err != nil {
		t.Fatalf("init: graph/manifests dir missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "graph", "snapshots")); err != nil {
		t.Fatalf("init: graph/snapshots dir missing: %v", err)
	}
	t.Log("init: OK")

	// === 2. Pack ===
	logData := &pack.ExecutionLog{
		Model:        pack.LogModel{Identifier: "gpt-4", Parameters: map[string]interface{}{"temperature": 0.0}},
		SystemPrompt: "You are a helpful assistant.",
		Prompts:      []pack.LogPrompt{{Role: "user", Content: "Summarize this file"}},
		Inputs:       []pack.LogInput{{Name: "readme.md", Content: "# Hello World\nThis is a test project."}},
		Steps: []pack.LogStep{
			{Index: 0, Type: "tool_call", Tool: "read_file", Parameters: map[string]interface{}{"path": "readme.md"}, Output: "# Hello World\nThis is a test project.", Deterministic: true},
			{Index: 1, Type: "tool_call", Tool: "write_file", Parameters: map[string]interface{}{"path": "summary.txt"}, Output: "A test project readme.", Deterministic: false},
		},
		Outputs:     []pack.LogOutput{{Name: "summary.txt", Content: "A test project readme."}},
		Environment: pack.LogEnvironment{OS: "darwin", Runtime: "go1.22", ToolVersions: map[string]string{"read_file": "1.0", "write_file": "1.0"}},
	}

	p, err := pack.CreatePack(root, logData)
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	if p.Hash == "" {
		t.Fatal("pack: hash is empty")
	}
	if err := pack.RegisterPack(root, p.Hash); err != nil {
		t.Fatalf("pack register: %v", err)
	}
	t.Logf("pack: created %s", store.ShortHash(p.Hash, 12))

	// === 3. Show (load + format) ===
	loaded, err := pack.LoadPack(root, p.Hash)
	if err != nil {
		t.Fatalf("show: %v", err)
	}
	if loaded.Hash != p.Hash {
		t.Fatalf("show: hash mismatch: %s != %s", loaded.Hash, p.Hash)
	}
	if loaded.Model.Identifier != "gpt-4" {
		t.Fatalf("show: model mismatch: %s", loaded.Model.Identifier)
	}
	if len(loaded.Steps) != 2 {
		t.Fatalf("show: expected 2 steps, got %d", len(loaded.Steps))
	}
	formatted := pack.FormatPack(loaded)
	if formatted == "" {
		t.Fatal("show: format produced empty output")
	}
	t.Log("show: OK")

	// === 4. Replay ===
	report, err := replay.Replay(root, p.Hash)
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if len(report.Steps) != 2 {
		t.Fatalf("replay: expected 2 step results, got %d", len(report.Steps))
	}
	// read_file is a supported executor; write_file is not — expect one failed step
	if report.Fidelity == "" {
		t.Fatal("replay: fidelity not set")
	}
	summary := report.Summary()
	if summary == "" {
		t.Fatal("replay: summary is empty")
	}
	t.Logf("replay: fidelity=%s", report.Fidelity)

	// === 5. Diff ===
	// Create a second pack with different prompt
	logData2 := &pack.ExecutionLog{
		Model:        pack.LogModel{Identifier: "gpt-4", Parameters: map[string]interface{}{"temperature": 0.0}},
		SystemPrompt: "You are a concise assistant.",
		Prompts:      []pack.LogPrompt{{Role: "user", Content: "Summarize this file"}},
		Inputs:       []pack.LogInput{{Name: "readme.md", Content: "# Hello World\nThis is a test project."}},
		Steps: []pack.LogStep{
			{Index: 0, Type: "tool_call", Tool: "read_file", Parameters: map[string]interface{}{"path": "readme.md"}, Output: "# Hello World\nThis is a test project.", Deterministic: true},
		},
		Outputs:     []pack.LogOutput{{Name: "summary.txt", Content: "Test project."}},
		Environment: pack.LogEnvironment{OS: "darwin", Runtime: "go1.22", ToolVersions: map[string]string{"read_file": "1.0"}},
	}
	p2, err := pack.CreatePack(root, logData2)
	if err != nil {
		t.Fatalf("diff setup: %v", err)
	}
	if err := pack.RegisterPack(root, p2.Hash); err != nil {
		t.Fatalf("diff setup register: %v", err)
	}

	driftReport, err := ctxdiff.Diff(root, p.Hash, p2.Hash)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if !driftReport.HasDrift {
		t.Fatal("diff: expected drift between different packs")
	}
	// Should detect prompt drift (different system prompts)
	hasPromptDrift := false
	for _, entry := range driftReport.Entries {
		if entry.Type == ctxdiff.PromptDrift {
			hasPromptDrift = true
		}
	}
	if !hasPromptDrift {
		t.Fatal("diff: expected prompt_drift")
	}
	jsonData, err := driftReport.JSON()
	if err != nil {
		t.Fatalf("diff JSON: %v", err)
	}
	// Validate JSON is parseable
	var jsonCheck map[string]interface{}
	if err := json.Unmarshal(jsonData, &jsonCheck); err != nil {
		t.Fatalf("diff JSON parse: %v", err)
	}
	humanOutput := driftReport.Human()
	if humanOutput == "" {
		t.Fatal("diff: human output is empty")
	}
	t.Logf("diff: %d drift entries", len(driftReport.Entries))

	// === 6. Verify ===
	// Write the output artifact to a file, then generate sidecar + verify
	outputDir := filepath.Join(dir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("verify setup: %v", err)
	}
	artifactPath := filepath.Join(outputDir, "summary.txt")
	if err := os.WriteFile(artifactPath, []byte("A test project readme."), 0644); err != nil {
		t.Fatalf("verify setup write: %v", err)
	}

	count, err := verify.GenerateSidecars(p, outputDir)
	if err != nil {
		t.Fatalf("verify sidecar gen: %v", err)
	}
	if count != 1 {
		t.Fatalf("verify: expected 1 sidecar, got %d", count)
	}
	// Check sidecar file exists
	sidecarPath := verify.SidecarPath(artifactPath)
	if _, err := os.Stat(sidecarPath); err != nil {
		t.Fatalf("verify: sidecar file missing: %v", err)
	}

	result, err := verify.Verify(root, artifactPath)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !result.ContentMatch {
		t.Fatalf("verify: content should match (expected %s, actual %s)", result.ContentExpected, result.ContentActual)
	}
	if result.PackHash != p.Hash {
		t.Fatalf("verify: pack hash mismatch")
	}
	verifyOutput := verify.FormatVerifyResult(result)
	if verifyOutput == "" {
		t.Fatal("verify: format produced empty output")
	}
	t.Log("verify: integrity confirmed")

	// === 7. Fork ===
	draftPath, err := sharing.Fork(root, p.Hash)
	if err != nil {
		t.Fatalf("fork: %v", err)
	}
	if _, err := os.Stat(draftPath); err != nil {
		t.Fatalf("fork: draft not found: %v", err)
	}

	// Read draft and verify parent
	draftData, err := os.ReadFile(draftPath)
	if err != nil {
		t.Fatalf("fork read: %v", err)
	}
	var draft pack.Pack
	if err := json.Unmarshal(draftData, &draft); err != nil {
		t.Fatalf("fork parse: %v", err)
	}
	if draft.Parent != p.Hash {
		t.Fatalf("fork: parent should be %s, got %s", p.Hash, draft.Parent)
	}

	// Finalize
	finalized, err := sharing.FinalizeDraft(root, draftPath)
	if err != nil {
		t.Fatalf("fork finalize: %v", err)
	}
	if finalized.Parent != p.Hash {
		t.Fatalf("fork finalize: parent not preserved")
	}
	if finalized.Hash == p.Hash {
		t.Fatal("fork finalize: hash should differ from parent")
	}
	if _, err := os.Stat(draftPath); !os.IsNotExist(err) {
		t.Fatal("fork finalize: draft should be removed")
	}
	t.Logf("fork: finalized %s (parent: %s)", store.ShortHash(finalized.Hash, 12), store.ShortHash(finalized.Parent, 12))

	// === 8. Log ===
	summaries, err := sharing.ListPacks(root, 50)
	if err != nil {
		t.Fatalf("log: %v", err)
	}
	// Should have 3 packs: original, second, forked
	if len(summaries) != 3 {
		t.Fatalf("log: expected 3 packs, got %d", len(summaries))
	}

	// Check that at least one has a parent (the forked one)
	hasParent := false
	for _, s := range summaries {
		if s.Parent != "" {
			hasParent = true
		}
	}
	if !hasParent {
		t.Fatal("log: expected at least one pack with parent")
	}

	listOutput := sharing.FormatPackList(summaries)
	if listOutput == "" {
		t.Fatal("log: format produced empty output")
	}
	t.Log("log: OK")

	t.Log("=== End-to-end test complete ===")
}

// TestEndToEndSelfDiffNoDrift verifies that diffing a pack against itself yields no drift.
func TestEndToEndSelfDiffNoDrift(t *testing.T) {
	dir := t.TempDir()
	root, err := store.InitStore(dir)
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	logData := &pack.ExecutionLog{
		Model:        pack.LogModel{Identifier: "test-model", Parameters: map[string]interface{}{}},
		SystemPrompt: "test",
		Prompts:      []pack.LogPrompt{{Role: "user", Content: "test"}},
		Inputs:       []pack.LogInput{},
		Steps:        []pack.LogStep{{Index: 0, Type: "tool_call", Tool: "test", Parameters: map[string]interface{}{}, Output: "out", Deterministic: true}},
		Outputs:      []pack.LogOutput{{Name: "out.txt", Content: "output"}},
		Environment:  pack.LogEnvironment{OS: "darwin", Runtime: "go1.22", ToolVersions: map[string]string{}},
	}

	p, err := pack.CreatePack(root, logData)
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	pack.RegisterPack(root, p.Hash)

	report, err := ctxdiff.Diff(root, p.Hash, p.Hash)
	if err != nil {
		t.Fatalf("diff: %v", err)
	}
	if report.HasDrift {
		t.Fatal("self-diff should have no drift")
	}
}

// TestEndToEndIndexDelta exercises the init → index → delta workflow.
func TestEndToEndIndexDelta(t *testing.T) {
	// Create a git repo with multiple commits
	repoDir := t.TempDir()

	gitRun := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %s: %v", args, out, err)
		}
	}

	gitRun("init")
	gitRun("checkout", "-b", "main")

	// Commit 1: initial files
	os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644)
	os.WriteFile(filepath.Join(repoDir, "README.md"), []byte("# Project\n\nA test project.\n"), 0644)
	os.WriteFile(filepath.Join(repoDir, "config.yaml"), []byte("version: 1\n"), 0644)
	gitRun("add", ".")
	gitRun("commit", "-m", "initial commit")

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	commit1 := strings.TrimSpace(string(out))

	// Commit 2: modify main.go, add util.go, delete config.yaml
	os.WriteFile(filepath.Join(repoDir, "main.go"), []byte("package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"), 0644)
	os.WriteFile(filepath.Join(repoDir, "util.go"), []byte("package main\n\nfunc helper() string { return \"hi\" }\n"), 0644)
	os.Remove(filepath.Join(repoDir, "config.yaml"))
	gitRun("add", ".")
	gitRun("commit", "-m", "update project")

	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoDir
	out, err = cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	commit2 := strings.TrimSpace(string(out))

	// === 1. Init store ===
	storeDir := t.TempDir()
	root, err := store.InitStore(storeDir)
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	t.Logf("init: store at %s", root)

	// Verify graph directories
	if _, err := os.Stat(filepath.Join(root, "graph", "manifests")); err != nil {
		t.Fatalf("graph/manifests missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "graph", "snapshots")); err != nil {
		t.Fatalf("graph/snapshots missing: %v", err)
	}

	// === 2. Index commit 1 ===
	if err := index.IndexCommit(root, repoDir, commit1); err != nil {
		t.Fatalf("index commit1: %v", err)
	}
	t.Logf("index: indexed %s", commit1[:8])

	// Verify JSONL files exist
	files1, err := graph.ReadRecords[graph.FileSnapshot](graph.FilesPath(root, commit1))
	if err != nil {
		t.Fatalf("reading files for commit1: %v", err)
	}
	if len(files1) != 3 {
		t.Fatalf("commit1: expected 3 file snapshots, got %d", len(files1))
	}
	t.Logf("index: commit1 has %d files", len(files1))

	// Check file properties
	for _, f := range files1 {
		if f.ContentSHA256 == "" {
			t.Error("file snapshot missing content hash")
		}
		if f.ByteSize == 0 {
			t.Error("file snapshot has zero byte size")
		}
	}

	// Verify commit record
	commits, err := graph.ReadRecords[graph.CommitRecord](graph.CommitsPath(root))
	if err != nil {
		t.Fatalf("reading commits: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit record, got %d", len(commits))
	}
	if commits[0].SHA != commit1 {
		t.Errorf("commit SHA: got %q, want %q", commits[0].SHA, commit1)
	}
	if commits[0].Message != "initial commit" {
		t.Errorf("commit message: got %q", commits[0].Message)
	}

	// === 3. Index commit 2 ===
	if err := index.IndexCommit(root, repoDir, commit2); err != nil {
		t.Fatalf("index commit2: %v", err)
	}
	t.Logf("index: indexed %s", commit2[:8])

	files2, err := graph.ReadRecords[graph.FileSnapshot](graph.FilesPath(root, commit2))
	if err != nil {
		t.Fatalf("reading files for commit2: %v", err)
	}
	if len(files2) != 3 { // main.go, util.go, README.md (config.yaml deleted)
		t.Fatalf("commit2: expected 3 file snapshots, got %d", len(files2))
	}

	// === 4. Compute delta ===
	deltaReport, err := delta.ComputeDelta(root, commit1, commit2)
	if err != nil {
		t.Fatalf("delta: %v", err)
	}
	t.Logf("delta: %d changed, %d added, %d deleted",
		len(deltaReport.FilesChanged), len(deltaReport.FilesAdded), len(deltaReport.FilesDeleted))

	if deltaReport.IsEmpty() {
		t.Fatal("delta should not be empty")
	}

	// main.go was modified
	if len(deltaReport.FilesChanged) != 1 || deltaReport.FilesChanged[0] != "main.go" {
		t.Errorf("FilesChanged: got %v, want [main.go]", deltaReport.FilesChanged)
	}

	// util.go was added
	if len(deltaReport.FilesAdded) != 1 || deltaReport.FilesAdded[0] != "util.go" {
		t.Errorf("FilesAdded: got %v, want [util.go]", deltaReport.FilesAdded)
	}

	// config.yaml was deleted
	if len(deltaReport.FilesDeleted) != 1 || deltaReport.FilesDeleted[0] != "config.yaml" {
		t.Errorf("FilesDeleted: got %v, want [config.yaml]", deltaReport.FilesDeleted)
	}

	// Verify JSON output
	jsonData, err := deltaReport.JSON()
	if err != nil {
		t.Fatalf("delta JSON: %v", err)
	}
	var jsonCheck map[string]interface{}
	if err := json.Unmarshal(jsonData, &jsonCheck); err != nil {
		t.Fatalf("delta JSON parse: %v", err)
	}

	// Verify human output
	humanOutput := deltaReport.Human()
	if humanOutput == "" {
		t.Fatal("delta: human output is empty")
	}
	if !strings.Contains(humanOutput, "main.go") {
		t.Error("human output should mention main.go")
	}

	// === 5. Verify self-delta is empty ===
	selfDelta, err := delta.ComputeDelta(root, commit2, commit2)
	if err != nil {
		t.Fatalf("self-delta: %v", err)
	}
	if !selfDelta.IsEmpty() {
		t.Error("self-delta should be empty")
	}

	// === 6. Verify JSONL determinism ===
	// Re-index commit1 (should be idempotent — skipped)
	if err := index.IndexCommit(root, repoDir, commit1); err != nil {
		t.Fatalf("re-index: %v", err)
	}

	// File snapshots should be identical
	files1Again, err := graph.ReadRecords[graph.FileSnapshot](graph.FilesPath(root, commit1))
	if err != nil {
		t.Fatal(err)
	}
	if len(files1Again) != len(files1) {
		t.Fatalf("re-index changed file count: %d vs %d", len(files1Again), len(files1))
	}
	for i := range files1 {
		if files1[i].ContentSHA256 != files1Again[i].ContentSHA256 {
			t.Errorf("file %d hash mismatch after re-index", i)
		}
	}

	// Verify path records accumulated correctly
	paths, err := graph.ReadRecords[graph.PathRecord](graph.PathsPath(root))
	if err != nil {
		t.Fatal(err)
	}
	// Should have 4 unique paths: main.go, README.md, config.yaml, util.go
	if len(paths) != 4 {
		t.Fatalf("expected 4 path records, got %d", len(paths))
	}

	t.Log("=== Index → Delta integration test complete ===")
}
