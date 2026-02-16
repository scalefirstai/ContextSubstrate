package store

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
)

const hashPrefix = "sha256:"

// HashContent computes the SHA-256 hash of data and returns it in "sha256:<hex>" format.
func HashContent(data []byte) string {
	h := sha256.Sum256(data)
	return hashPrefix + hex.EncodeToString(h[:])
}

// ParseHash splits a "sha256:<hex>" reference into its algorithm and hex parts.
func ParseHash(ref string) (algorithm string, hexStr string, err error) {
	if !strings.HasPrefix(ref, hashPrefix) {
		return "", "", fmt.Errorf("invalid hash reference: expected %s prefix, got %q", hashPrefix, ref)
	}
	hexStr = ref[len(hashPrefix):]
	if len(hexStr) != 64 {
		return "", "", fmt.Errorf("invalid hash reference: expected 64 hex chars, got %d", len(hexStr))
	}
	if _, err := hex.DecodeString(hexStr); err != nil {
		return "", "", fmt.Errorf("invalid hash reference: bad hex encoding: %w", err)
	}
	return "sha256", hexStr, nil
}

// ValidateHash checks whether a hash reference is well-formed.
func ValidateHash(ref string) bool {
	_, _, err := ParseHash(ref)
	return err == nil
}

// NormalizeHash accepts either "sha256:<hex>" or plain "<hex>" and returns "sha256:<hex>".
func NormalizeHash(ref string) (string, error) {
	if strings.HasPrefix(ref, hashPrefix) {
		if !ValidateHash(ref) {
			return "", fmt.Errorf("invalid hash reference: %q", ref)
		}
		return ref, nil
	}
	// Assume plain hex
	full := hashPrefix + ref
	if !ValidateHash(full) {
		return "", fmt.Errorf("invalid hash: %q", ref)
	}
	return full, nil
}

// ShortHash returns the first n characters of the hex portion of a hash reference.
func ShortHash(ref string, n int) string {
	_, hexStr, err := ParseHash(ref)
	if err != nil {
		return ref
	}
	if n > len(hexStr) {
		n = len(hexStr)
	}
	return hexStr[:n]
}

// ResolveHash accepts a full hash, short hex prefix, or ctx:// URI and resolves
// it to a full "sha256:<hex>" reference by searching the packs index.
// Returns an error if the prefix is ambiguous (matches multiple packs) or matches none.
func ResolveHash(storeRoot string, ref string) (string, error) {
	// Strip ctx:// prefix if present
	ref = stripCtxPrefix(ref)

	// Try exact match first (full hash or full hex)
	if normalized, err := NormalizeHash(ref); err == nil {
		return normalized, nil
	}

	// Must be a short prefix — validate it's hex
	if !isHexString(ref) {
		return "", fmt.Errorf("invalid hash prefix: %q is not valid hex", ref)
	}
	if len(ref) < 4 {
		return "", fmt.Errorf("hash prefix too short: need at least 4 characters, got %d", len(ref))
	}

	// Scan packs/ directory for prefix matches
	packsDir := storeRoot + "/packs/"
	entries, err := os.ReadDir(packsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no packs found")
		}
		return "", fmt.Errorf("reading packs index: %w", err)
	}

	var matches []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ref) {
			matches = append(matches, hashPrefix+name)
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("no pack found with prefix %q", ref)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("ambiguous hash prefix %q: matches %d packs", ref, len(matches))
	}
}

// stripCtxPrefix removes "ctx://" from the beginning of a reference if present.
func stripCtxPrefix(ref string) string {
	const ctxPrefix = "ctx://"
	if strings.HasPrefix(ref, ctxPrefix) {
		return ref[len(ctxPrefix):]
	}
	return ref
}

// isHexString checks whether s contains only valid hexadecimal characters.
func isHexString(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
