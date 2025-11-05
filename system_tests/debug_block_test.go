//go:build debugblock

package arbtest

import (
	"testing"
)

func TestExperimentalDebugBlockInjection(t *testing.T) {
	testDebugBlockInjection(t, true)
}
