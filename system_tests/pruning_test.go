// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md
package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/offchainlabs/nitro/arbutil"
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

func TestPruningDBSizeReduction(t *testing.T) {
	// TODO test "validator" pruning mode - requires latest confirmed
	for _, mode := range []string{"full", "minimal"} {
		t.Run(fmt.Sprintf("-%s-mode-without-parallel-storage-traversal", mode), func(t *testing.T) { runPruningDBSizeReductionTest(t, mode, false) })
		t.Run(fmt.Sprintf("-%s-mode-with-parallel-storage-traversal", mode), func(t *testing.T) { runPruningDBSizeReductionTest(t, mode, true) })
	}
}

func runPruningDBSizeReductionTest(t *testing.T, mode string, pruneParallelStorageTraversal bool) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithDatabase(rawdb.DBPebble)
	// PathScheme prunes the state trie by itself, so only HashScheme should be tested
	builder.RequireScheme(t, rawdb.HashScheme)

	builder.nodeConfig.ParentChainReader.UseFinalityData = true
	builder.nodeConfig.BlockValidator.Enable = true

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
		executionDB, err := stack.OpenDatabaseWithOptions("l2chaindata", node.DatabaseOptions{MetricsNamespace: "l2chaindata/", PebbleExtraOptions: conf.PersistentConfigDefault.Pebble.ExtraOptions("l2chaindata")})
		Require(t, err)
		defer executionDB.Close()
		executionDBEntriesBeforePruning := countStateEntries(executionDB)

		prand := testhelpers.NewPseudoRandomDataSource(t, 1)
		var testKeys [][]byte
		for i := 0; i < 100; i++ {
			// generate test keys with length of hash to emulate legacy state trie nodes
			testKeys = append(testKeys, prand.GetHash().Bytes())
		}
		for _, key := range testKeys {
			err = executionDB.Put(key, common.FromHex("0xdeadbeef"))
			Require(t, err)
		}
		for _, key := range testKeys {
			if has, _ := executionDB.Has(key); !has {
				Fatal(t, "internal test error - failed to check existence of test key")
			}
		}

		initConfig := conf.InitConfigDefault
		initConfig.Prune = mode
		initConfig.PruneParallelStorageTraversal = pruneParallelStorageTraversal
		coreCacheConfig := gethexec.DefaultCacheConfigFor(&builder.execConfig.Caching)
		persistentConfig := conf.PersistentConfigDefault
		err = pruning.PruneExecutionDB(ctx, executionDB, stack, &initConfig, coreCacheConfig, &persistentConfig, builder.L1.Client, *builder.L2.ConsensusNode.DeployInfo, false, false)
		Require(t, err)

		for _, key := range testKeys {
			if has, _ := executionDB.Has(key); has {
				Fatal(t, "test key hasn't been pruned as expected")
			}
		}

		executionDBEntriesAfterPruning := countStateEntries(executionDB)
		t.Log("db entries pre-pruning:", executionDBEntriesBeforePruning)
		t.Log("db entries post-pruning:", executionDBEntriesAfterPruning)

		if executionDBEntriesAfterPruning >= executionDBEntriesBeforePruning {
			Fatal(t, "The db doesn't have less entries after pruning then before. Before:", executionDBEntriesBeforePruning, "After:", executionDBEntriesAfterPruning)
		}
	}()

	testClient, cleanup := builder.Build2ndNode(t, &SecondNodeParams{stackConfig: builder.l2StackConfig})
	defer cleanup()

	currentBlock := waitForChainToCatchUp(t, ctx, testClient, lastBlock)

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

func TestPruningStateAvailabilityValidator(t *testing.T) {
	runPruningStateAvailabilityTest(t, "validator")
}

func TestPruningStateAvailabilityMinimal(t *testing.T) {
	runPruningStateAvailabilityTest(t, "minimal")
}

func TestPruningStateAvailabilityFull(t *testing.T) {
	runPruningStateAvailabilityTest(t, "full")
}

func runPruningStateAvailabilityTest(t *testing.T, mode string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithDatabase(rawdb.DBPebble)
	// PathScheme prunes the state trie by itself, so only HashScheme should be tested
	builder.RequireScheme(t, rawdb.HashScheme)

	builder.nodeConfig.ParentChainReader.UseFinalityData = false
	builder.nodeConfig.BlockValidator.Enable = true
	// How many blocks to keep in recent block state tries after node restart
	blocksToKeepAfterRestart := uint64(48)
	builder.execConfig.Caching.BlockCount = blocksToKeepAfterRestart
	builder.DontParalellise()

	_ = builder.Build(t)
	l2cleanupDone := false
	defer func() {
		if !l2cleanupDone {
			builder.L2.cleanup()
		}
		builder.L1.cleanup()
	}()
	builder.L2Info.GenerateAccount("User2")

	numOfBlocksToGenerate := 300

	generateBlocks(t, ctx, builder, builder.L2, numOfBlocksToGenerate)

	safeMsgIdx := arbutil.MessageIndex(260)
	safeMsgResult, err := builder.L2.ExecNode.ResultAtMessageIndex(safeMsgIdx).Await(ctx)
	Require(t, err)
	safeFinalityData := arbutil.FinalityData{
		MsgIdx:    safeMsgIdx,
		BlockHash: safeMsgResult.BlockHash,
	}

	finalizedMsgIdx := arbutil.MessageIndex(250)
	finalizedMsgResult, err := builder.L2.ExecNode.ResultAtMessageIndex(finalizedMsgIdx).Await(ctx)
	Require(t, err)
	finalizedFinalityData := arbutil.FinalityData{
		MsgIdx:    finalizedMsgIdx,
		BlockHash: finalizedMsgResult.BlockHash,
	}

	validatedMsgIdx := arbutil.MessageIndex(140)
	validatedMsgResult, err := builder.L2.ExecNode.ResultAtMessageIndex(validatedMsgIdx).Await(ctx)
	Require(t, err)
	validatedFinalityData := arbutil.FinalityData{
		MsgIdx:    validatedMsgIdx,
		BlockHash: validatedMsgResult.BlockHash,
	}

	err = builder.L2.ExecNode.SyncMonitor.SetFinalityData(builder.L2.ExecNode.ExecutionDB, &safeFinalityData, &finalizedFinalityData, &validatedFinalityData)
	Require(t, err)

	lastBlock, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)

	expectedFinalizedBlockHash := rawdb.ReadFinalizedBlockHash(builder.L2.ExecNode.ExecutionDB)

	finalizedBlock, err := builder.L2.Client.BlockByHash(ctx, expectedFinalizedBlockHash)
	Require(t, err)

	if lastBlock < finalizedBlock.Number().Uint64() {
		t.Fatalf("lastBlock: %d should have been greater than finalized block: %d", lastBlock, finalizedBlock.Number().Uint64())
	}

	// Since we're running a regular node (without archival mode), we manually commit
	// some of the trie nodes related to the block to test if prunning procedure indeed
	// deletes such blocks specified with the exception of last validate and finalized
	// blocks (if in "validator" mode)
	bc := builder.L2.ExecNode.Backend.ArbInterface().BlockChain()
	triedb := bc.StateCache().TrieDB()

	for i := 1; i < numOfBlocksToGenerate; i++ {
		// #nosec G115
		header := bc.GetHeaderByNumber(uint64(i))
		err = triedb.Commit(header.Root, false)
		Require(t, err)
	}

	l2cleanupDone = true
	builder.L2.cleanup()
	t.Log("stopped l2 node")

	func() {
		stack, err := node.New(builder.l2StackConfig)
		Require(t, err)
		defer stack.Close()
		executionDB, err := stack.OpenDatabaseWithOptions("l2chaindata", node.DatabaseOptions{MetricsNamespace: "l2chaindata/", PebbleExtraOptions: conf.PersistentConfigDefault.Pebble.ExtraOptions("l2chaindata")})
		Require(t, err)
		defer executionDB.Close()
		executionDBEntriesBeforePruning := countStateEntries(executionDB)

		// No need to check validated and finality data since we do that on TestPruningDBSizeReduction test

		initConfig := conf.InitConfigDefault
		initConfig.Prune = mode

		coreCacheConfig := gethexec.DefaultCacheConfigFor(&builder.execConfig.Caching)
		persistentConfig := conf.PersistentConfigDefault
		err = pruning.PruneExecutionDBWithDistance(ctx, executionDB, stack, &initConfig, coreCacheConfig, &persistentConfig, builder.L1.Client, *builder.L2.ConsensusNode.DeployInfo, false, false, 100)
		Require(t, err)

		executionDBEntriesAfterPruning := countStateEntries(executionDB)
		t.Log("db entries pre-pruning:", executionDBEntriesBeforePruning)
		t.Log("db entries post-pruning:", executionDBEntriesAfterPruning)

		if executionDBEntriesAfterPruning >= executionDBEntriesBeforePruning {
			Fatal(t, "The db doesn't have less entries after pruning then before. Before:", executionDBEntriesBeforePruning, "After:", executionDBEntriesAfterPruning)
		}
	}()

	// We could have restarted the same node, but spinning up a node with the same
	// executionDB simulates the desired scenario
	testClientL2, cleanup := builder.Build2ndNode(t, &SecondNodeParams{stackConfig: builder.l2StackConfig})
	defer cleanup()

	// We make sure that genesis block can be queried since it's always added as an important
	// root so it's trie node should not have been deleted
	_, err = testClientL2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(0))
	Require(t, err)

	// There's a separate mechanism that keeps recent block state tries in memory/database regardless
	// of whether they're marked as important roots. This is controlled by the BlockCount configuration
	// parameter (also known as TriesInMemory in the underlying blockchain config) The default value
	// for BlockCount is 128 but in this test we set to 48 (blocksToKeepAfterRestart)

	newLastBlock := waitForChainToCatchUp(t, ctx, testClientL2, lastBlock)

	// #nosec G115
	balanceShouldntExistUntilBlock := int64(newLastBlock) - int64(blocksToKeepAfterRestart) + 1
	// #nosec G115
	for i := int64(1); i < int64(newLastBlock); i++ {
		// Create a safety buffer (+/- 2 blocks) around the expected prune point.
		// Due to synchronization latency, the second node's state may vary slightly,
		// making the exact availability of these boundary blocks non-deterministic.
		if i >= balanceShouldntExistUntilBlock-2 && i <= balanceShouldntExistUntilBlock+2 {
			continue
		} else if i < balanceShouldntExistUntilBlock {
			// Make sure we can't get balance for User2 for the blocks that's been pruned which should be
			// all blocks between [1, checkUntilBlock) with the exception of last validated and last finalized blocks
			if arbutil.MessageIndex(i) == validatedMsgIdx || arbutil.MessageIndex(i) == finalizedMsgIdx {
				continue
			}
			_, err = testClientL2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(int64(i)))
			if !strings.Contains(err.Error(), "missing trie node") {
				t.Fatalf("Expected balance retrieval to fail for block %d", i)
			}
		} else {
			_, err = testClientL2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(i))
			Require(t, err)
		}
	}

	// We do the same for last validated and last finalized blocks since they should have been added as important roots
	// we only check these in validator mode since all other modes they are also pruned
	if mode == "validator" {
		_, err = testClientL2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(int64(validatedMsgIdx)))
		Require(t, err)
	}

	if mode == "validator" || mode == "full" {
		_, err = testClientL2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(int64(finalizedMsgIdx)))
		Require(t, err)
	}
}

func waitForChainToCatchUp(t *testing.T, ctx context.Context, testClient *TestClient, lastBlock uint64) uint64 {
	currentBlock := uint64(0)
	var err error
	// wait for the chain to catch up
	for currentBlock < lastBlock {
		currentBlock, err = testClient.Client.BlockNumber(ctx)
		Require(t, err)
		time.Sleep(20 * time.Millisecond)
	}

	return currentBlock
}
