package store

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// registerTestPack creates a fake packs/ entry for the given hash in the test store.
func registerTestPack(t *testing.T, root string, hash string) {
	t.Helper()
	_, hexStr, err := ParseHash(hash)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "packs", hexStr), []byte(hash), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestHashContent(t *testing.T) {
	hash := HashContent([]byte("hello world"))
	if !strings.HasPrefix(hash, "sha256:") {
		t.Errorf("expected sha256: prefix, got %s", hash)
	}
	// SHA-256 of "hello world" is known
	expected := "sha256:b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9"
	if hash != expected {
		t.Errorf("expected %s, got %s", expected, hash)
	}
}

func TestHashContentDeterministic(t *testing.T) {
	a := HashContent([]byte("test data"))
	b := HashContent([]byte("test data"))
	if a != b {
		t.Errorf("expected deterministic hash, got %s and %s", a, b)
	}
}

func TestHashContentDifferent(t *testing.T) {
	a := HashContent([]byte("data a"))
	b := HashContent([]byte("data b"))
	if a == b {
		t.Error("expected different hashes for different content")
	}
}

func TestParseHash(t *testing.T) {
	hash := HashContent([]byte("test"))
	algo, hex, err := ParseHash(hash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if algo != "sha256" {
		t.Errorf("expected algorithm sha256, got %s", algo)
	}
	if len(hex) != 64 {
		t.Errorf("expected 64 hex chars, got %d", len(hex))
	}
}

func TestParseHashInvalid(t *testing.T) {
	tests := []string{
		"md5:abc",
		"sha256:short",
		"sha256:zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz",
		"noprefixabcdef0123456789abcdef0123456789abcdef0123456789abcdef012345",
	}
	for _, ref := range tests {
		if ValidateHash(ref) {
			t.Errorf("expected invalid for %q", ref)
		}
	}
}

func TestValidateHash(t *testing.T) {
	hash := HashContent([]byte("valid"))
	if !ValidateHash(hash) {
		t.Errorf("expected valid for %s", hash)
	}
}

func TestNormalizeHash(t *testing.T) {
	hash := HashContent([]byte("test"))
	_, hex, _ := ParseHash(hash)

	// Full reference should pass through
	norm, err := NormalizeHash(hash)
	if err != nil || norm != hash {
		t.Errorf("expected %s, got %s (err: %v)", hash, norm, err)
	}

	// Plain hex should get prefixed
	norm, err = NormalizeHash(hex)
	if err != nil || norm != hash {
		t.Errorf("expected %s, got %s (err: %v)", hash, norm, err)
	}
}

func TestShortHash(t *testing.T) {
	hash := HashContent([]byte("test"))
	short := ShortHash(hash, 12)
	if len(short) != 12 {
		t.Errorf("expected 12 chars, got %d", len(short))
	}
}

func TestResolveHashFullRef(t *testing.T) {
	hash := HashContent([]byte("test"))
	_, hexStr, _ := ParseHash(hash)
	root := setupTestStore(t)
	registerTestPack(t, root, hash)

	// Full sha256: reference
	resolved, err := ResolveHash(root, hash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != hash {
		t.Errorf("expected %s, got %s", hash, resolved)
	}

	// Full plain hex
	resolved, err = ResolveHash(root, hexStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != hash {
		t.Errorf("expected %s, got %s", hash, resolved)
	}
}

func TestResolveHashShortPrefix(t *testing.T) {
	hash := HashContent([]byte("test"))
	_, hexStr, _ := ParseHash(hash)
	root := setupTestStore(t)
	registerTestPack(t, root, hash)

	// 12-char prefix (same as ShortHash output)
	prefix := hexStr[:12]
	resolved, err := ResolveHash(root, prefix)
	if err != nil {
		t.Fatalf("unexpected error for prefix %q: %v", prefix, err)
	}
	if resolved != hash {
		t.Errorf("expected %s, got %s", hash, resolved)
	}

	// Minimum 4-char prefix
	prefix4 := hexStr[:4]
	resolved, err = ResolveHash(root, prefix4)
	if err != nil {
		t.Fatalf("unexpected error for prefix %q: %v", prefix4, err)
	}
	if resolved != hash {
		t.Errorf("expected %s, got %s", hash, resolved)
	}
}

func TestResolveHashCtxURI(t *testing.T) {
	hash := HashContent([]byte("test"))
	_, hexStr, _ := ParseHash(hash)
	root := setupTestStore(t)
	registerTestPack(t, root, hash)

	// Full hex with ctx:// prefix
	resolved, err := ResolveHash(root, "ctx://"+hexStr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != hash {
		t.Errorf("expected %s, got %s", hash, resolved)
	}

	// Short prefix with ctx:// scheme
	resolved, err = ResolveHash(root, "ctx://"+hexStr[:12])
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolved != hash {
		t.Errorf("expected %s, got %s", hash, resolved)
	}
}

func TestResolveHashAmbiguous(t *testing.T) {
	root := setupTestStore(t)

	hex1 := "abcd1111111111111111111111111111111111111111111111111111111111111111"[0:64]
	hex2 := "abcd2222222222222222222222222222222222222222222222222222222222222222"[0:64]
	os.WriteFile(filepath.Join(root, "packs", hex1), []byte("sha256:"+hex1), 0644)
	os.WriteFile(filepath.Join(root, "packs", hex2), []byte("sha256:"+hex2), 0644)

	_, err := ResolveHash(root, "abcd")
	if err == nil {
		t.Fatal("expected ambiguous error, got nil")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Errorf("expected ambiguous error, got: %v", err)
	}
}

func TestResolveHashNotFound(t *testing.T) {
	root := setupTestStore(t)

	_, err := ResolveHash(root, "abcdef123456")
	if err == nil {
		t.Fatal("expected not-found error, got nil")
	}
	if !strings.Contains(err.Error(), "no pack found") {
		t.Errorf("expected not-found error, got: %v", err)
	}
}

func TestResolveHashTooShort(t *testing.T) {
	root := setupTestStore(t)

	_, err := ResolveHash(root, "abc")
	if err == nil {
		t.Fatal("expected too-short error, got nil")
	}
	if !strings.Contains(err.Error(), "too short") {
		t.Errorf("expected too-short error, got: %v", err)
	}
}

func TestResolveHashInvalidHex(t *testing.T) {
	root := setupTestStore(t)

	_, err := ResolveHash(root, "zzzz")
	if err == nil {
		t.Fatal("expected invalid hex error, got nil")
	}
	if !strings.Contains(err.Error(), "not valid hex") {
		t.Errorf("expected invalid hex error, got: %v", err)
	}
}
