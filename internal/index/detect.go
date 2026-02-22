package index

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// ChangeSet describes the files that changed between two commits.
type ChangeSet struct {
	BaseSHA      string
	HeadSHA      string
	FilesChanged []string // Modified files
	FilesAdded   []string // New files
	FilesDeleted []string // Removed files
}

// AllFiles returns all files that were affected (changed, added, or deleted).
func (cs *ChangeSet) AllFiles() []string {
	all := make([]string, 0, len(cs.FilesChanged)+len(cs.FilesAdded)+len(cs.FilesDeleted))
	all = append(all, cs.FilesChanged...)
	all = append(all, cs.FilesAdded...)
	all = append(all, cs.FilesDeleted...)
	return all
}

// IsEmpty returns true if no files were affected.
func (cs *ChangeSet) IsEmpty() bool {
	return len(cs.FilesChanged) == 0 && len(cs.FilesAdded) == 0 && len(cs.FilesDeleted) == 0
}

// DetectChanges uses git diff-tree to detect file changes between two commits.
// repoRoot must be the root of the git repository.
func DetectChanges(repoRoot, baseSHA, headSHA string) (*ChangeSet, error) {
	cmd := exec.Command("git", "diff-tree", "-r", "--no-commit-id", "--name-status", baseSHA, headSHA)
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git diff-tree %s..%s: %s: %w", baseSHA, headSHA, stderr.String(), err)
	}

	cs := &ChangeSet{
		BaseSHA: baseSHA,
		HeadSHA: headSHA,
	}

	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}

		status := parts[0]
		path := parts[1]

		switch {
		case status == "A":
			cs.FilesAdded = append(cs.FilesAdded, path)
		case status == "D":
			cs.FilesDeleted = append(cs.FilesDeleted, path)
		case strings.HasPrefix(status, "M"), strings.HasPrefix(status, "T"):
			cs.FilesChanged = append(cs.FilesChanged, path)
		case strings.HasPrefix(status, "R"):
			// Rename: treat as delete old + add new
			renameParts := strings.SplitN(path, "\t", 2)
			if len(renameParts) == 2 {
				cs.FilesDeleted = append(cs.FilesDeleted, renameParts[0])
				cs.FilesAdded = append(cs.FilesAdded, renameParts[1])
			}
		case strings.HasPrefix(status, "C"):
			// Copy: treat as add
			copyParts := strings.SplitN(path, "\t", 2)
			if len(copyParts) == 2 {
				cs.FilesAdded = append(cs.FilesAdded, copyParts[1])
			}
		}
	}

	return cs, scanner.Err()
}

// ListFilesAtCommit returns all tracked files at a given commit.
func ListFilesAtCommit(repoRoot, commitSHA string) ([]string, error) {
	cmd := exec.Command("git", "ls-tree", "-r", "--name-only", commitSHA)
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git ls-tree %s: %s: %w", commitSHA, stderr.String(), err)
	}

	var files []string
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			files = append(files, line)
		}
	}

	return files, scanner.Err()
}

// GetCommitInfo retrieves basic metadata for a commit.
type CommitInfo struct {
	SHA       string
	ParentSHA string
	Author    string
	Message   string
	Timestamp string // ISO 8601
}

func GetCommitInfo(repoRoot, commitSHA string) (*CommitInfo, error) {
	// Format: SHA\nParentSHA\nAuthor\nTimestamp\nMessage
	cmd := exec.Command("git", "log", "-1", "--format=%H%n%P%n%an <%ae>%n%aI%n%s", commitSHA)
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git log %s: %s: %w", commitSHA, stderr.String(), err)
	}

	lines := strings.SplitN(stdout.String(), "\n", 5)
	if len(lines) < 5 {
		return nil, fmt.Errorf("unexpected git log output for %s", commitSHA)
	}

	// ParentSHA may contain multiple parents (merge commit); take first
	parentSHA := strings.SplitN(strings.TrimSpace(lines[1]), " ", 2)[0]

	return &CommitInfo{
		SHA:       strings.TrimSpace(lines[0]),
		ParentSHA: parentSHA,
		Author:    strings.TrimSpace(lines[2]),
		Timestamp: strings.TrimSpace(lines[3]),
		Message:   strings.TrimSpace(lines[4]),
	}, nil
}

// GetHeadSHA returns the full SHA of HEAD.
func GetHeadSHA(repoRoot string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git rev-parse HEAD: %s: %w", stderr.String(), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetRepoRoot returns the root of the git repository containing dir.
func GetRepoRoot(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("not a git repository: %s: %w", stderr.String(), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}
