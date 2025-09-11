// Copyright 2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package namedfieldsinit

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func testData(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	return filepath.Join(filepath.Dir(wd), "testdata")
}

func TestNamedFieldsInit(t *testing.T) {
	testdata := testData(t)
	analysistest.Run(t, testdata, Analyzer, "namedfieldsinit", "namedfieldsinit/otherpkg")
}
