package structinit

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestLinter(t *testing.T) {
	got := errCount(analysistest.Run(t, getModuleRoot(t), analyzerForTests, aPackagePath))
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
