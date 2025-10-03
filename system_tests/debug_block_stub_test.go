//go:build !debugblock

package arbtest

import (
	"testing"
)

func TestDebugBlockInjection(t *testing.T) {
	t.Run("production", func(t *testing.T) { testDebugBlockInjection(t, true) })
}
