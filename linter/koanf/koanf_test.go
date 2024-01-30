package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"golang.org/x/tools/go/analysis/analysistest"
)

var (
	incorrectFlag = "incorrect_flag"
	mismatch      = "mismatch"
	unused        = "unused"
)

func testData(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	return filepath.Join(filepath.Dir(wd), "testdata")
}

// Tests koanf/a package that contains two types of errors where:
// - koanf tag doesn't match field name.
// - flag definition doesn't match field name.
// Errors are marked as comments in the package source file.
func TestMismatch(t *testing.T) {
	testdata := testData(t)
	got := errCounts(analysistest.Run(t, testdata, analyzerForTests, "koanf/a"))
	want := map[string]int{
		incorrectFlag: 2,
		mismatch:      1,
	}
	if diff := cmp.Diff(got, want); diff != "" {
		t.Errorf("analysistest.Run() unexpected diff:\n%s\n", diff)
	}
}

func TestUnused(t *testing.T) {
	testdata := testData(t)
	got := errCounts(analysistest.Run(t, testdata, analyzerForTests, "koanf/b"))
	if diff := cmp.Diff(got, map[string]int{"unused": 2}); diff != "" {
		t.Errorf("analysistest.Run() unexpected diff:\n%s\n", diff)
	}
}

func errCounts(res []*analysistest.Result) map[string]int {
	m := make(map[string]int)
	for _, r := range res {
		if rs, ok := r.Result.(Result); ok {
			for _, e := range rs.Errors {
				var s string
				switch {
				case errors.Is(e.err, errIncorrectFlag):
					s = incorrectFlag
				case errors.Is(e.err, errMismatch):
					s = mismatch
				case errors.Is(e.err, errUnused):
					s = unused
				}
				m[s] = m[s] + 1
			}
		}
	}
	return m
}
