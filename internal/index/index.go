package index

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/contextsubstrate/ctx/internal/graph"
)

// IndexCommit indexes a single commit, creating file snapshots, path records,
// and a commit record in the graph store.
func IndexCommit(storeRoot, repoRoot, commitSHA string) error {
	// Ensure graph directories exist
	if err := graph.InitGraph(storeRoot); err != nil {
		return fmt.Errorf("initializing graph: %w", err)
	}

	// Check if already indexed
	snapDir := graph.SnapshotDir(storeRoot, commitSHA)
	if _, err := os.Stat(graph.FilesPath(storeRoot, commitSHA)); err == nil {
		return nil // already indexed
	}

	// Get commit info
	info, err := GetCommitInfo(repoRoot, commitSHA)
	if err != nil {
		return fmt.Errorf("getting commit info: %w", err)
	}

	// Record commit
	authoredAt, _ := time.Parse(time.RFC3339, info.Timestamp)
	commitRec := graph.CommitRecord{
		Type:       graph.TypeCommit,
		Repo:       repoRoot,
		SHA:        info.SHA,
		ParentSHA:  info.ParentSHA,
		Author:     info.Author,
		Message:    info.Message,
		AuthoredAt: authoredAt,
	}
	if err := graph.AppendRecord(graph.CommitsPath(storeRoot), commitRec); err != nil {
		return fmt.Errorf("recording commit: %w", err)
	}

	// List all files at this commit
	files, err := ListFilesAtCommit(repoRoot, commitSHA)
	if err != nil {
		return fmt.Errorf("listing files: %w", err)
	}

	// Load existing paths for update
	existingPaths, err := graph.ReadRecords[graph.PathRecord](graph.PathsPath(storeRoot))
	if err != nil {
		return fmt.Errorf("reading paths: %w", err)
	}
	pathIndex := make(map[string]*graph.PathRecord, len(existingPaths))
	for i := range existingPaths {
		pathIndex[existingPaths[i].Path] = &existingPaths[i]
	}

	// Create snapshot directory
	if err := os.MkdirAll(snapDir, 0755); err != nil {
		return fmt.Errorf("creating snapshot dir: %w", err)
	}

	// Build path→pathID map for edge resolution
	filePathToID := make(map[string]string)
	for _, fpath := range files {
		filePathToID[fpath] = pathIDFromPath(fpath)
	}

	// Index each file
	var fileSnapshots []any
	var allSymbols []graph.SymbolRecord
	var allRegions []graph.RegionRecord
	var allImportEdges []graph.ImportEdge
	var newPaths []graph.PathRecord

	// Track file contents for symbol/edge extraction pass
	type fileData struct {
		path    string
		pathID  string
		lang    string
		content []byte
	}
	var sourceFiles []fileData

	for _, fpath := range files {
		pathID := filePathToID[fpath]

		// Update or create path record
		if existing, ok := pathIndex[fpath]; ok {
			existing.LastSeenCommit = &commitSHA
		} else {
			pr := graph.PathRecord{
				Type:            graph.TypePath,
				PathID:          pathID,
				Repo:            repoRoot,
				Path:            fpath,
				FirstSeenCommit: commitSHA,
			}
			newPaths = append(newPaths, pr)
			pathIndex[fpath] = &pr
		}

		// Get file content at this commit
		content, err := getFileContentAtCommit(repoRoot, commitSHA, fpath)
		if err != nil {
			// Skip files we can't read (e.g., submodules)
			continue
		}

		isBinary := isBinaryContent(content)
		lang := detectLanguage(fpath)
		loc := 0
		if !isBinary {
			loc = countLines(content)
		}

		h := sha256.Sum256(content)
		contentHash := hex.EncodeToString(h[:])

		blobOID, err := getGitBlobOID(repoRoot, commitSHA, fpath)
		if err != nil {
			blobOID = ""
		}

		snap := graph.FileSnapshot{
			Type:          graph.TypeFileSnapshot,
			Commit:        commitSHA,
			PathID:        pathID,
			BlobOID:       blobOID,
			ContentSHA256: contentHash,
			Language:      lang,
			ByteSize:      len(content),
			LOC:           loc,
			IsGenerated:   isGeneratedFile(fpath),
			IsBinary:      isBinary,
		}
		fileSnapshots = append(fileSnapshots, snap)

		// Collect source files for symbol/edge extraction
		if !isBinary && lang != "" {
			sourceFiles = append(sourceFiles, fileData{
				path:    fpath,
				pathID:  pathID,
				lang:    lang,
				content: content,
			})
		}
	}

	// Sort file snapshots by PathID for deterministic output
	sort.Slice(fileSnapshots, func(i, j int) bool {
		a := fileSnapshots[i].(graph.FileSnapshot)
		b := fileSnapshots[j].(graph.FileSnapshot)
		return a.PathID < b.PathID
	})

	// Write file snapshots
	if err := graph.WriteRecords(graph.FilesPath(storeRoot, commitSHA), fileSnapshots); err != nil {
		return fmt.Errorf("writing file snapshots: %w", err)
	}

	// Extract symbols, regions, and imports from source files
	for _, sf := range sourceFiles {
		syms, regs := ExtractSymbols(sf.content, sf.lang, commitSHA, sf.pathID)
		allSymbols = append(allSymbols, syms...)
		allRegions = append(allRegions, regs...)

		imports := ExtractImports(sf.content, sf.lang, commitSHA, sf.pathID, filePathToID)
		allImportEdges = append(allImportEdges, imports...)
	}

	// Extract call edges using all known symbols
	knownSymbols := make(map[string]string, len(allSymbols))
	for _, s := range allSymbols {
		knownSymbols[s.Name] = s.SymbolID
		// Also index by short name for methods (e.g., "Method" for "Receiver.Method")
		parts := strings.Split(s.Name, ".")
		if len(parts) > 1 {
			knownSymbols[parts[len(parts)-1]] = s.SymbolID
		}
	}

	var allCallEdges []graph.CallEdge
	for _, sf := range sourceFiles {
		// Get symbols for this file
		var fileSymbols []graph.SymbolRecord
		var fileRegions []graph.RegionRecord
		for _, s := range allSymbols {
			if s.PathID == sf.pathID {
				fileSymbols = append(fileSymbols, s)
			}
		}
		for _, r := range allRegions {
			if r.PathID == sf.pathID {
				fileRegions = append(fileRegions, r)
			}
		}

		calls := ExtractCallEdgesWithRegions(sf.content, sf.lang, commitSHA, fileSymbols, fileRegions, knownSymbols)
		allCallEdges = append(allCallEdges, calls...)
	}

	// Write symbol records
	if len(allSymbols) > 0 {
		symRecords := make([]any, len(allSymbols))
		for i := range allSymbols {
			symRecords[i] = allSymbols[i]
		}
		if err := graph.WriteRecords(graph.SymbolsPath(storeRoot, commitSHA), symRecords); err != nil {
			return fmt.Errorf("writing symbols: %w", err)
		}
	}

	// Write region records
	if len(allRegions) > 0 {
		regRecords := make([]any, len(allRegions))
		for i := range allRegions {
			regRecords[i] = allRegions[i]
		}
		if err := graph.WriteRecords(graph.RegionsPath(storeRoot, commitSHA), regRecords); err != nil {
			return fmt.Errorf("writing regions: %w", err)
		}
	}

	// Write import edges
	if len(allImportEdges) > 0 {
		impRecords := make([]any, len(allImportEdges))
		for i := range allImportEdges {
			impRecords[i] = allImportEdges[i]
		}
		if err := graph.WriteRecords(graph.ImportEdgesPath(storeRoot, commitSHA), impRecords); err != nil {
			return fmt.Errorf("writing import edges: %w", err)
		}
	}

	// Write call edges
	if len(allCallEdges) > 0 {
		callRecords := make([]any, len(allCallEdges))
		for i := range allCallEdges {
			callRecords[i] = allCallEdges[i]
		}
		if err := graph.WriteRecords(graph.CallEdgesPath(storeRoot, commitSHA), callRecords); err != nil {
			return fmt.Errorf("writing call edges: %w", err)
		}
	}

	// Append new path records
	for _, pr := range newPaths {
		if err := graph.AppendRecord(graph.PathsPath(storeRoot), pr); err != nil {
			return fmt.Errorf("recording path: %w", err)
		}
	}

	return nil
}

// IndexRange indexes all commits between baseSHA and headSHA incrementally.
func IndexRange(storeRoot, repoRoot, baseSHA, headSHA string) error {
	// Get list of commits from base to head
	commits, err := listCommitRange(repoRoot, baseSHA, headSHA)
	if err != nil {
		return fmt.Errorf("listing commit range: %w", err)
	}

	for _, sha := range commits {
		if err := IndexCommit(storeRoot, repoRoot, sha); err != nil {
			return fmt.Errorf("indexing commit %s: %w", sha[:8], err)
		}
	}

	return nil
}

// listCommitRange returns commits from baseSHA (exclusive) to headSHA (inclusive).
func listCommitRange(repoRoot, baseSHA, headSHA string) ([]string, error) {
	cmd := exec.Command("git", "rev-list", "--reverse", baseSHA+".."+headSHA)
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git rev-list %s..%s: %s: %w", baseSHA, headSHA, stderr.String(), err)
	}

	var commits []string
	for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			commits = append(commits, line)
		}
	}

	return commits, nil
}

func getFileContentAtCommit(repoRoot, commitSHA, filePath string) ([]byte, error) {
	cmd := exec.Command("git", "show", commitSHA+":"+filePath)
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git show %s:%s: %s: %w", commitSHA, filePath, stderr.String(), err)
	}

	return stdout.Bytes(), nil
}

func getGitBlobOID(repoRoot, commitSHA, filePath string) (string, error) {
	cmd := exec.Command("git", "rev-parse", commitSHA+":"+filePath)
	cmd.Dir = repoRoot

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git rev-parse %s:%s: %s: %w", commitSHA, filePath, stderr.String(), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func pathIDFromPath(p string) string {
	h := sha256.Sum256([]byte(p))
	return hex.EncodeToString(h[:16]) // 128-bit path ID
}

// detectLanguage returns a language identifier based on file extension.
func detectLanguage(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".js", ".jsx", ".mjs", ".cjs":
		return "javascript"
	case ".py":
		return "python"
	case ".rs":
		return "rust"
	case ".java":
		return "java"
	case ".rb":
		return "ruby"
	case ".c", ".h":
		return "c"
	case ".cpp", ".cc", ".cxx", ".hpp":
		return "cpp"
	case ".cs":
		return "csharp"
	case ".swift":
		return "swift"
	case ".kt", ".kts":
		return "kotlin"
	case ".md":
		return "markdown"
	case ".yaml", ".yml":
		return "yaml"
	case ".json":
		return "json"
	case ".toml":
		return "toml"
	case ".xml":
		return "xml"
	case ".html", ".htm":
		return "html"
	case ".css":
		return "css"
	case ".scss", ".sass":
		return "scss"
	case ".sql":
		return "sql"
	case ".sh", ".bash":
		return "shell"
	case ".dockerfile":
		return "dockerfile"
	case ".proto":
		return "protobuf"
	default:
		// Check filename-based detection
		base := strings.ToLower(filepath.Base(path))
		switch {
		case base == "dockerfile":
			return "dockerfile"
		case base == "makefile":
			return "makefile"
		case base == "go.mod":
			return "gomod"
		case base == "go.sum":
			return "gosum"
		case strings.HasSuffix(base, ".mod"):
			return "gomod"
		}
		return ""
	}
}

func isBinaryContent(data []byte) bool {
	// Check for null bytes in the first 8KB
	limit := 8192
	if len(data) < limit {
		limit = len(data)
	}
	for i := 0; i < limit; i++ {
		if data[i] == 0 {
			return true
		}
	}
	// Also check if the content is valid UTF-8
	if !utf8.Valid(data[:limit]) {
		return true
	}
	return false
}

func countLines(data []byte) int {
	if len(data) == 0 {
		return 0
	}
	n := bytes.Count(data, []byte("\n"))
	// If file doesn't end with newline, count the last line
	if data[len(data)-1] != '\n' {
		n++
	}
	return n
}

func isGeneratedFile(path string) bool {
	lower := strings.ToLower(path)
	patterns := []string{
		"generated", "vendor/", "node_modules/",
		".min.js", ".min.css", "go.sum",
		"package-lock.json", "yarn.lock", "pnpm-lock.yaml",
	}
	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}
	return false
}
