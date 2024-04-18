package arbtest

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/nitro/cmd/staterecovery"
	"github.com/offchainlabs/nitro/execution/gethexec"
)

func TestRectreateMissingStates(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.execConfig.Caching.Archive = true
	builder.execConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 16
	builder.execConfig.Caching.SnapshotCache = 0 // disable snapshots
	_ = builder.Build(t)
	l2cleanupDone := false
	defer func() {
		if !l2cleanupDone {
			builder.L2.cleanup()
		}
		builder.L1.cleanup()
	}()
	builder.L2Info.GenerateAccount("User2")
	var txs []*types.Transaction
	for i := uint64(0); i < 200; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, common.Big1, nil)
		txs = append(txs, tx)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
	}
	for _, tx := range txs {
		_, err := builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}
	lastBlock, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)
	l2cleanupDone = true
	builder.L2.cleanup()
	t.Log("stopped l2 node")
	func() {
		stack, err := node.New(builder.l2StackConfig)
		Require(t, err)
		defer stack.Close()
		chainDb, err := stack.OpenDatabase("l2chaindata", 0, 0, "l2chaindata/", false)
		Require(t, err)
		defer chainDb.Close()
		cacheConfig := gethexec.DefaultCacheConfigFor(stack, &gethexec.DefaultCachingConfig)
		bc, err := gethexec.GetBlockChain(chainDb, cacheConfig, builder.chainConfig, builder.execConfig.TxLookupLimit)
		Require(t, err)
		err = staterecovery.RecreateMissingStates(chainDb, bc, cacheConfig, 1)
		Require(t, err)
	}()

	testClient, cleanup := builder.Build2ndNode(t, &SecondNodeParams{stackConfig: builder.l2StackConfig})
	defer cleanup()

	currentBlock := uint64(0)
	// wait for the chain to catch up
	for currentBlock < lastBlock {
		currentBlock, err = testClient.Client.BlockNumber(ctx)
		Require(t, err)
		time.Sleep(20 * time.Millisecond)
	}

	currentBlock, err = testClient.Client.BlockNumber(ctx)
	Require(t, err)
	bc := testClient.ExecNode.Backend.ArbInterface().BlockChain()
	triedb := bc.StateCache().TrieDB()
	for i := uint64(0); i <= currentBlock; i++ {
		header := bc.GetHeaderByNumber(i)
		_, err := bc.StateAt(header.Root)
		Require(t, err)
		tr, err := trie.New(trie.TrieID(header.Root), triedb)
		Require(t, err)
		it, err := tr.NodeIterator(nil)
		Require(t, err)
		for it.Next(true) {
		}
		Require(t, it.Error())
	}

	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, common.Big1, nil)
	err = testClient.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = testClient.EnsureTxSucceeded(tx)
	Require(t, err)
}
