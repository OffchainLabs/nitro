//go:build !debugblock

package arbtest

import (
	"testing"
)

func TestDebugBlockInjectionStub(t *testing.T) {
	testDebugBlockInjection(t, false)
}
