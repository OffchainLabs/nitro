package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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
	for i := 0; i < 20; i++ {
		builder.L2.TransferBalanceTo(t, "Owner", util.RemapL1Address(user.From), big.NewInt(1e18), builder.L2Info)
	}
	expectBalance, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("Owner"), big.NewInt(16))
	Require(t, err)

	// Start a second node so that it already has blocks and then check that archive requests are routed correctly
	archiveNodeNames := []string{"nodeA", "nodeB", "nodeC", "nodeD"}
	lastBlocks := []uint64{2, 6, 10, 15}
	var archiveConfigs []arbitrum.ArchiveRedirectConfig
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
		archiveConfigs = append(archiveConfigs, arbitrum.ArchiveRedirectConfig{
			URL:       ipcPath,
			Timeout:   time.Second,
			LastBlock: lastBlocks[i],
		})
	}
	execConfig := builder.execConfig
	execConfig.RPC.ArchiveRedirects = archiveConfigs
	testClientB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{execConfig: execConfig})
	defer cleanupB()

	// All nodes should be able to receive this- so one of them will be randomly picked, so we'll check against general target string
	_, err = testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("Owner"), big.NewInt(2))
	t.Logf("error from archive redirect: %s", err.Error())
	if err == nil || !strings.Contains(err.Error(), "error from") {
		t.Fatal("incorrect form of error received")
	}

	// nodeA should not be considered in the random pick so we check againt that not being selected
	_, err = testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("Owner"), big.NewInt(5))
	t.Logf("error from archive redirect: %s", err.Error())
	if err == nil || !strings.Contains(err.Error(), "error from") || err.Error() == "error from nodeA" {
		t.Fatal("incorrect form of error received")
	}

	// nodeA and nodeB should not be considered in the random pick so we check againt that not being selected
	_, err = testClientB.Client.BalanceAt(ctx, builder.L2Info.GetAddress("Owner"), big.NewInt(10))
	t.Logf("error from archive redirect: %s", err.Error())
	if err.Error() != "error from nodeC" && err.Error() != "error from nodeD" {
		t.Fatal("incorrect form of error received")
	}

	// nodeD should be the only one to be considered in the random pick so we check againt that being selected
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
