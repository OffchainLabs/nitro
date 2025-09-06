package structinit

import (
	"bytes"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/tools/go/analysis/analysistest"
)

const aPackagePath = "github.com/offchainlabs/nitro/linters/testdata/src/structinit/a"
const bPackagePath = "github.com/offchainlabs/nitro/linters/testdata/src/structinit/b"

func TestFieldCountInSinglePackage(t *testing.T) {
	result := analysistest.Run(t, getModuleRoot(t), FieldCountAnalyzer, aPackagePath)
	require.Equal(t, 1, len(result),
		"Expected single result - analysis was run for a single package")

	actual := result[0].Result.(fieldCounts)
	expected := fieldCounts{aPackagePath + ".InterestingStruct": 2}
	require.True(t, reflect.DeepEqual(actual, expected))
}

func TestFieldCountAcrossPackages(t *testing.T) {
	result := analysistest.Run(t, getModuleRoot(t), FieldCountAnalyzer, bPackagePath)
	require.Equal(t, 1, len(result),
		"Expected two results - analysis was run for a single package")

	actual := result[0].Result.(fieldCounts)
	expected := fieldCounts{aPackagePath + ".InterestingStruct": 2, bPackagePath + ".AnotherStruct": 1}
	require.True(t, reflect.DeepEqual(actual, expected))
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

	return strings.TrimSpace(out.String())
}
