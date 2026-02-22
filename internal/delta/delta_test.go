package delta

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/contextsubstrate/ctx/internal/graph"
	"github.com/contextsubstrate/ctx/internal/index"
)

// setupTestRepoAndStore creates a git repo with 2 commits and a store with both indexed.
func setupTestRepoAndStore(t *testing.T) (storeRoot, repoRoot, baseSHA, headSHA string) {
	t.Helper()

	repoRoot = t.TempDir()
	storeRoot = filepath.Join(t.TempDir(), ".ctx")
	os.MkdirAll(storeRoot, 0755)

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = repoRoot
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@test.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@test.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("command %v failed: %s: %v", args, out, err)
		}
	}

	run("git", "init")
	run("git", "checkout", "-b", "main")

	// First commit
	os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644)
	os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Test\n"), 0644)
	run("git", "add", ".")
	run("git", "commit", "-m", "initial")

	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoRoot
	out, _ := cmd.Output()
	baseSHA = trimNewline(string(out))

	// Second commit: modify main.go, add util.go, delete README.md
	os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte("package main\n\nimport \"fmt\"\n\nfunc main() { fmt.Println(\"hi\") }\n"), 0644)
	os.WriteFile(filepath.Join(repoRoot, "util.go"), []byte("package main\n\nfunc helper() {}\n"), 0644)
	os.Remove(filepath.Join(repoRoot, "README.md"))
	run("git", "add", ".")
	run("git", "commit", "-m", "update")

	cmd = exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoRoot
	out, _ = cmd.Output()
	headSHA = trimNewline(string(out))

	// Index both commits
	if err := index.IndexCommit(storeRoot, repoRoot, baseSHA); err != nil {
		t.Fatalf("IndexCommit base: %v", err)
	}
	if err := index.IndexCommit(storeRoot, repoRoot, headSHA); err != nil {
		t.Fatalf("IndexCommit head: %v", err)
	}

	return
}

func TestComputeDelta(t *testing.T) {
	storeRoot, _, baseSHA, headSHA := setupTestRepoAndStore(t)

	report, err := ComputeDelta(storeRoot, baseSHA, headSHA)
	if err != nil {
		t.Fatalf("ComputeDelta: %v", err)
	}

	if report.Base != baseSHA {
		t.Errorf("Base: got %q, want %q", report.Base, baseSHA)
	}
	if report.Head != headSHA {
		t.Errorf("Head: got %q, want %q", report.Head, headSHA)
	}

	// main.go was modified
	if len(report.FilesChanged) != 1 || report.FilesChanged[0] != "main.go" {
		t.Errorf("FilesChanged: got %v, want [main.go]", report.FilesChanged)
	}

	// util.go was added
	if len(report.FilesAdded) != 1 || report.FilesAdded[0] != "util.go" {
		t.Errorf("FilesAdded: got %v, want [util.go]", report.FilesAdded)
	}

	// README.md was deleted
	if len(report.FilesDeleted) != 1 || report.FilesDeleted[0] != "README.md" {
		t.Errorf("FilesDeleted: got %v, want [README.md]", report.FilesDeleted)
	}
}

func TestComputeDeltaSelfNoop(t *testing.T) {
	storeRoot, _, _, headSHA := setupTestRepoAndStore(t)

	report, err := ComputeDelta(storeRoot, headSHA, headSHA)
	if err != nil {
		t.Fatalf("ComputeDelta self: %v", err)
	}

	if !report.IsEmpty() {
		t.Errorf("self-delta should be empty, got: changed=%v, added=%v, deleted=%v",
			report.FilesChanged, report.FilesAdded, report.FilesDeleted)
	}
}

func TestDeltaReportJSON(t *testing.T) {
	report := &DeltaReport{
		Base:         "abc123",
		Head:         "def456",
		FilesChanged: []string{"main.go"},
		FilesAdded:   []string{"new.go"},
		FilesDeleted: []string{"old.go"},
	}

	data, err := report.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}

	if parsed["base"] != "abc123" {
		t.Errorf("base: got %v", parsed["base"])
	}
}

func TestDeltaReportHuman(t *testing.T) {
	report := &DeltaReport{
		Base:         "abc12345678",
		Head:         "def45678901",
		FilesChanged: []string{"main.go"},
		FilesAdded:   []string{"new.go"},
		FilesDeleted: []string{"old.go"},
	}

	human := report.Human()
	if human == "" {
		t.Fatal("Human() returned empty string")
	}

	// Check it contains expected sections
	for _, want := range []string{"abc12345", "def45678", "+ new.go", "- old.go", "~ main.go"} {
		if !containsStr(human, want) {
			t.Errorf("Human() missing %q in output:\n%s", want, human)
		}
	}
}

func TestDeltaReportEmpty(t *testing.T) {
	report := &DeltaReport{Base: "aaa", Head: "bbb"}

	if !report.IsEmpty() {
		t.Error("expected empty report")
	}

	human := report.Human()
	if !containsStr(human, "No changes detected") {
		t.Errorf("expected 'No changes detected' in: %s", human)
	}
}

func TestComputeDeltaUnindexedCommit(t *testing.T) {
	storeRoot := filepath.Join(t.TempDir(), ".ctx")
	os.MkdirAll(storeRoot, 0755)
	graph.InitGraph(storeRoot)

	// Should handle gracefully (empty file list for unindexed commits)
	report, err := ComputeDelta(storeRoot, "nonexistent1", "nonexistent2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.IsEmpty() {
		t.Error("delta of unindexed commits should be empty")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func trimNewline(s string) string {
	for len(s) > 0 && (s[len(s)-1] == '\n' || s[len(s)-1] == '\r') {
		s = s[:len(s)-1]
	}
	return s
}
