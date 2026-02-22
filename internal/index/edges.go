package index

import (
	"regexp"
	"sort"
	"strings"

	"github.com/contextsubstrate/ctx/internal/graph"
)

// ExtractImports detects import statements and returns ImportEdge records.
// pathIndex maps file path → pathID for resolving internal imports.
func ExtractImports(content []byte, language, commitSHA, fromPathID string, pathIndex map[string]string) []graph.ImportEdge {
	if len(content) == 0 {
		return nil
	}

	var edges []graph.ImportEdge
	switch language {
	case "go":
		edges = extractGoImports(string(content), commitSHA, fromPathID)
	case "typescript", "javascript":
		edges = extractTSImports(string(content), commitSHA, fromPathID, pathIndex)
	case "python":
		edges = extractPythonImports(string(content), commitSHA, fromPathID)
	}

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].ToExternalModule != edges[j].ToExternalModule {
			return edges[i].ToExternalModule < edges[j].ToExternalModule
		}
		return edges[i].ToPathID < edges[j].ToPathID
	})

	return edges
}

// callerName extracts just the function name from a SymbolRecord.
func callerName(sym graph.SymbolRecord) string {
	// For methods like "Receiver.Method", return "Method"
	parts := strings.Split(sym.Name, ".")
	return parts[len(parts)-1]
}

// StartLine/EndLine helpers that read from associated regions.
// Since SymbolRecord doesn't store line info directly, we use a helper
// that looks up the caller in the symbol list and returns region info.
// For simplicity, we add helper methods via extension on the caller side.

// symbolLineRange returns (startLine, endLine) for a symbol by looking up its
// DefRegionID in the provided regions.
func symbolLineRange(sym graph.SymbolRecord, regions []graph.RegionRecord) (int, int) {
	for _, r := range regions {
		if r.RegionID == sym.DefRegionID {
			return r.StartLine, r.EndLine
		}
	}
	return 0, 0
}

// ExtractCallEdgesWithRegions is the region-aware version of ExtractCallEdges.
func ExtractCallEdgesWithRegions(content []byte, language, commitSHA string, callerSymbols []graph.SymbolRecord, regions []graph.RegionRecord, knownSymbols map[string]string) []graph.CallEdge {
	if len(content) == 0 || len(callerSymbols) == 0 {
		return nil
	}

	lines := strings.Split(string(content), "\n")
	callRe := regexp.MustCompile(`(\w+)\s*\(`)
	var edges []graph.CallEdge

	for _, caller := range callerSymbols {
		startLine, endLine := symbolLineRange(caller, regions)
		if startLine == 0 {
			continue
		}

		for lineIdx := startLine - 1; lineIdx < endLine && lineIdx < len(lines); lineIdx++ {
			matches := callRe.FindAllStringSubmatch(lines[lineIdx], -1)
			for _, m := range matches {
				calledName := m[1]
				if isKeyword(calledName, language) || calledName == callerName(caller) {
					continue
				}

				edge := graph.CallEdge{
					Type:         graph.TypeCallEdge,
					Commit:       commitSHA,
					FromSymbolID: caller.SymbolID,
					CallType:     "direct",
					Confidence:   0.5,
				}

				if targetID, ok := knownSymbols[calledName]; ok {
					edge.ToSymbolID = targetID
					edge.Confidence = 0.8
				} else {
					edge.ToExternalRef = calledName
				}

				edges = append(edges, edge)
			}
		}
	}

	edges = deduplicateCallEdges(edges)

	sort.Slice(edges, func(i, j int) bool {
		if edges[i].FromSymbolID != edges[j].FromSymbolID {
			return edges[i].FromSymbolID < edges[j].FromSymbolID
		}
		if edges[i].ToSymbolID != edges[j].ToSymbolID {
			return edges[i].ToSymbolID < edges[j].ToSymbolID
		}
		return edges[i].ToExternalRef < edges[j].ToExternalRef
	})

	return edges
}

func deduplicateCallEdges(edges []graph.CallEdge) []graph.CallEdge {
	seen := make(map[string]bool)
	var unique []graph.CallEdge
	for _, e := range edges {
		key := e.FromSymbolID + "→" + e.ToSymbolID + "→" + e.ToExternalRef
		if !seen[key] {
			seen[key] = true
			unique = append(unique, e)
		}
	}
	return unique
}

// --- Go import extraction ---

var (
	goSingleImportRe = regexp.MustCompile(`(?m)^import\s+"([^"]+)"`)
	goBlockImportRe  = regexp.MustCompile(`(?ms)^import\s*\(\s*\n(.*?)\n\s*\)`)
	goImportLineRe   = regexp.MustCompile(`\s*(?:\w+\s+)?"([^"]+)"`)
)

func extractGoImports(content, commitSHA, fromPathID string) []graph.ImportEdge {
	var edges []graph.ImportEdge

	// Single import
	for _, m := range goSingleImportRe.FindAllStringSubmatch(content, -1) {
		edges = append(edges, graph.ImportEdge{
			Type:             graph.TypeImportEdge,
			Commit:           commitSHA,
			FromPathID:       fromPathID,
			ToExternalModule: m[1],
		})
	}

	// Block import
	blocks := goBlockImportRe.FindAllStringSubmatch(content, -1)
	for _, block := range blocks {
		for _, line := range strings.Split(block[1], "\n") {
			if m := goImportLineRe.FindStringSubmatch(line); m != nil {
				edges = append(edges, graph.ImportEdge{
					Type:             graph.TypeImportEdge,
					Commit:           commitSHA,
					FromPathID:       fromPathID,
					ToExternalModule: m[1],
				})
			}
		}
	}

	return edges
}

// --- TypeScript/JavaScript import extraction ---

var (
	tsImportRe    = regexp.MustCompile(`(?m)^import\s+.*?\s+from\s+['"]([^'"]+)['"]`)
	tsRequireRe   = regexp.MustCompile(`(?m)require\s*\(\s*['"]([^'"]+)['"]\s*\)`)
	tsDynImportRe = regexp.MustCompile(`(?m)import\s*\(\s*['"]([^'"]+)['"]\s*\)`)
)

func extractTSImports(content, commitSHA, fromPathID string, pathIndex map[string]string) []graph.ImportEdge {
	var edges []graph.ImportEdge
	seen := make(map[string]bool)

	for _, re := range []*regexp.Regexp{tsImportRe, tsRequireRe, tsDynImportRe} {
		for _, m := range re.FindAllStringSubmatch(content, -1) {
			module := m[1]
			if seen[module] {
				continue
			}
			seen[module] = true

			edge := graph.ImportEdge{
				Type:       graph.TypeImportEdge,
				Commit:     commitSHA,
				FromPathID: fromPathID,
			}

			// Try to resolve relative imports to internal path IDs
			if strings.HasPrefix(module, ".") && pathIndex != nil {
				resolved := false
				for path, pid := range pathIndex {
					if matchesRelativeImport(path, module) {
						edge.ToPathID = pid
						resolved = true
						break
					}
				}
				if !resolved {
					edge.ToExternalModule = module
				}
			} else {
				edge.ToExternalModule = module
			}

			edges = append(edges, edge)
		}
	}

	return edges
}

func matchesRelativeImport(filePath, importPath string) bool {
	// Simplified: check if the import path suffix matches the file path
	clean := strings.TrimPrefix(importPath, "./")
	clean = strings.TrimPrefix(clean, "../")

	// Try common extensions
	for _, ext := range []string{"", ".ts", ".tsx", ".js", ".jsx", "/index.ts", "/index.js"} {
		if strings.HasSuffix(filePath, clean+ext) {
			return true
		}
	}
	return false
}

// --- Python import extraction ---

var (
	pyImportRe     = regexp.MustCompile(`(?m)^import\s+(\S+)`)
	pyFromImportRe = regexp.MustCompile(`(?m)^from\s+(\S+)\s+import`)
)

func extractPythonImports(content, commitSHA, fromPathID string) []graph.ImportEdge {
	var edges []graph.ImportEdge
	seen := make(map[string]bool)

	for _, re := range []*regexp.Regexp{pyImportRe, pyFromImportRe} {
		for _, m := range re.FindAllStringSubmatch(content, -1) {
			module := m[1]
			if seen[module] {
				continue
			}
			seen[module] = true

			edges = append(edges, graph.ImportEdge{
				Type:             graph.TypeImportEdge,
				Commit:           commitSHA,
				FromPathID:       fromPathID,
				ToExternalModule: module,
			})
		}
	}

	return edges
}

// --- Keyword filtering ---

var keywords = map[string]map[string]bool{
	"go": {
		"if": true, "else": true, "for": true, "range": true, "return": true,
		"switch": true, "case": true, "break": true, "continue": true, "defer": true,
		"go": true, "select": true, "chan": true, "map": true, "make": true,
		"new": true, "len": true, "cap": true, "append": true, "copy": true,
		"delete": true, "panic": true, "recover": true, "close": true,
		"print": true, "println": true, "string": true, "int": true, "bool": true,
		"byte": true, "error": true, "nil": true, "true": true, "false": true,
	},
	"typescript": {
		"if": true, "else": true, "for": true, "while": true, "return": true,
		"switch": true, "case": true, "break": true, "continue": true, "throw": true,
		"try": true, "catch": true, "finally": true, "new": true, "delete": true,
		"typeof": true, "instanceof": true, "void": true, "this": true, "super": true,
		"class": true, "extends": true, "implements": true, "import": true, "export": true,
		"default": true, "const": true, "let": true, "var": true, "function": true,
		"async": true, "await": true, "yield": true, "from": true, "as": true,
		"true": true, "false": true, "null": true, "undefined": true,
		"console": true, "require": true, "module": true,
	},
	"javascript": nil, // populated below
	"python": {
		"if": true, "else": true, "elif": true, "for": true, "while": true,
		"return": true, "break": true, "continue": true, "pass": true, "raise": true,
		"try": true, "except": true, "finally": true, "with": true, "as": true,
		"import": true, "from": true, "class": true, "def": true, "lambda": true,
		"and": true, "or": true, "not": true, "in": true, "is": true,
		"True": true, "False": true, "None": true, "self": true, "cls": true,
		"print": true, "len": true, "range": true, "type": true, "int": true,
		"str": true, "list": true, "dict": true, "set": true, "tuple": true,
		"isinstance": true, "issubclass": true, "super": true, "property": true,
	},
}

func init() {
	// JavaScript uses same keywords as TypeScript
	keywords["javascript"] = keywords["typescript"]
}

func isKeyword(name, language string) bool {
	if kw, ok := keywords[language]; ok {
		return kw[name]
	}
	return false
}
