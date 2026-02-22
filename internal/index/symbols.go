package index

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/contextsubstrate/ctx/internal/graph"
)

// SymbolKind enumerates recognized symbol types.
const (
	SymbolFunction  = "function"
	SymbolMethod    = "method"
	SymbolType      = "type"
	SymbolInterface = "interface"
	SymbolClass     = "class"
	SymbolConstant  = "constant"
	SymbolVariable  = "variable"
)

// VisibilityExported marks an exported/public symbol.
const (
	VisibilityExported = "exported"
	VisibilityPrivate  = "private"
)

// rawSymbol is an intermediate representation before building graph records.
type rawSymbol struct {
	Kind       string
	Name       string
	Signature  string
	Docstring  string
	Visibility string
	StartLine  int
	EndLine    int
}

// ExtractSymbols extracts symbol definitions from file content based on language.
// Returns symbol records and region records for the given file.
func ExtractSymbols(content []byte, language, commitSHA, pathID string) ([]graph.SymbolRecord, []graph.RegionRecord) {
	if len(content) == 0 {
		return nil, nil
	}

	var raws []rawSymbol
	switch language {
	case "go":
		raws = extractGoSymbols(string(content))
	case "typescript", "javascript":
		raws = extractTSSymbols(string(content))
	case "python":
		raws = extractPythonSymbols(string(content))
	default:
		return nil, nil
	}

	var symbols []graph.SymbolRecord
	var regions []graph.RegionRecord

	for _, raw := range raws {
		fqName := raw.Name
		symbolID := makeSymbolID(pathID, raw.Name, raw.Kind)
		regionID := makeRegionID(pathID, raw.StartLine, raw.EndLine)

		symHash := hashString(raw.Signature + raw.Name)

		region := graph.RegionRecord{
			Type:       graph.TypeRegion,
			Commit:     commitSHA,
			RegionID:   regionID,
			PathID:     pathID,
			RegionHash: hashString(fmt.Sprintf("%d:%d", raw.StartLine, raw.EndLine)),
			Purpose:    "definition",
			StartLine:  raw.StartLine,
			StartCol:   0,
			EndLine:    raw.EndLine,
			EndCol:     0,
		}
		regions = append(regions, region)

		sym := graph.SymbolRecord{
			Type:        graph.TypeSymbol,
			Commit:      commitSHA,
			SymbolID:    symbolID,
			PathID:      pathID,
			Kind:        raw.Kind,
			Name:        raw.Name,
			FQName:      fqName,
			Visibility:  raw.Visibility,
			Language:    language,
			Signature:   raw.Signature,
			Docstring:   raw.Docstring,
			SymbolHash:  symHash,
			DefRegionID: regionID,
		}
		symbols = append(symbols, sym)
	}

	// Sort for deterministic output
	sort.Slice(symbols, func(i, j int) bool {
		return symbols[i].SymbolID < symbols[j].SymbolID
	})
	sort.Slice(regions, func(i, j int) bool {
		return regions[i].RegionID < regions[j].RegionID
	})

	return symbols, regions
}

// --- Go symbol extraction ---

var (
	goFuncRe      = regexp.MustCompile(`(?m)^func\s+(\w+)\s*\(([^)]*)\)\s*(.*)`)
	goMethodRe    = regexp.MustCompile(`(?m)^func\s+\(\s*\w+\s+\*?(\w+)\s*\)\s+(\w+)\s*\(([^)]*)\)\s*(.*)`)
	goTypeRe      = regexp.MustCompile(`(?m)^type\s+(\w+)\s+(struct|interface)\s*\{`)
	goConstVarRe  = regexp.MustCompile(`(?m)^(const|var)\s+(\w+)\s`)
)

func extractGoSymbols(content string) []rawSymbol {
	lines := strings.Split(content, "\n")
	var results []rawSymbol

	for i, line := range lines {
		lineNum := i + 1

		// Methods (must check before functions)
		if m := goMethodRe.FindStringSubmatch(line); m != nil {
			receiver := m[1]
			name := m[2]
			params := m[3]
			ret := strings.TrimSpace(m[4])
			sig := fmt.Sprintf("func (%s) %s(%s) %s", receiver, name, params, ret)
			results = append(results, rawSymbol{
				Kind:       SymbolMethod,
				Name:       receiver + "." + name,
				Signature:  strings.TrimSpace(sig),
				Visibility: goVisibility(name),
				StartLine:  lineNum,
				EndLine:    findGoBlockEnd(lines, i),
			})
			continue
		}

		// Functions
		if m := goFuncRe.FindStringSubmatch(line); m != nil {
			name := m[1]
			params := m[2]
			ret := strings.TrimSpace(m[3])
			sig := fmt.Sprintf("func %s(%s) %s", name, params, ret)
			results = append(results, rawSymbol{
				Kind:       SymbolFunction,
				Name:       name,
				Signature:  strings.TrimSpace(sig),
				Visibility: goVisibility(name),
				StartLine:  lineNum,
				EndLine:    findGoBlockEnd(lines, i),
			})
			continue
		}

		// Types
		if m := goTypeRe.FindStringSubmatch(line); m != nil {
			name := m[1]
			kind := SymbolType
			if m[2] == "interface" {
				kind = SymbolInterface
			}
			results = append(results, rawSymbol{
				Kind:       kind,
				Name:       name,
				Signature:  strings.TrimSpace(line),
				Visibility: goVisibility(name),
				StartLine:  lineNum,
				EndLine:    findGoBlockEnd(lines, i),
			})
			continue
		}

		// Constants and variables
		if m := goConstVarRe.FindStringSubmatch(line); m != nil {
			declType := m[1]
			name := m[2]
			kind := SymbolConstant
			if declType == "var" {
				kind = SymbolVariable
			}
			results = append(results, rawSymbol{
				Kind:       kind,
				Name:       name,
				Signature:  strings.TrimSpace(line),
				Visibility: goVisibility(name),
				StartLine:  lineNum,
				EndLine:    lineNum,
			})
		}
	}

	return results
}

func goVisibility(name string) string {
	if len(name) > 0 && name[0] >= 'A' && name[0] <= 'Z' {
		return VisibilityExported
	}
	return VisibilityPrivate
}

func findGoBlockEnd(lines []string, startIdx int) int {
	depth := 0
	for i := startIdx; i < len(lines); i++ {
		for _, ch := range lines[i] {
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
	}
	return startIdx + 1
}

// --- TypeScript/JavaScript symbol extraction ---

var (
	tsFuncRe       = regexp.MustCompile(`(?m)^(?:export\s+)?(?:async\s+)?function\s+(\w+)\s*\(([^)]*)\)`)
	tsArrowRe      = regexp.MustCompile(`(?m)^(?:export\s+)?(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?\(([^)]*)\)\s*(?::\s*\w+)?\s*=>`)
	tsClassRe      = regexp.MustCompile(`(?m)^(?:export\s+)?(?:abstract\s+)?class\s+(\w+)`)
	tsInterfaceRe  = regexp.MustCompile(`(?m)^(?:export\s+)?interface\s+(\w+)`)
	tsTypeRe       = regexp.MustCompile(`(?m)^(?:export\s+)?type\s+(\w+)`)
	tsMethodRe     = regexp.MustCompile(`(?m)^\s+(?:(?:public|private|protected|static|async|readonly)\s+)*(\w+)\s*\(([^)]*)\)`)
)

func extractTSSymbols(content string) []rawSymbol {
	lines := strings.Split(content, "\n")
	var results []rawSymbol

	for i, line := range lines {
		lineNum := i + 1

		// Functions
		if m := tsFuncRe.FindStringSubmatch(line); m != nil {
			name := m[1]
			vis := tsVisibility(line)
			results = append(results, rawSymbol{
				Kind:       SymbolFunction,
				Name:       name,
				Signature:  strings.TrimSpace(line),
				Visibility: vis,
				StartLine:  lineNum,
				EndLine:    findBraceBlockEnd(lines, i),
			})
			continue
		}

		// Arrow functions
		if m := tsArrowRe.FindStringSubmatch(line); m != nil {
			name := m[1]
			vis := tsVisibility(line)
			results = append(results, rawSymbol{
				Kind:       SymbolFunction,
				Name:       name,
				Signature:  strings.TrimSpace(line),
				Visibility: vis,
				StartLine:  lineNum,
				EndLine:    findBraceBlockEnd(lines, i),
			})
			continue
		}

		// Classes
		if m := tsClassRe.FindStringSubmatch(line); m != nil {
			name := m[1]
			vis := tsVisibility(line)
			results = append(results, rawSymbol{
				Kind:       SymbolClass,
				Name:       name,
				Signature:  strings.TrimSpace(line),
				Visibility: vis,
				StartLine:  lineNum,
				EndLine:    findBraceBlockEnd(lines, i),
			})
			continue
		}

		// Interfaces
		if m := tsInterfaceRe.FindStringSubmatch(line); m != nil {
			name := m[1]
			vis := tsVisibility(line)
			results = append(results, rawSymbol{
				Kind:       SymbolInterface,
				Name:       name,
				Signature:  strings.TrimSpace(line),
				Visibility: vis,
				StartLine:  lineNum,
				EndLine:    findBraceBlockEnd(lines, i),
			})
			continue
		}

		// Type aliases
		if m := tsTypeRe.FindStringSubmatch(line); m != nil {
			name := m[1]
			vis := tsVisibility(line)
			results = append(results, rawSymbol{
				Kind:       SymbolType,
				Name:       name,
				Signature:  strings.TrimSpace(line),
				Visibility: vis,
				StartLine:  lineNum,
				EndLine:    lineNum,
			})
			continue
		}
	}

	return results
}

func tsVisibility(line string) string {
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "export ") {
		return VisibilityExported
	}
	return VisibilityPrivate
}

func findBraceBlockEnd(lines []string, startIdx int) int {
	depth := 0
	for i := startIdx; i < len(lines); i++ {
		for _, ch := range lines[i] {
			if ch == '{' {
				depth++
			} else if ch == '}' {
				depth--
				if depth == 0 {
					return i + 1
				}
			}
		}
	}
	return startIdx + 1
}

// --- Python symbol extraction ---

var (
	pyFuncRe  = regexp.MustCompile(`(?m)^(\s*)def\s+(\w+)\s*\(([^)]*)\)`)
	pyClassRe = regexp.MustCompile(`(?m)^class\s+(\w+)`)
)

func extractPythonSymbols(content string) []rawSymbol {
	lines := strings.Split(content, "\n")
	var results []rawSymbol

	for i, line := range lines {
		lineNum := i + 1

		// Classes (must check first, before methods inside them)
		if m := pyClassRe.FindStringSubmatch(line); m != nil {
			name := m[1]
			vis := pyVisibility(name)
			results = append(results, rawSymbol{
				Kind:       SymbolClass,
				Name:       name,
				Signature:  strings.TrimSpace(line),
				Visibility: vis,
				StartLine:  lineNum,
				EndLine:    findPyBlockEnd(lines, i),
			})
			continue
		}

		// Functions/methods
		if m := pyFuncRe.FindStringSubmatch(line); m != nil {
			indent := m[1]
			name := m[2]
			params := m[3]
			kind := SymbolFunction
			if len(indent) > 0 {
				kind = SymbolMethod
			}
			vis := pyVisibility(name)
			sig := fmt.Sprintf("def %s(%s)", name, params)
			// Check for docstring on next line
			docstring := ""
			if i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if strings.HasPrefix(nextLine, `"""`) || strings.HasPrefix(nextLine, `'''`) {
					docstring = strings.Trim(nextLine, `"' `)
				}
			}
			results = append(results, rawSymbol{
				Kind:       kind,
				Name:       name,
				Signature:  sig,
				Docstring:  docstring,
				Visibility: vis,
				StartLine:  lineNum,
				EndLine:    findPyBlockEnd(lines, i),
			})
			continue
		}
	}

	return results
}

func pyVisibility(name string) string {
	if strings.HasPrefix(name, "_") {
		return VisibilityPrivate
	}
	return VisibilityExported
}

func findPyBlockEnd(lines []string, startIdx int) int {
	if startIdx >= len(lines) {
		return startIdx + 1
	}

	// Determine the indentation level of this definition
	startIndent := countLeadingSpaces(lines[startIdx])

	for i := startIdx + 1; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue // skip blank lines
		}
		indent := countLeadingSpaces(line)
		if indent <= startIndent {
			return i // line at same or less indentation = block ended
		}
	}
	return len(lines) // extends to end of file
}

func countLeadingSpaces(line string) int {
	count := 0
	for _, ch := range line {
		if ch == ' ' {
			count++
		} else if ch == '\t' {
			count += 4
		} else {
			break
		}
	}
	return count
}

// --- ID generation helpers ---

func makeSymbolID(pathID, name, kind string) string {
	h := sha256.Sum256([]byte(pathID + ":" + kind + ":" + name))
	return hex.EncodeToString(h[:16])
}

func makeRegionID(pathID string, startLine, endLine int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%d", pathID, startLine, endLine)))
	return hex.EncodeToString(h[:16])
}

func hashString(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:16])
}
