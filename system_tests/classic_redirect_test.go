// Copyright 2021-2025, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestClassicRedirectURLNotLeaked(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// nitro will expect that listener provides an RPC server.
	// However we don't need to provide a real RPC server here, so calls to it will fail
	listener, err := testhelpers.FreeTCPPortListener()
	Require(t, err)
	defer listener.Close()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, false)
	builder.execConfig.RPC.ClassicRedirect = "http://" + listener.Addr().String()
	builder.execConfig.RPC.ClassicRedirectTimeout = time.Second
	cleanup := builder.Build(t)
	defer cleanup()

	l2rpc := builder.L2.Stack.Attach()

	var result traceResult
	err = l2rpc.CallContext(ctx, &result, "arbtrace_call", callTxArgs{}, []string{"trace"}, rpc.BlockNumberOrHash{})
	// checks that it errors and that the error message does not contain the classic redirect URL
	expectedErrMsg := "Failed to call fallback API"
	if err == nil || err.Error() != expectedErrMsg {
		t.Fatalf("Expected error message to be %s, got %v", expectedErrMsg, err)
	}
}
