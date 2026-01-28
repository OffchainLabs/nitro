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

func TestPruning(t *testing.T) {
	// TODO test "validator" pruning mode - requires latest confirmed
	for _, mode := range []string{"full", "minimal"} {
		t.Run(fmt.Sprintf("-%s-mode-without-parallel-storage-traversal", mode), func(t *testing.T) { testPruning(t, mode, false) })
		t.Run(fmt.Sprintf("-%s-mode-with-parallel-storage-traversal", mode), func(t *testing.T) { testPruning(t, mode, true) })
	}
}

func testPruning(t *testing.T, mode string, pruneParallelStorageTraversal bool) {
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

	// Cache both validated and finalized block hashes for l2 executionDB to later
	// add to the new executionDB below
	data, err := builder.L2.ExecNode.ExecutionDB.Get(gethexec.ValidatedBlockHashKey)
	Require(t, err)
	expectedValidatedBlockHash := common.BytesToHash(data)
	expectedFinalizedBlockHash := rawdb.ReadFinalizedBlockHash(builder.L2.ExecNode.ExecutionDB)

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

		data, err := executionDB.Get(gethexec.ValidatedBlockHashKey)
		Require(t, err)
		validatedBlockHash := common.BytesToHash(data)
		finalizedBlockHash := rawdb.ReadFinalizedBlockHash(executionDB)

		if validatedBlockHash != expectedValidatedBlockHash {
			t.Fatalf("validatedBlockHash: %s does not match expected ValidatedBlockHash: %s", validatedBlockHash.Hex(), expectedValidatedBlockHash.Hex())
		}

		if finalizedBlockHash != expectedFinalizedBlockHash {
			t.Fatalf("finalizedBlockHash: %s does not match expected finalizedBlockHash: %s", finalizedBlockHash.Hex(), expectedFinalizedBlockHash.Hex())
		}

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

func TestStateAfterPruning(t *testing.T) {
	for _, mode := range []string{"validator", "full", "minimal"} {
		t.Run(fmt.Sprintf("-%s-mode-after_pruning_test", mode), func(t *testing.T) { testStateAfterPruning(t, mode) })
	}
}

func testStateAfterPruning(t *testing.T, mode string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithDatabase(rawdb.DBPebble)
	// PathScheme prunes the state trie by itself, so only HashScheme should be tested
	builder.RequireScheme(t, rawdb.HashScheme)

	builder.nodeConfig.ParentChainReader.UseFinalityData = false
	builder.nodeConfig.BlockValidator.Enable = true
	// Used to avoid l1 timestamp delta error since we're creating several blocks
	builder.execConfig.Sequencer.MaxAcceptableTimestampDelta = time.Hour * 72

	_ = builder.Build(t)
	l2cleanupDone := false
	defer func() {
		if !l2cleanupDone {
			builder.L2.cleanup()
		}
		builder.L1.cleanup()
	}()
	builder.L2Info.GenerateAccount("User2")

	numOfBlocksToGenerate := 5000

	// We need to generate these many blocks because of how pruning procedure
	// works. Such procedure has an offset `minRootDistance` set to `2000`; so,
	// if we want to force both last validated and last finalized blocks to be
	// persisted by pruning procedure we need them to be > 2000 blocks apart.
	generateBlocks(t, ctx, builder, builder.L2, numOfBlocksToGenerate)

	safeMsgIdx := arbutil.MessageIndex(4860)
	safeMsgResult, err := builder.L2.ExecNode.ResultAtMessageIndex(safeMsgIdx).Await(ctx)
	Require(t, err)
	safeFinalityData := arbutil.FinalityData{
		MsgIdx:    safeMsgIdx,
		BlockHash: safeMsgResult.BlockHash,
	}

	finalizedMsgIdx := arbutil.MessageIndex(4850)
	finalizedMsgResult, err := builder.L2.ExecNode.ResultAtMessageIndex(finalizedMsgIdx).Await(ctx)
	Require(t, err)
	finalizedFinalityData := arbutil.FinalityData{
		MsgIdx:    finalizedMsgIdx,
		BlockHash: finalizedMsgResult.BlockHash,
	}

	validatedMsgIdx := arbutil.MessageIndex(2550)
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

	balanceAt1, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(1))
	Require(t, err)
	balanceAt50, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(50))
	Require(t, err)
	// #nosec G115
	balanceAtLastBlock, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(int64(lastBlock)))
	Require(t, err)

	if balanceAtLastBlock.Cmp(balanceAt50) < 0 || balanceAt50.Cmp(balanceAt1) < 0 {
		t.Fatal("Balances for User2 are expected to monotonically increase")
	}

	expectedFinalizedBlockHash := rawdb.ReadFinalizedBlockHash(builder.L2.ExecNode.ExecutionDB)

	finalizedBlock, err := builder.L2.Client.BlockByHash(ctx, expectedFinalizedBlockHash)

	if lastBlock < finalizedBlock.Number().Uint64() {
		t.Fatalf("lastBlock: %d should have been greater than finalized block: %d", lastBlock, finalizedBlock.Number().Uint64())
	}

	// Since we're running a regular node (without archival mode), we manually commit
	// some of the blocks to test if prunning procedure indeed deletes such blocks specified
	// with the exception of last validate and finalized blocks (if in "validator" mode)
	bc := builder.L2.ExecNode.Backend.ArbInterface().BlockChain()
	triedb := bc.StateCache().TrieDB()

	for i := 2000; i < numOfBlocksToGenerate; i++ {
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

		// No need to check validated and finality data since we do that on the test above

		initConfig := conf.InitConfigDefault
		initConfig.Prune = mode

		coreCacheConfig := gethexec.DefaultCacheConfigFor(&builder.execConfig.Caching)
		persistentConfig := conf.PersistentConfigDefault
		err = pruning.PruneExecutionDB(ctx, executionDB, stack, &initConfig, coreCacheConfig, &persistentConfig, builder.L1.Client, *builder.L2.ConsensusNode.DeployInfo, false, false)
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
	builder.execConfig.Caching.Archive = false
	testClientL2, cleanup := builder.Build2ndNode(t, &SecondNodeParams{stackConfig: builder.l2StackConfig})
	defer cleanup()

	finalizedBlockHash := rawdb.ReadFinalizedBlockHash(testClientL2.ExecNode.ExecutionDB)

	if finalizedBlockHash != expectedFinalizedBlockHash {
		t.Fatalf("Expected finalizedBlockHash: %s does not match finalizedBlockHash after l2 restart: %s", expectedFinalizedBlockHash.Hex(), finalizedBlockHash.Hex())
	}

	// We make sure that genesis block can be queried since it's always added as an important
	// root so it's trie node should have been deleted
	_, err = testClientL2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(0))
	Require(t, err)

	// Make sure we can't get balance for User2 for the blocks that's been pruned which should be
	// all blocks between [1, 5000) with the exception of last validated and last finalized blocks
	for i := 1; i < numOfBlocksToGenerate; i++ {
		if mode == "validator" && (arbutil.MessageIndex(i) == validatedMsgIdx || arbutil.MessageIndex(i) == finalizedMsgIdx) {
			continue
		}
		_, err = testClientL2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(int64(i)))
		if !strings.Contains(err.Error(), "missing trie node") {
			t.Fatalf("Expected balance retrieval to fail for block %d", i)
		}
	}

	newLastBlock, err := testClientL2.Client.BlockNumber(ctx)
	Require(t, err)

	if newLastBlock < lastBlock {
		t.Fatalf("Expected last block of new node %d to be equal or ahead of original node last block: %d", newLastBlock, lastBlock)
	}

	// #nosec G115
	newLastBlockInt := int64(newLastBlock)

	// Now we check if the blocks that got committed as part of builder.cleanup() got persisted. We do that by calling
	// BalanceAt(...) since that requires stateDB for such block to be presetn to succeed
	_, err = testClientL2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(newLastBlockInt))
	Require(t, err)

	_, err = testClientL2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(newLastBlockInt-1))
	Require(t, err)

	// This is yet another block committed by builder.cleanup() as (HEAD - 127 - 1)
	_, err = testClientL2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(newLastBlockInt-126))
	Require(t, err)

	// We do the same for last validated and last finalized blocks since they should have been added as important roots
	// we only check these in validator mode since all other modes they are also pruned
	if mode == "validator" {
		_, err = testClientL2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(int64(validatedMsgIdx)))
		Require(t, err)

		_, err = testClientL2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(int64(finalizedMsgIdx)))
		Require(t, err)
	}
}
