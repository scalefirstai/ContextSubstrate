package index

import (
	"testing"

	"github.com/contextsubstrate/ctx/internal/graph"
)

func TestExtractGoSymbols(t *testing.T) {
	content := []byte(`package main

import "fmt"

const Version = "1.0"

var debug = false

type Config struct {
	Name string
	Port int
}

type Reader interface {
	Read(p []byte) (int, error)
}

func main() {
	fmt.Println("hello")
}

func helper(x int) string {
	return fmt.Sprintf("%d", x)
}

func (c *Config) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("name required")
	}
	return nil
}
`)

	symbols, regions := ExtractSymbols(content, "go", "abc123", "pathid1")

	if len(symbols) == 0 {
		t.Fatal("expected symbols from Go file")
	}
	if len(regions) == 0 {
		t.Fatal("expected regions from Go file")
	}

	// Check we found the expected symbols
	found := make(map[string]string)
	for _, s := range symbols {
		found[s.Name] = s.Kind
	}

	expected := map[string]string{
		"Version":         SymbolConstant,
		"debug":           SymbolVariable,
		"Config":          SymbolType,
		"Reader":          SymbolInterface,
		"main":            SymbolFunction,
		"helper":          SymbolFunction,
		"Config.Validate": SymbolMethod,
	}

	for name, kind := range expected {
		gotKind, ok := found[name]
		if !ok {
			t.Errorf("missing symbol %q", name)
		} else if gotKind != kind {
			t.Errorf("symbol %q: got kind %q, want %q", name, gotKind, kind)
		}
	}

	// Check visibility
	for _, s := range symbols {
		if s.Name == "main" && s.Visibility != VisibilityPrivate {
			t.Errorf("main should be private, got %q", s.Visibility)
		}
		if s.Name == "Config" && s.Visibility != VisibilityExported {
			t.Errorf("Config should be exported, got %q", s.Visibility)
		}
	}
}

func TestExtractTSSymbols(t *testing.T) {
	content := []byte(`
export function greet(name: string): string {
  return "hello " + name;
}

function privateHelper() {
  console.log("private");
}

export const handler = async (req: Request) => {
  return new Response("ok");
}

export class UserService {
  constructor(private db: Database) {}

  async getUser(id: string): Promise<User> {
    return this.db.find(id);
  }
}

export interface Config {
  port: number;
  host: string;
}

export type UserID = string;
`)

	symbols, _ := ExtractSymbols(content, "typescript", "abc123", "pathid2")

	if len(symbols) == 0 {
		t.Fatal("expected symbols from TypeScript file")
	}

	found := make(map[string]string)
	for _, s := range symbols {
		found[s.Name] = s.Kind
	}

	if _, ok := found["greet"]; !ok {
		t.Error("missing 'greet' function")
	}
	if _, ok := found["privateHelper"]; !ok {
		t.Error("missing 'privateHelper' function")
	}
	if _, ok := found["handler"]; !ok {
		t.Error("missing 'handler' arrow function")
	}
	if _, ok := found["UserService"]; !ok {
		t.Error("missing 'UserService' class")
	}
	if _, ok := found["Config"]; !ok {
		t.Error("missing 'Config' interface")
	}
	if _, ok := found["UserID"]; !ok {
		t.Error("missing 'UserID' type")
	}

	// Check visibility
	for _, s := range symbols {
		if s.Name == "greet" && s.Visibility != VisibilityExported {
			t.Errorf("greet should be exported, got %q", s.Visibility)
		}
		if s.Name == "privateHelper" && s.Visibility != VisibilityPrivate {
			t.Errorf("privateHelper should be private, got %q", s.Visibility)
		}
	}
}

func TestExtractPythonSymbols(t *testing.T) {
	content := []byte(`
class UserService:
    """Manages user operations."""

    def __init__(self, db):
        """Initialize with database."""
        self.db = db

    def get_user(self, user_id):
        """Get a user by ID."""
        return self.db.find(user_id)

    def _validate(self, data):
        pass

def process_request(req):
    """Process an incoming request."""
    return req.body

def _internal_helper():
    pass
`)

	symbols, _ := ExtractSymbols(content, "python", "abc123", "pathid3")

	if len(symbols) == 0 {
		t.Fatal("expected symbols from Python file")
	}

	found := make(map[string]string)
	for _, s := range symbols {
		found[s.Name] = s.Kind
	}

	if kind, ok := found["UserService"]; !ok || kind != SymbolClass {
		t.Errorf("expected UserService class, got %v", found["UserService"])
	}
	if kind, ok := found["process_request"]; !ok || kind != SymbolFunction {
		t.Errorf("expected process_request function, got %v", found["process_request"])
	}
	if kind, ok := found["_internal_helper"]; !ok || kind != SymbolFunction {
		t.Errorf("expected _internal_helper function, got %v", found["_internal_helper"])
	}

	// Check Python visibility
	for _, s := range symbols {
		if s.Name == "_internal_helper" && s.Visibility != VisibilityPrivate {
			t.Errorf("_internal_helper should be private, got %q", s.Visibility)
		}
		if s.Name == "process_request" && s.Visibility != VisibilityExported {
			t.Errorf("process_request should be exported, got %q", s.Visibility)
		}
	}
}

func TestExtractSymbolsUnsupportedLanguage(t *testing.T) {
	content := []byte("some content")
	symbols, regions := ExtractSymbols(content, "unknown", "abc", "pid")
	if symbols != nil || regions != nil {
		t.Error("expected nil for unsupported language")
	}
}

func TestExtractSymbolsEmptyContent(t *testing.T) {
	symbols, regions := ExtractSymbols(nil, "go", "abc", "pid")
	if symbols != nil || regions != nil {
		t.Error("expected nil for empty content")
	}
}

func TestExtractImportsGo(t *testing.T) {
	content := []byte(`package main

import "fmt"

import (
	"os"
	"strings"
	mylib "github.com/example/lib"
)

func main() {}
`)

	imports := ExtractImports(content, "go", "abc123", "pathid1", nil)

	if len(imports) != 4 {
		t.Fatalf("expected 4 imports, got %d", len(imports))
	}

	modules := make(map[string]bool)
	for _, imp := range imports {
		modules[imp.ToExternalModule] = true
	}

	for _, expected := range []string{"fmt", "os", "strings", "github.com/example/lib"} {
		if !modules[expected] {
			t.Errorf("missing import %q", expected)
		}
	}
}

func TestExtractImportsTS(t *testing.T) {
	content := []byte(`
import { foo } from 'bar';
import React from 'react';
const fs = require('fs');
const mod = import('./dynamic');
`)

	imports := ExtractImports(content, "typescript", "abc123", "pathid1", nil)

	if len(imports) < 3 {
		t.Fatalf("expected at least 3 imports, got %d", len(imports))
	}

	modules := make(map[string]bool)
	for _, imp := range imports {
		if imp.ToExternalModule != "" {
			modules[imp.ToExternalModule] = true
		}
	}

	for _, expected := range []string{"bar", "react", "fs"} {
		if !modules[expected] {
			t.Errorf("missing import %q", expected)
		}
	}
}

func TestExtractImportsPython(t *testing.T) {
	content := []byte(`
import os
import sys
from pathlib import Path
from collections import defaultdict
`)

	imports := ExtractImports(content, "python", "abc123", "pathid1", nil)

	if len(imports) < 3 {
		t.Fatalf("expected at least 3 imports, got %d", len(imports))
	}

	modules := make(map[string]bool)
	for _, imp := range imports {
		modules[imp.ToExternalModule] = true
	}

	for _, expected := range []string{"os", "sys", "pathlib"} {
		if !modules[expected] {
			t.Errorf("missing import %q", expected)
		}
	}
}

func TestExtractCallEdgesWithRegions(t *testing.T) {
	content := []byte(`package main

func caller() {
	helper()
	fmt.Println("hello")
}

func helper() {
	doWork()
}
`)

	symbols, regions := ExtractSymbols(content, "go", "abc", "pid")

	knownSymbols := make(map[string]string)
	for _, s := range symbols {
		knownSymbols[s.Name] = s.SymbolID
	}

	edges := ExtractCallEdgesWithRegions(content, "go", "abc", symbols, regions, knownSymbols)

	if len(edges) == 0 {
		t.Fatal("expected call edges")
	}

	// Check that caller → helper is found
	foundHelperCall := false
	for _, e := range edges {
		if e.ToSymbolID == knownSymbols["helper"] {
			foundHelperCall = true
			if e.Confidence < 0.7 {
				t.Errorf("resolved call should have higher confidence, got %f", e.Confidence)
			}
		}
	}
	if !foundHelperCall {
		t.Error("expected call edge from caller to helper")
	}
}

func TestIndexCommitWithSymbols(t *testing.T) {
	repoRoot := setupTestRepo(t)
	storeRoot := t.TempDir() + "/.ctx"

	sha, err := GetHeadSHA(repoRoot)
	if err != nil {
		t.Fatal(err)
	}

	if err := IndexCommit(storeRoot, repoRoot, sha); err != nil {
		t.Fatalf("IndexCommit: %v", err)
	}

	// The test repo has .go files, so we should have symbols
	symbols, err := graph.ReadRecords[graph.SymbolRecord](graph.SymbolsPath(storeRoot, sha))
	if err != nil {
		t.Fatalf("reading symbols: %v", err)
	}

	if len(symbols) == 0 {
		t.Error("expected symbols to be extracted from Go files")
	}

	// Check for known functions from setupTestRepo (main, helper)
	foundMain := false
	for _, s := range symbols {
		if s.Name == "main" && s.Kind == SymbolFunction {
			foundMain = true
		}
	}
	if !foundMain {
		t.Error("expected to find 'main' function symbol")
	}

	// Check regions were created
	regions, err := graph.ReadRecords[graph.RegionRecord](graph.RegionsPath(storeRoot, sha))
	if err != nil {
		t.Fatalf("reading regions: %v", err)
	}
	if len(regions) == 0 {
		t.Error("expected regions to be created")
	}

	// Check import edges were created (main.go imports "fmt")
	imports, err := graph.ReadRecords[graph.ImportEdge](graph.ImportEdgesPath(storeRoot, sha))
	if err != nil {
		t.Fatalf("reading imports: %v", err)
	}
	if len(imports) == 0 {
		t.Error("expected import edges to be created")
	}
}
