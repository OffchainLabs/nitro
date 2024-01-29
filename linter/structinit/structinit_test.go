package main

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

func TestLinter(t *testing.T) {
	testdata := testData(t)
	got := errCount(analysistest.Run(t, testdata, analyzerForTests, "structinit/a"))
	if got != 2 {
		t.Errorf("analysistest.Run() got %d errors, expected 2", got)
	}
}

func errCount(res []*analysistest.Result) int {
	cnt := 0
	for _, r := range res {
		if rs, ok := r.Result.(Result); ok {
			cnt += len(rs.Errors)
		}
	}
	return cnt
}
