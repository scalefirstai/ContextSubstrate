package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/contextsubstrate/ctx/internal/graph"
)

// CacheDir is the subdirectory name within .ctx/ for the cache.
const CacheDir = "cache"

// CacheEntry represents a cached artifact keyed by content hash.
type CacheEntry struct {
	Key         string          `json:"key"`
	ArtifactType string        `json:"artifact_type"`
	ScopeType   string          `json:"scope_type"`
	ScopeID     string          `json:"scope_id"`
	ContentHash string          `json:"content_hash"`
	Model       string          `json:"model,omitempty"`
	Payload     json.RawMessage `json:"payload"`
	TokensIn    int             `json:"tokens_in,omitempty"`
	TokensOut   int             `json:"tokens_out,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

// InitCache creates the .ctx/cache/ directory.
func InitCache(storeRoot string) error {
	cacheDir := filepath.Join(storeRoot, CacheDir)
	return os.MkdirAll(cacheDir, 0755)
}

func cachePath(storeRoot string) string {
	return filepath.Join(storeRoot, CacheDir, "entries.jsonl")
}

// Get retrieves a cached entry by content hash and artifact type.
// Returns nil if not found.
func Get(storeRoot, contentHash, artifactType string) (*CacheEntry, error) {
	entries, err := graph.ReadRecords[CacheEntry](cachePath(storeRoot))
	if err != nil {
		return nil, fmt.Errorf("reading cache: %w", err)
	}

	for i := range entries {
		if entries[i].ContentHash == contentHash && entries[i].ArtifactType == artifactType {
			return &entries[i], nil
		}
	}

	return nil, nil
}

// Put stores a cache entry. If an entry with the same key exists, it is replaced.
func Put(storeRoot string, entry *CacheEntry) error {
	if err := InitCache(storeRoot); err != nil {
		return err
	}

	if entry.Key == "" {
		entry.Key = makeCacheKey(entry.ContentHash, entry.ArtifactType, entry.ScopeID)
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}

	// Read existing entries
	existing, err := graph.ReadRecords[CacheEntry](cachePath(storeRoot))
	if err != nil {
		return fmt.Errorf("reading cache: %w", err)
	}

	// Replace or append
	replaced := false
	for i := range existing {
		if existing[i].Key == entry.Key {
			existing[i] = *entry
			replaced = true
			break
		}
	}
	if !replaced {
		existing = append(existing, *entry)
	}

	// Sort by key for deterministic output
	sort.Slice(existing, func(i, j int) bool {
		return existing[i].Key < existing[j].Key
	})

	// Write back
	records := make([]any, len(existing))
	for i := range existing {
		records[i] = existing[i]
	}
	return graph.WriteRecords(cachePath(storeRoot), records)
}

// Invalidate removes cache entries whose ContentHash matches any of the given hashes.
// Returns the number of entries removed.
func Invalidate(storeRoot string, contentHashes []string) (int, error) {
	existing, err := graph.ReadRecords[CacheEntry](cachePath(storeRoot))
	if err != nil {
		return 0, fmt.Errorf("reading cache: %w", err)
	}

	if len(existing) == 0 {
		return 0, nil
	}

	hashSet := make(map[string]bool, len(contentHashes))
	for _, h := range contentHashes {
		hashSet[h] = true
	}

	var kept []any
	removed := 0
	for _, e := range existing {
		if hashSet[e.ContentHash] {
			removed++
		} else {
			kept = append(kept, e)
		}
	}

	if removed > 0 {
		if err := graph.WriteRecords(cachePath(storeRoot), kept); err != nil {
			return 0, fmt.Errorf("writing cache: %w", err)
		}
	}

	return removed, nil
}

// List returns all cache entries.
func List(storeRoot string) ([]CacheEntry, error) {
	return graph.ReadRecords[CacheEntry](cachePath(storeRoot))
}

func makeCacheKey(contentHash, artifactType, scopeID string) string {
	h := sha256.Sum256([]byte(contentHash + ":" + artifactType + ":" + scopeID))
	return hex.EncodeToString(h[:16])
}
