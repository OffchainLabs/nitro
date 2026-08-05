// Copyright 2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

// External test package: blank-imports the test subpackages so their
// registrations run, then hands off to systest.Run. Lives outside
// package systest to avoid an import cycle (systest can't import the suites).
package systest_test

import (
	"fmt"
	"os"
	"strings"
	"testing"

	_ "github.com/offchainlabs/nitro/system_tests/v2/smoketest"
	"github.com/offchainlabs/nitro/system_tests/v2/systest"
)

func TestMain(m *testing.M) {
	systest.RunTestMain(m)
}

func TestRunner(t *testing.T) {
	systest.Run(t)
}

func TestSubpackagesAreImported(t *testing.T) {
	entries, err := os.ReadDir("../")
	if err != nil {
		t.Fatalf("read v2 dir: %v", err)
	}
	src, err := os.ReadFile("runner_test.go")
	if err != nil {
		t.Fatalf("read runner_test.go: %v", err)
	}
	runnerSrc := string(src)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if name == "systest" || strings.HasPrefix(name, ".") {
			continue
		}
		want := fmt.Sprintf(`"github.com/offchainlabs/nitro/system_tests/v2/%s"`, name)
		if !strings.Contains(runnerSrc, want) {
			t.Errorf("subpackage %q not blank-imported in runner_test.go — add:\n\t_ %s", name, want)
		}
	}
}
