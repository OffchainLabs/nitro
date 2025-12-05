// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
//
// Based on https://github.com/andydotdev/omitlint

package jsonneverempty

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

const aPackagePath = "github.com/offchainlabs/nitro/linters/testdata/src/jsonneverempty/a"

func TestOmitemptyTagValidity(t *testing.T) {
	analysistest.Run(t, getModuleRoot(t), Analyzer, aPackagePath)
}

func getModuleRoot(t *testing.T) string {
	t.Helper()

	var out bytes.Buffer
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Dir}}")
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to get module root directoryy: %v", err)
	}
	parts := strings.Split(out.String(), "\n")
	return strings.TrimSpace(parts[0])
}
