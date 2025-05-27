package util

import (
	"runtime"

	_ "go.uber.org/automaxprocs"
)

// Automaxprocs automatically set GOMAXPROCS to match Linux container CPU quota.
// So we are wrapping it here to make sure we do not call it anywhere else without importing automaxprocs.
func GoMaxProcs() int {
	return runtime.GOMAXPROCS(-1)
}
