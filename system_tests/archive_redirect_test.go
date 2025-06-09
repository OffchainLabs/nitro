package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"

	"github.com/offchainlabs/nitro/arbos/util"
)

type ethAPIStub struct {
	archiveNodeName string
}

func (e *ethAPIStub) GetBalance(ctx context.Context, address common.Address, blockNrOrHash rpc.BlockNumberOrHash) (*hexutil.Big, error) {
	return nil, fmt.Errorf("error from %s", e.archiveNodeName)
}

func TestRequestForwardingToArchiveNodes(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	cleanup := builder.Build(t)
	defer cleanup()

	// Generate at least 15 blocks so that archive fallback clients will be rerouted these requests to
	builder.L2Info.GenerateAccount("User")
	user := builder.L2Info.GetDefaultTransactOpts("User", ctx)
	var lastTx *types.Transaction
	for i := 0; i < 20; i++ {
		lastTx, _ = builder.L2.TransferBalanceTo(t, "Owner", util.RemapL1Address(user.From), big.NewInt(1e18), builder.L2Info)
	}
	expectBalance, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("Owner"), big.NewInt(16))
	Require(t, err)

	// Start a second node so that it already has blocks and then check that archive requests are routed correctly
	archiveNodeNames := []string{"nodeA", "nodeB", "nodeC", "nodeD"}
	lastBlocks := []uint64{2, 6, 10, 15}
	var archiveConfigs []arbitrum.BlockRedirectConfig
	for i, node := range archiveNodeNames {
		ipcPath := tmpPath(t, node+".ipc")
		var apis []rpc.API
		apis = append(apis, rpc.API{
			Namespace: "eth",
			Version:   "1.0",
			Service:   &ethAPIStub{archiveNodeName: node},
			Public:    false,
		})
		listener, srv, err := rpc.StartIPCEndpoint(ipcPath, apis)
		Require(t, err)
		defer srv.Stop()
		defer listener.Close()
		archiveConfigs = append(archiveConfigs, arbitrum.BlockRedirectConfig{
			URL:       ipcPath,
			Timeout:   time.Second,
			LastBlock: lastBlocks[i],
		})
	}
	execConfig := builder.execConfig
	execConfig.RPC.BlockRedirects = archiveConfigs
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{execConfig: execConfig})
	defer cleanupB()

	// Ensure that clientB has caught up by checking if it has the lastTx
	_, err = testClientB.EnsureTxSucceeded(lastTx)
	Require(t, err)

	// Only nodeA should receive this request
	_, err = testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("Owner"), big.NewInt(2))
	t.Logf("error from archive redirect: %s", err.Error())
	if err.Error() != "error from nodeA" {
		t.Fatalf("unexpected error from archive redirect. Got: %s, Want: error from nodeA", err.Error())
	}

	// Only nodeB should receive this request
	_, err = testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("Owner"), big.NewInt(5))
	t.Logf("error from archive redirect: %s", err.Error())
	if err.Error() != "error from nodeB" {
		t.Fatalf("unexpected error from archive redirect. Got: %s, Want: error from nodeB", err.Error())
	}

	// Only nodeC should receive this request
	_, err = testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("Owner"), big.NewInt(10))
	t.Logf("error from archive redirect: %s", err.Error())
	if err.Error() != "error from nodeC" {
		t.Fatalf("unexpected error from archive redirect. Got: %s, Want: error from nodeC", err.Error())
	}

	// Only nodeD should receive this request
	_, err = testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("Owner"), big.NewInt(15))
	if err.Error() != "error from nodeD" {
		t.Fatalf("unexpected error from archive redirect. Got: %s, Want: error from nodeD", err.Error())
	}
	// Blocks > 15 should not be routed to any archive nodes so BalanceAt would not error
	val, err := testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("Owner"), big.NewInt(16))
	Require(t, err)
	if val.Cmp(expectBalance) != 0 {
		t.Fatal("unexpected balance of owner")
	}
}
