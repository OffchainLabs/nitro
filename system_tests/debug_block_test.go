//go:build debugblock

package arbtest

import (
	"testing"
)

func TestDebugBlockInjection(t *testing.T) {
	t.Run("debugblock", func(t *testing.T) { testDebugBlockInjection(t, false) })
}
