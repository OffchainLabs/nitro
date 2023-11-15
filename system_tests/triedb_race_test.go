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
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestTrieDBCommitRace(t *testing.T) {
	_ = testhelpers.InitTestLog(t, log.LvlError)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.execConfig.RPC.MaxRecreateStateDepth = arbitrum.InfiniteMaxRecreateStateDepth
	builder.execConfig.Sequencer.MaxBlockSpeed = 0
	builder.execConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	builder.execConfig.Caching.Archive = true
	builder.execConfig.Caching.BlockCount = 127
	builder.execConfig.Caching.BlockAge = 0
	builder.execConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 127
	builder.execConfig.Caching.MaxAmountOfGasToSkipStateSaving = 0
	cleanup := builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")
	bc := builder.L2.ExecNode.Backend.ArbInterface().BlockChain()

	var wg sync.WaitGroup
	quit := make(chan struct{})
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			default:
				builder.L2.TransferBalance(t, "Faucet", "User2", common.Big1, builder.L2Info)
			case <-quit:
				return
			}
		}
	}()
	api := builder.L2.ExecNode.Backend.APIBackend()
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
