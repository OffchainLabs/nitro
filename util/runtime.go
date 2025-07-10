package util

import (
	"runtime"

	"go.uber.org/automaxprocs/maxprocs"
)

func init() {
	// Disable maxprocs logs
	_, _ = maxprocs.Set()
}

// GoMaxProcs wraps runtime.GOMAXPROCS here to ensure that maxprocs.Set()
// is always called first.
func GoMaxProcs() int {
	return runtime.GOMAXPROCS(-1)
}
