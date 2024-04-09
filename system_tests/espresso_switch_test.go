package arbtest

import (
	"context"
	"testing"
)

func TestEspressoSwitch(t *testing.T) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder, l2Node, l2Info, cleanup := runNodes(ctx, t)
	defer cleanup()
}
