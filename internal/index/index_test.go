package index

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/contextsubstrate/ctx/internal/graph"
)

// setupTestRepo creates a temporary git repo with some initial commits.
func setupTestRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
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
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "initial commit")

	// Second commit
	if err := os.WriteFile(filepath.Join(dir, "util.go"), []byte("package main\n\nfunc helper() string {\n\treturn \"hello\"\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(helper())\n}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", ".")
	run("git", "commit", "-m", "add util and update main")

	return dir
}

func TestDetectChanges(t *testing.T) {
	repoRoot := setupTestRepo(t)

	// Get the two commit SHAs
	cmd := exec.Command("git", "log", "--format=%H", "--reverse")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}

	commits := splitLines(string(out))
	if len(commits) < 2 {
		t.Fatalf("expected at least 2 commits, got %d", len(commits))
	}

	cs, err := DetectChanges(repoRoot, commits[0], commits[1])
	if err != nil {
		t.Fatalf("DetectChanges: %v", err)
	}

	if len(cs.FilesAdded) != 1 || cs.FilesAdded[0] != "util.go" {
		t.Errorf("FilesAdded: got %v, want [util.go]", cs.FilesAdded)
	}
	if len(cs.FilesChanged) != 1 || cs.FilesChanged[0] != "main.go" {
		t.Errorf("FilesChanged: got %v, want [main.go]", cs.FilesChanged)
	}
	if len(cs.FilesDeleted) != 0 {
		t.Errorf("FilesDeleted: got %v, want []", cs.FilesDeleted)
	}
}

func TestListFilesAtCommit(t *testing.T) {
	repoRoot := setupTestRepo(t)

	// Get first commit SHA
	cmd := exec.Command("git", "log", "--format=%H", "--reverse")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}

	commits := splitLines(string(out))
	files, err := ListFilesAtCommit(repoRoot, commits[0])
	if err != nil {
		t.Fatalf("ListFilesAtCommit: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files in first commit, got %d: %v", len(files), files)
	}
}

func TestGetCommitInfo(t *testing.T) {
	repoRoot := setupTestRepo(t)

	sha, err := GetHeadSHA(repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	info, err := GetCommitInfo(repoRoot, sha)
	if err != nil {
		t.Fatalf("GetCommitInfo: %v", err)
	}

	if info.SHA != sha {
		t.Errorf("SHA: got %q, want %q", info.SHA, sha)
	}
	if info.Message != "add util and update main" {
		t.Errorf("Message: got %q", info.Message)
	}
	if info.ParentSHA == "" {
		t.Error("expected parent SHA")
	}
}

func TestIndexCommit(t *testing.T) {
	repoRoot := setupTestRepo(t)
	storeRoot := filepath.Join(t.TempDir(), ".ctx")
	os.MkdirAll(storeRoot, 0755)

	sha, err := GetHeadSHA(repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	if err := IndexCommit(storeRoot, repoRoot, sha); err != nil {
		t.Fatalf("IndexCommit: %v", err)
	}

	// Verify commit record
	commits, err := graph.ReadRecords[graph.CommitRecord](graph.CommitsPath(storeRoot))
	if err != nil {
		t.Fatalf("reading commits: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit record, got %d", len(commits))
	}
	if commits[0].SHA != sha {
		t.Errorf("commit SHA: got %q, want %q", commits[0].SHA, sha)
	}

	// Verify file snapshots
	files, err := graph.ReadRecords[graph.FileSnapshot](graph.FilesPath(storeRoot, sha))
	if err != nil {
		t.Fatalf("reading files: %v", err)
	}
	if len(files) != 3 { // main.go, util.go, README.md
		t.Fatalf("expected 3 file snapshots, got %d", len(files))
	}

	// Check a specific file
	found := false
	for _, f := range files {
		if f.Language == "go" && f.LOC > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected at least one Go file with LOC > 0")
	}

	// Verify path records
	paths, err := graph.ReadRecords[graph.PathRecord](graph.PathsPath(storeRoot))
	if err != nil {
		t.Fatalf("reading paths: %v", err)
	}
	if len(paths) != 3 {
		t.Fatalf("expected 3 path records, got %d", len(paths))
	}
}

func TestIndexCommitIdempotent(t *testing.T) {
	repoRoot := setupTestRepo(t)
	storeRoot := filepath.Join(t.TempDir(), ".ctx")
	os.MkdirAll(storeRoot, 0755)

	sha, err := GetHeadSHA(repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	if err := IndexCommit(storeRoot, repoRoot, sha); err != nil {
		t.Fatalf("first IndexCommit: %v", err)
	}
	if err := IndexCommit(storeRoot, repoRoot, sha); err != nil {
		t.Fatalf("second IndexCommit should be idempotent: %v", err)
	}

	// Should still have only 1 commit record (second was skipped)
	commits, err := graph.ReadRecords[graph.CommitRecord](graph.CommitsPath(storeRoot))
	if err != nil {
		t.Fatalf("reading commits: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit record (idempotent), got %d", len(commits))
	}
}

func TestIndexRange(t *testing.T) {
	repoRoot := setupTestRepo(t)
	storeRoot := filepath.Join(t.TempDir(), ".ctx")
	os.MkdirAll(storeRoot, 0755)

	// Get both commit SHAs
	cmd := exec.Command("git", "log", "--format=%H", "--reverse")
	cmd.Dir = repoRoot
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}

	commits := splitLines(string(out))
	if len(commits) < 2 {
		t.Fatal("need at least 2 commits")
	}

	// Index the first commit directly
	if err := IndexCommit(storeRoot, repoRoot, commits[0]); err != nil {
		t.Fatalf("IndexCommit base: %v", err)
	}

	// Index range from first to second
	if err := IndexRange(storeRoot, repoRoot, commits[0], commits[1]); err != nil {
		t.Fatalf("IndexRange: %v", err)
	}

	// Should have both commits indexed
	commitRecs, err := graph.ReadRecords[graph.CommitRecord](graph.CommitsPath(storeRoot))
	if err != nil {
		t.Fatal(err)
	}
	if len(commitRecs) != 2 {
		t.Fatalf("expected 2 commit records, got %d", len(commitRecs))
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"main.go", "go"},
		{"app.ts", "typescript"},
		{"index.tsx", "typescript"},
		{"script.py", "python"},
		{"lib.rs", "rust"},
		{"App.java", "java"},
		{"README.md", "markdown"},
		{"config.yaml", "yaml"},
		{"data.json", "json"},
		{"Makefile", "makefile"},
		{"Dockerfile", "dockerfile"},
		{"unknown.xyz", ""},
	}

	for _, tt := range tests {
		got := detectLanguage(tt.path)
		if got != tt.want {
			t.Errorf("detectLanguage(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestGetRepoRoot(t *testing.T) {
	repoRoot := setupTestRepo(t)

	got, err := GetRepoRoot(repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	// Resolve symlinks for comparison on macOS where /tmp may be a symlink
	expected, _ := filepath.EvalSymlinks(repoRoot)
	actual, _ := filepath.EvalSymlinks(got)
	if actual != expected {
		t.Errorf("GetRepoRoot: got %q, want %q", actual, expected)
	}
}

func splitLines(s string) []string {
	var lines []string
	for _, line := range filepath.SplitList(s) {
		if line != "" {
			lines = append(lines, line)
		}
	}
	// filepath.SplitList splits by OS path separator, not newlines
	// Use strings.Split instead
	lines = nil
	for _, line := range split(s) {
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func split(s string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			result = append(result, line)
			start = i + 1
		}
	}
	if start < len(s) {
		result = append(result, s[start:])
	}
	return result
}
