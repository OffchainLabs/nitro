package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAll(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	testdata := filepath.Join(filepath.Dir(wd), "testdata")
	res := analysistest.Run(t, testdata, analyzerForTests, "rightshift")
	want := []int{6, 11, 12}
	got := erroLines(res)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("analysistest.Ru() unexpected diff in error lines:\n%s\n", diff)
	}
}

func erroLines(errs []*analysistest.Result) []int {
	var ret []int
	for _, e := range errs {
		if r, ok := e.Result.(Result); ok {
			for _, err := range r.Errors {
				ret = append(ret, err.Pos.Line)
			}
		}
	}
	return ret
}
