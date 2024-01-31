package arbtest

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/pruning"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func countStateEntries(db ethdb.Iteratee) int {
	entries := 0
	it := db.NewIterator(nil, nil)
	for it.Next() {
		isCode, _ := rawdb.IsCodeKey(it.Key())
		if len(it.Key()) == common.HashLength || isCode {
			entries++
		}
	}
	it.Release()
	return entries
}

func TestPruning(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
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
		chainDb, err := stack.OpenDatabase("chaindb", 0, 0, "", false)
		Require(t, err)
		defer chainDb.Close()
		chainDbEntriesBeforePruning := countStateEntries(chainDb)

		prand := testhelpers.NewPseudoRandomDataSource(t, 1)
		var testKeys [][]byte
		for i := 0; i < 100; i++ {
			// generate test keys with length of hash to emulate legacy state trie nodes
			testKeys = append(testKeys, prand.GetHash().Bytes())
		}
		for _, key := range testKeys {
			err = chainDb.Put(key, common.FromHex("0xdeadbeef"))
			Require(t, err)
		}
		for _, key := range testKeys {
			if has, _ := chainDb.Has(key); !has {
				Fatal(t, "internal test error - failed to check existence of test key")
			}
		}

		initConfig := conf.InitConfigDefault
		initConfig.Prune = "full"
		coreCacheConfig := gethexec.DefaultCacheConfigFor(stack, &builder.execConfig.Caching)
		err = pruning.PruneChainDb(ctx, chainDb, stack, &initConfig, coreCacheConfig, builder.L1.Client, *builder.L2.ConsensusNode.DeployInfo, false)
		Require(t, err)

		for _, key := range testKeys {
			if has, _ := chainDb.Has(key); has {
				Fatal(t, "test key hasn't been pruned as expected")
			}
		}

		chainDbEntriesAfterPruning := countStateEntries(chainDb)
		t.Log("db entries pre-pruning:", chainDbEntriesBeforePruning)
		t.Log("db entries post-pruning:", chainDbEntriesAfterPruning)

		if chainDbEntriesAfterPruning >= chainDbEntriesBeforePruning {
			Fatal(t, "The db doesn't have less entries after pruning then before. Before:", chainDbEntriesBeforePruning, "After:", chainDbEntriesAfterPruning)
		}
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
	var start uint64
	if currentBlock+1 >= builder.execConfig.Caching.BlockCount {
		start = currentBlock + 1 - builder.execConfig.Caching.BlockCount
	} else {
		start = 0
	}
	for i := start; i <= currentBlock; i++ {
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
