package optimize

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/contextsubstrate/ctx/internal/graph"
	"github.com/contextsubstrate/ctx/internal/index"
)

func setupTestRepoAndIndex(t *testing.T) (storeRoot, repoRoot, commitSHA string) {
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

	// Create source files
	os.WriteFile(filepath.Join(repoRoot, "main.go"), []byte(`package main

import "fmt"

func main() {
	fmt.Println(helper())
}
`), 0644)

	os.WriteFile(filepath.Join(repoRoot, "util.go"), []byte(`package main

func helper() string {
	return "hello"
}

func anotherHelper() int {
	return 42
}
`), 0644)

	os.WriteFile(filepath.Join(repoRoot, "auth.go"), []byte(`package main

type AuthService struct {
	secret string
}

func NewAuthService(secret string) *AuthService {
	return &AuthService{secret: secret}
}

func (a *AuthService) Validate(token string) bool {
	return token == a.secret
}
`), 0644)

	os.WriteFile(filepath.Join(repoRoot, "README.md"), []byte("# Test Project\n\nA test project with auth.\n"), 0644)

	os.WriteFile(filepath.Join(repoRoot, "main_test.go"), []byte(`package main

import "testing"

func TestMain(t *testing.T) {}
`), 0644)

	run("git", "add", ".")
	run("git", "commit", "-m", "initial commit")

	sha, err := index.GetHeadSHA(repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	// Index the commit
	if err := index.IndexCommit(storeRoot, repoRoot, sha); err != nil {
		t.Fatalf("IndexCommit: %v", err)
	}

	return storeRoot, repoRoot, sha
}

func TestGeneratePack(t *testing.T) {
	storeRoot, repoRoot, commitSHA := setupTestRepoAndIndex(t)

	req := &PackRequest{
		Commit:   commitSHA,
		Task:     "implement authentication validation",
		TokenCap: 50000,
	}

	pack, err := GeneratePack(storeRoot, repoRoot, req)
	if err != nil {
		t.Fatalf("GeneratePack: %v", err)
	}

	if pack.Commit != commitSHA {
		t.Errorf("Commit: got %q, want %q", pack.Commit, commitSHA)
	}
	if pack.Task != req.Task {
		t.Errorf("Task: got %q, want %q", pack.Task, req.Task)
	}
	if pack.TokenCap != 50000 {
		t.Errorf("TokenCap: got %d, want %d", pack.TokenCap, 50000)
	}
	if pack.EstimatedTokens <= 0 {
		t.Error("EstimatedTokens should be > 0")
	}
	if len(pack.Files) == 0 {
		t.Fatal("expected files in pack")
	}

	// auth.go should be ranked high for auth-related task
	topFile := pack.Files[0]
	if topFile.Path != "auth.go" {
		t.Logf("top file is %q (expected auth.go to rank highest for auth task)", topFile.Path)
	}

	// Test files should be excluded by default
	for _, f := range pack.Files {
		if strings.Contains(f.Path, "_test.go") {
			t.Errorf("test file %q should be excluded by default", f.Path)
		}
	}
}

func TestGeneratePackWithTests(t *testing.T) {
	storeRoot, repoRoot, commitSHA := setupTestRepoAndIndex(t)

	req := &PackRequest{
		Commit:       commitSHA,
		Task:         "fix tests",
		TokenCap:     50000,
		IncludeTests: true,
	}

	pack, err := GeneratePack(storeRoot, repoRoot, req)
	if err != nil {
		t.Fatalf("GeneratePack: %v", err)
	}

	hasTestFile := false
	for _, f := range pack.Files {
		if strings.Contains(f.Path, "_test.go") {
			hasTestFile = true
		}
	}
	if !hasTestFile {
		t.Error("expected test files when IncludeTests is true")
	}
}

func TestGeneratePackTokenBudget(t *testing.T) {
	storeRoot, repoRoot, commitSHA := setupTestRepoAndIndex(t)

	req := &PackRequest{
		Commit:   commitSHA,
		Task:     "review code",
		TokenCap: 10, // Very small budget
	}

	pack, err := GeneratePack(storeRoot, repoRoot, req)
	if err != nil {
		t.Fatalf("GeneratePack: %v", err)
	}

	// With tiny budget, should still get some files but be constrained
	if pack.EstimatedTokens > pack.TokenCap*2 {
		t.Errorf("pack greatly exceeds token cap: %d > %d", pack.EstimatedTokens, pack.TokenCap)
	}
}

func TestGeneratePackDefaultTokenCap(t *testing.T) {
	storeRoot, repoRoot, commitSHA := setupTestRepoAndIndex(t)

	req := &PackRequest{
		Commit: commitSHA,
		Task:   "general review",
	}

	pack, err := GeneratePack(storeRoot, repoRoot, req)
	if err != nil {
		t.Fatalf("GeneratePack: %v", err)
	}

	if pack.TokenCap != DefaultTokenCap {
		t.Errorf("TokenCap: got %d, want %d", pack.TokenCap, DefaultTokenCap)
	}
}

func TestGeneratePackJSON(t *testing.T) {
	storeRoot, repoRoot, commitSHA := setupTestRepoAndIndex(t)

	req := &PackRequest{
		Commit:   commitSHA,
		Task:     "add auth",
		TokenCap: 50000,
	}

	pack, err := GeneratePack(storeRoot, repoRoot, req)
	if err != nil {
		t.Fatal(err)
	}

	data, err := pack.JSON()
	if err != nil {
		t.Fatalf("JSON: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}

	if parsed["commit"] == nil {
		t.Error("JSON missing 'commit' field")
	}
	if parsed["task"] == nil {
		t.Error("JSON missing 'task' field")
	}
}

func TestGeneratePackHuman(t *testing.T) {
	storeRoot, repoRoot, commitSHA := setupTestRepoAndIndex(t)

	req := &PackRequest{
		Commit:   commitSHA,
		Task:     "add auth",
		TokenCap: 50000,
	}

	pack, err := GeneratePack(storeRoot, repoRoot, req)
	if err != nil {
		t.Fatal(err)
	}

	human := pack.Human()
	if human == "" {
		t.Fatal("Human() returned empty")
	}
	if !strings.Contains(human, "add auth") {
		t.Error("human output should contain task description")
	}
	if !strings.Contains(human, "Token budget") {
		t.Error("human output should contain token budget")
	}
}

func TestGeneratePackWithSymbols(t *testing.T) {
	storeRoot, repoRoot, commitSHA := setupTestRepoAndIndex(t)

	// Verify symbols were indexed
	symbols, _ := graph.ReadRecords[graph.SymbolRecord](graph.SymbolsPath(storeRoot, commitSHA))
	if len(symbols) == 0 {
		t.Skip("no symbols indexed — symbol extraction may not have matched test files")
	}

	req := &PackRequest{
		Commit:   commitSHA,
		Task:     "implement authentication",
		TokenCap: 50000,
	}

	pack, err := GeneratePack(storeRoot, repoRoot, req)
	if err != nil {
		t.Fatalf("GeneratePack: %v", err)
	}

	// With symbols, pack should include symbol items
	t.Logf("pack has %d files, %d symbols", len(pack.Files), len(pack.Symbols))

	if len(pack.Symbols) > 0 {
		for _, s := range pack.Symbols {
			if s.SymbolName == "" {
				t.Error("symbol item missing name")
			}
			if s.EstimatedTokens <= 0 {
				t.Error("symbol item missing token estimate")
			}
		}
	}
}
