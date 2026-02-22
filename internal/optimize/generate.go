package optimize

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/contextsubstrate/ctx/internal/graph"
	"github.com/contextsubstrate/ctx/internal/index"
)

// DefaultTokenCap is the default maximum token estimate for a generated pack.
const DefaultTokenCap = 32000

// Approximate tokens per byte for source code (conservative estimate).
const tokensPerByte = 0.25

// PackRequest specifies what to include in an optimized context pack.
type PackRequest struct {
	Commit       string `json:"commit"`
	Task         string `json:"task"`
	TokenCap     int    `json:"token_cap"`
	IncludeTests bool   `json:"include_tests"`
}

// PackItem represents a single item included in the optimized pack.
type PackItem struct {
	Path            string `json:"path"`
	Language        string `json:"language,omitempty"`
	SymbolID        string `json:"symbol_id,omitempty"`
	SymbolName      string `json:"symbol_name,omitempty"`
	EstimatedTokens int    `json:"estimated_tokens"`
	Reason          string `json:"reason"`
}

// OptimizedPack is the result of pack generation.
type OptimizedPack struct {
	Commit          string     `json:"commit"`
	Task            string     `json:"task"`
	Files           []PackItem `json:"files"`
	Symbols         []PackItem `json:"symbols,omitempty"`
	Snippets        []PackItem `json:"snippets,omitempty"`
	ADRConstraints  []string   `json:"adr_constraints,omitempty"`
	EstimatedTokens int        `json:"estimated_tokens"`
	TokenCap        int        `json:"token_cap"`
}

// JSON returns the optimized pack as formatted JSON.
func (p *OptimizedPack) JSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// Human returns a human-readable summary of the optimized pack.
func (p *OptimizedPack) Human() string {
	var b strings.Builder

	fmt.Fprintf(&b, "Optimized Pack for commit %s\n", shortSHA(p.Commit))
	fmt.Fprintf(&b, "Task: %s\n", p.Task)
	fmt.Fprintf(&b, "───────────────────────────────────\n")
	fmt.Fprintf(&b, "Token budget: %d / %d (%.0f%% used)\n",
		p.EstimatedTokens, p.TokenCap,
		float64(p.EstimatedTokens)/float64(p.TokenCap)*100)

	if len(p.Files) > 0 {
		fmt.Fprintf(&b, "\nFiles (%d):\n", len(p.Files))
		for _, f := range p.Files {
			fmt.Fprintf(&b, "  %-40s ~%5d tokens  [%s]\n", f.Path, f.EstimatedTokens, f.Reason)
		}
	}

	if len(p.Symbols) > 0 {
		fmt.Fprintf(&b, "\nSymbols (%d):\n", len(p.Symbols))
		for _, s := range p.Symbols {
			fmt.Fprintf(&b, "  %-40s ~%5d tokens  [%s]\n", s.SymbolName, s.EstimatedTokens, s.Reason)
		}
	}

	return b.String()
}

// GeneratePack builds an optimized context pack from the indexed graph.
// It selects files and symbols relevant to the task within the token budget.
func GeneratePack(storeRoot, repoRoot string, req *PackRequest) (*OptimizedPack, error) {
	if req.TokenCap <= 0 {
		req.TokenCap = DefaultTokenCap
	}

	commitSHA := req.Commit
	if commitSHA == "" {
		var err error
		commitSHA, err = index.GetHeadSHA(repoRoot)
		if err != nil {
			return nil, fmt.Errorf("getting HEAD: %w", err)
		}
	}

	// Read file snapshots for this commit
	files, err := graph.ReadRecords[graph.FileSnapshot](graph.FilesPath(storeRoot, commitSHA))
	if err != nil {
		return nil, fmt.Errorf("reading files for %s: %w", commitSHA[:8], err)
	}

	// Read path records to resolve PathID → path
	paths, err := graph.ReadRecords[graph.PathRecord](graph.PathsPath(storeRoot))
	if err != nil {
		return nil, fmt.Errorf("reading paths: %w", err)
	}
	pathLookup := make(map[string]string, len(paths))
	for _, p := range paths {
		pathLookup[p.PathID] = p.Path
	}

	// Read symbols if available
	symbols, _ := graph.ReadRecords[graph.SymbolRecord](graph.SymbolsPath(storeRoot, commitSHA))

	// Score and rank files
	taskLower := strings.ToLower(req.Task)
	taskWords := extractTaskWords(taskLower)

	type scoredFile struct {
		snapshot graph.FileSnapshot
		path     string
		score    float64
		tokens   int
		reason   string
	}

	var candidates []scoredFile
	for _, f := range files {
		path := pathLookup[f.PathID]
		if path == "" {
			continue
		}

		// Skip binary and generated files
		if f.IsBinary || f.IsGenerated {
			continue
		}

		// Skip test files unless requested
		if !req.IncludeTests && isTestFile(path) {
			continue
		}

		tokens := estimateTokens(f.ByteSize)
		score := scoreFile(path, f.Language, taskWords)

		reason := "relevant"
		if score >= 2.0 {
			reason = "high-relevance"
		} else if score >= 1.0 {
			reason = "medium-relevance"
		} else {
			reason = "low-relevance"
		}

		candidates = append(candidates, scoredFile{
			snapshot: f,
			path:     path,
			score:    score,
			tokens:   tokens,
			reason:   reason,
		})
	}

	// Sort by score descending
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}
		return candidates[i].path < candidates[j].path
	})

	// Fill pack within token budget
	pack := &OptimizedPack{
		Commit:   commitSHA,
		Task:     req.Task,
		TokenCap: req.TokenCap,
	}

	remainingTokens := req.TokenCap
	for _, c := range candidates {
		if c.tokens > remainingTokens {
			// Try to include if it's high-scoring and we still have most of the budget
			if c.score < 2.0 || remainingTokens < req.TokenCap/4 {
				continue
			}
		}

		pack.Files = append(pack.Files, PackItem{
			Path:            c.path,
			Language:        c.snapshot.Language,
			EstimatedTokens: c.tokens,
			Reason:          c.reason,
		})
		remainingTokens -= c.tokens

		if remainingTokens <= 0 {
			break
		}
	}

	// Add relevant symbols
	if len(symbols) > 0 {
		type scoredSymbol struct {
			symbol graph.SymbolRecord
			score  float64
			tokens int
		}

		// Build set of included file PathIDs
		includedPaths := make(map[string]bool)
		for _, f := range pack.Files {
			for _, snap := range files {
				if pathLookup[snap.PathID] == f.Path {
					includedPaths[snap.PathID] = true
					break
				}
			}
		}

		var symCandidates []scoredSymbol
		for _, sym := range symbols {
			if !includedPaths[sym.PathID] {
				continue
			}

			score := scoreSymbol(sym, taskWords)
			tokens := estimateTokens(len(sym.Signature) + len(sym.Docstring))
			if tokens < 10 {
				tokens = 10 // minimum
			}

			symCandidates = append(symCandidates, scoredSymbol{
				symbol: sym,
				score:  score,
				tokens: tokens,
			})
		}

		sort.Slice(symCandidates, func(i, j int) bool {
			return symCandidates[i].score > symCandidates[j].score
		})

		for _, sc := range symCandidates {
			if sc.tokens > remainingTokens {
				continue
			}

			pack.Symbols = append(pack.Symbols, PackItem{
				Path:            pathLookup[sc.symbol.PathID],
				SymbolID:        sc.symbol.SymbolID,
				SymbolName:      sc.symbol.FQName,
				EstimatedTokens: sc.tokens,
				Reason:          fmt.Sprintf("task-relevant-%s", sc.symbol.Kind),
			})
			remainingTokens -= sc.tokens

			if remainingTokens <= 0 {
				break
			}
		}
	}

	// Compute total
	total := 0
	for _, f := range pack.Files {
		total += f.EstimatedTokens
	}
	for _, s := range pack.Symbols {
		total += s.EstimatedTokens
	}
	pack.EstimatedTokens = total

	return pack, nil
}

// estimateTokens provides a rough token count from byte size.
func estimateTokens(byteSize int) int {
	tokens := int(float64(byteSize) * tokensPerByte)
	if tokens < 1 {
		tokens = 1
	}
	return tokens
}

// scoreFile assigns a relevance score to a file based on task keywords.
func scoreFile(path, language string, taskWords []string) float64 {
	score := 0.0
	pathLower := strings.ToLower(path)

	// Boost source code files
	switch language {
	case "go", "typescript", "javascript", "python", "rust", "java":
		score += 0.5
	}

	// Boost files whose path matches task words
	for _, word := range taskWords {
		if strings.Contains(pathLower, word) {
			score += 2.0
		}
	}

	// Boost entry points
	base := strings.ToLower(path)
	if strings.Contains(base, "main.") || strings.Contains(base, "index.") || strings.Contains(base, "app.") {
		score += 0.5
	}

	// Penalty for deep nesting
	depth := strings.Count(path, "/")
	if depth > 3 {
		score -= float64(depth-3) * 0.1
	}

	return score
}

// scoreSymbol assigns a relevance score to a symbol based on task keywords.
func scoreSymbol(sym graph.SymbolRecord, taskWords []string) float64 {
	score := 0.0
	nameLower := strings.ToLower(sym.Name)
	fqLower := strings.ToLower(sym.FQName)

	// Boost exported symbols
	if sym.Visibility == "exported" {
		score += 1.0
	}

	// Boost functions and methods
	if sym.Kind == "function" || sym.Kind == "method" {
		score += 0.5
	}

	// Match task words
	for _, word := range taskWords {
		if strings.Contains(nameLower, word) || strings.Contains(fqLower, word) {
			score += 2.0
		}
	}

	return score
}

// extractTaskWords splits a task description into searchable words.
func extractTaskWords(task string) []string {
	// Split on common delimiters
	words := strings.FieldsFunc(task, func(r rune) bool {
		return r == ' ' || r == ',' || r == '.' || r == ';' || r == ':' || r == '-' || r == '_' || r == '/' || r == '\'' || r == '"'
	})

	// Filter short/common words
	var result []string
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true,
		"to": true, "in": true, "of": true, "for": true, "is": true,
		"it": true, "on": true, "at": true, "by": true, "with": true,
		"from": true, "this": true, "that": true, "be": true, "as": true,
		"add": true, "fix": true, "update": true, "implement": true,
	}

	for _, w := range words {
		w = strings.ToLower(w)
		if len(w) >= 3 && !stopWords[w] {
			result = append(result, w)
		}
	}

	return result
}

func isTestFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.Contains(lower, "_test.") ||
		strings.Contains(lower, ".test.") ||
		strings.Contains(lower, ".spec.") ||
		strings.Contains(lower, "__tests__/") ||
		strings.Contains(lower, "test/") ||
		strings.Contains(lower, "tests/")
}

func shortSHA(sha string) string {
	if len(sha) > 8 {
		return sha[:8]
	}
	return sha
}
