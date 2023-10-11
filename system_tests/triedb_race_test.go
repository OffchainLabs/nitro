package arbtest

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestTrieDBCommitRace(t *testing.T) {
	_ = testhelpers.InitTestLog(t, log.LvlError)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	execConfig := gethexec.ConfigDefaultTest()
	execConfig.RPC.MaxRecreateStateDepth = arbitrum.InfiniteMaxRecreateStateDepth
	execConfig.Sequencer.MaxBlockSpeed = 0
	execConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	execConfig.Caching.Archive = true
	execConfig.Caching.BlockCount = 127
	execConfig.Caching.BlockAge = 0
	execConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 127
	execConfig.Caching.MaxAmountOfGasToSkipStateSaving = 0
	l2info, node, l2client, _, _, _, l1stack := createTestNodeOnL1WithConfig(t, ctx, true, nil, execConfig, nil, nil)
	cancel = func() {
		defer requireClose(t, l1stack)
		defer node.StopAndWait()
	}
	defer cancel()
	execNode := getExecNode(t, node)
	l2info.GenerateAccount("User2")
	bc := execNode.Backend.ArbInterface().BlockChain()

	var wg sync.WaitGroup
	quit := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			default:
				TransferBalance(t, "Faucet", "User2", common.Big1, l2info, l2client, ctx)
			case <-quit:
				return
			}
		}
	}()
	api := execNode.Backend.APIBackend()
	blockNumber := 1
	for i := 0; i < 5; i++ {
		var roots []common.Hash
		for len(roots) < 1024 {
			select {
			default:
				block, err := api.BlockByNumber(ctx, rpc.BlockNumber(blockNumber))
				if err == nil && block != nil {
					root := block.Root()
					if statedb, err := bc.StateAt(root); err == nil {
						err := statedb.Database().TrieDB().Reference(root, common.Hash{})
						Require(t, err)
						roots = append(roots, root)
					}
					blockNumber += 1
				}
			case <-quit:
				return
			}
		}
		t.Log("dereferencing...")
		for _, root := range roots {
			err := bc.TrieDB().Dereference(root)
			Require(t, err)
			time.Sleep(1)
		}
	}
	close(quit)
	wg.Wait()
}
