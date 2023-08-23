package main

import (
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAll(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get wd: %s", err)
	}
	testdata := filepath.Join(filepath.Dir(wd), "testdata")
	res := analysistest.Run(t, testdata, analyzerForTests, "pointercheck")
	if cnt := countErrors(res); cnt != 6 {
		t.Errorf("analysistest.Run() got %v errors, expected 6", cnt)
	}
}

func countErrors(errs []*analysistest.Result) int {
	cnt := 0
	for _, e := range errs {
		if r, ok := e.Result.(Result); ok {
			cnt += len(r.Errors)
		}
	}
	return cnt
}
