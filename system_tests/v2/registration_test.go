// Copyright 2021-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package v2

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRegistrationSafety scans all .go files in this package for functions
// matching testConfig* and testRun* patterns, then verifies that every
// testRun* has a corresponding registration in init(). This catches the
// silent failure where you write a test but forget to register it.
func TestRegistrationSafety(t *testing.T) {
	fset := token.NewFileSet()

	// Find the package directory.
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	// Parse all .go files in the package (non-test files only for the
	// functions, since testConfig/testRun live in non-test files).
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}

	var configFuncs []string
	var runFuncs []string

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		path := filepath.Join(dir, name)
		file, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}

		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Recv != nil {
				continue // skip methods
			}
			fname := fn.Name.Name
			if strings.HasPrefix(fname, "testConfig") {
				configFuncs = append(configFuncs, fname)
			}
			if strings.HasPrefix(fname, "testRun") {
				runFuncs = append(runFuncs, fname)
			}
		}
	}

	// Build a set of registered test names from the registry.
	registered := make(map[string]bool)
	for _, entry := range GetRegistry() {
		registered[entry.Name] = true
	}
	for _, suite := range GetSuiteRegistry() {
		registered[suite.Name] = true
	}

	// Check that every testRun* function has a corresponding registration.
	// We match by suffix: testRunTransfer -> "Transfer" should appear in some
	// registered test name.
	for _, runFunc := range runFuncs {
		suffix := strings.TrimPrefix(runFunc, "testRun")
		found := false
		for name := range registered {
			// Registered names are like "TestTransfer" or "Retryables".
			// The suffix "Transfer" should appear in the registered name.
			if strings.Contains(name, suffix) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("function %s() exists but no matching RegisterTest/RegisterSuite found (expected a registration containing %q)", runFunc, suffix)
		}
	}

	if len(runFuncs) == 0 {
		t.Log("warning: no testRun* functions found — is this package empty?")
	}

	t.Logf("registration check: %d testConfig*, %d testRun*, %d registered entries",
		len(configFuncs), len(runFuncs), len(registered))
}
