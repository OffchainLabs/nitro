// Copyright 2023-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package structinit

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"
)

const aPackagePath = "github.com/offchainlabs/nitro/linters/testdata/src/structinit/a"
const bPackagePath = "github.com/offchainlabs/nitro/linters/testdata/src/structinit/b"

func TestFieldCountingInSinglePackage(t *testing.T) {
	result := analysistest.Run(t, getModuleRoot(t), Analyzer, aPackagePath)
	require.Equal(t, 1, len(result),
		"Expected single result - analysis was run for a single package")

	actual := extractErrorMessages(result[0])
	// a.go contains two incorrect initializations of InterestingStruct
	expected := []string{
		errorMessage(aPackagePath+".InterestingStruct", 1, 2),
		errorMessage(aPackagePath+".InterestingStruct", 1, 2),
	}
	require.ElementsMatch(t, actual, expected)
}

func TestFieldCountingAcrossPackages(t *testing.T) {
	result := analysistest.Run(t, getModuleRoot(t), Analyzer, bPackagePath)
	require.Equal(t, 1, len(result),
		"Expected two results - analysis was run for a single package")

	actual := extractErrorMessages(result[0])
	// b.go contains a single incorrect initialization of InterestingStruct
	expected := []string{
		errorMessage(aPackagePath+".InterestingStruct", 0, 2),
	}
	require.ElementsMatch(t, actual, expected)
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

func extractErrorMessages(analyzerResult *analysistest.Result) []string {
	var errors []string
	if structErrs, ok := analyzerResult.Result.([]structError); ok {
		for _, structErr := range structErrs {
			errors = append(errors, structErr.Message)
		}
	}
	return errors
}
