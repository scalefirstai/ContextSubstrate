package graph

import "time"

// CommitRecord represents a git commit identity in the context graph.
type CommitRecord struct {
	Type       string    `json:"type"`
	Repo       string    `json:"repo"`
	SHA        string    `json:"sha"`
	ParentSHA  string    `json:"parent_sha,omitempty"`
	Author     string    `json:"author"`
	Message    string    `json:"message"`
	AuthoredAt time.Time `json:"authored_at"`
}

// PathRecord represents a stable file path identity tracked across commits.
type PathRecord struct {
	Type            string  `json:"type"`
	PathID          string  `json:"path_id"`
	Repo            string  `json:"repo"`
	Path            string  `json:"path"`
	FirstSeenCommit string  `json:"first_seen_commit"`
	LastSeenCommit  *string `json:"last_seen_commit,omitempty"`
}

// FileSnapshot captures a file's state at a specific commit.
type FileSnapshot struct {
	Type          string `json:"type"`
	Commit        string `json:"commit"`
	PathID        string `json:"path_id"`
	BlobOID       string `json:"blob_oid"`
	ContentSHA256 string `json:"content_sha256"`
	Language      string `json:"language"`
	ByteSize      int    `json:"byte_size"`
	LOC           int    `json:"loc"`
	IsGenerated   bool   `json:"is_generated"`
	IsBinary      bool   `json:"is_binary"`
}

// SymbolRecord represents a symbol definition (function, class, type, etc.).
type SymbolRecord struct {
	Type        string `json:"type"`
	Commit      string `json:"commit"`
	SymbolID    string `json:"symbol_id"`
	PathID      string `json:"path_id"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	FQName      string `json:"fqname"`
	Visibility  string `json:"visibility"`
	Language    string `json:"language"`
	Signature   string `json:"signature,omitempty"`
	Docstring   string `json:"docstring,omitempty"`
	SymbolHash  string `json:"symbol_hash"`
	DefRegionID string `json:"def_region_id"`
}

// RegionRecord represents a text span within a file.
type RegionRecord struct {
	Type       string `json:"type"`
	Commit     string `json:"commit"`
	RegionID   string `json:"region_id"`
	PathID     string `json:"path_id"`
	RegionHash string `json:"region_hash"`
	Purpose    string `json:"purpose"`
	StartLine  int    `json:"start_line"`
	StartCol   int    `json:"start_col"`
	EndLine    int    `json:"end_line"`
	EndCol     int    `json:"end_col"`
}

// ImportEdge represents a file-level import dependency.
type ImportEdge struct {
	Type             string `json:"type"`
	Commit           string `json:"commit"`
	FromPathID       string `json:"from_path_id"`
	ToPathID         string `json:"to_path_id,omitempty"`
	ToExternalModule string `json:"to_external_module,omitempty"`
	ImportRegionID   string `json:"import_region_id,omitempty"`
}

// CallEdge represents a symbol-level call dependency.
type CallEdge struct {
	Type           string  `json:"type"`
	Commit         string  `json:"commit"`
	FromSymbolID   string  `json:"from_symbol_id"`
	ToSymbolID     string  `json:"to_symbol_id,omitempty"`
	ToExternalRef  string  `json:"to_external_ref,omitempty"`
	CallRegionID   string  `json:"call_region_id,omitempty"`
	CallType       string  `json:"call_type,omitempty"`
	Confidence     float64 `json:"confidence,omitempty"`
}

// Record type constants for the Type field.
const (
	TypeCommit       = "commit"
	TypePath         = "path"
	TypeFileSnapshot = "file_snapshot"
	TypeSymbol       = "symbol"
	TypeRegion       = "region"
	TypeImportEdge   = "import_edge"
	TypeCallEdge     = "call_edge"
)
