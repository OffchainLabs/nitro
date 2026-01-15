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
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true).WithDatabase(rawdb.DBPebble)
	// PathScheme prunes the state trie by itself, so only HashScheme should be tested
	builder.RequireScheme(t, rawdb.HashScheme)

	builder.nodeConfig.ParentChainReader.UseFinalityData = false
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
	generateBlocks(t, ctx, builder, builder.L2, 200)

	safeMsgIdx := arbutil.MessageIndex(14)
	safeMsgResult, err := builder.L2.ExecNode.ResultAtMessageIndex(safeMsgIdx).Await(ctx)
	Require(t, err)
	safeFinalityData := arbutil.FinalityData{
		MsgIdx:    safeMsgIdx,
		BlockHash: safeMsgResult.BlockHash,
	}

	finalizedMsgIdx := arbutil.MessageIndex(9)
	finalizedMsgResult, err := builder.L2.ExecNode.ResultAtMessageIndex(finalizedMsgIdx).Await(ctx)
	Require(t, err)
	finalizedFinalityData := arbutil.FinalityData{
		MsgIdx:    finalizedMsgIdx,
		BlockHash: finalizedMsgResult.BlockHash,
	}

	validatedMsgIdx := arbutil.MessageIndex(6)
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
	balanceAtLastBlock, err := builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(int64(lastBlock)))
	Require(t, err)

	if balanceAtLastBlock.Cmp(balanceAt50) < 0 || balanceAt50.Cmp(balanceAt1) < 0 {
		t.Fatal("Balances for User2 are expected to monotonically increase")
	}

	t.Logf("balanceAt1: %d, balanceAt50: %d, balanceAtLastBlock: %d", balanceAt1, balanceAt50, balanceAtLastBlock)

	expectedFinalizedBlockHash := rawdb.ReadFinalizedBlockHash(builder.L2.ExecNode.ExecutionDB)

	finalizedBlock, err := builder.L2.Client.BlockByHash(ctx, expectedFinalizedBlockHash)
	t.Logf("Finalized block is %d with hash: %s", finalizedBlock.Number().Uint64(), finalizedBlock.Hash().Hex())

	if lastBlock < finalizedBlock.Number().Uint64() {
		t.Fatalf("lastBlock: %d should have been greater than finalized block: %d", lastBlock, finalizedBlock.Number().Uint64())
	}

	lastBlockBeforeCleanup, err := builder.L2.Client.BlockNumber(ctx)
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

		// No need to check validated and finality data since we do that on the test above

		initConfig := conf.InitConfigDefault
		initConfig.Prune = "validator"
		// initConfig.Prune = "minimal"

		// initConfig.PruneParallelStorageTraversal = pruneParallelStorageTraversal
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

	builder.l2StackConfig.DBEngine = "pebble"
	builder.nodeConfig.ParentChainReader.Enable = false
	builder.withL1 = false
	builder.L2.cleanup = func() {}
	builder.RestartL2Node(t)
	t.Log("restarted the node")

	finalizedBlockHash := rawdb.ReadFinalizedBlockHash(builder.L2.ExecNode.ExecutionDB)

	if finalizedBlockHash != expectedFinalizedBlockHash {
		t.Fatalf("Expected finalizedBlockHash: %s does not match finalizedBlockHash after l2 restart: %s", expectedFinalizedBlockHash.Hex(), finalizedBlockHash.Hex())
	}

	_, err = builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(1))
	if !strings.Contains(err.Error(), "missing trie node") {
		t.Fatal("Expected balance retrieval to fail for block 1")
	}
	// Require(t, err)
	_, err = builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(50))
	if !strings.Contains(err.Error(), "missing trie node") {
		t.Fatal("Expected balance retrieval to fail for block 50")
	}

	for i := 1; i < 100; i++ {
		missingBlock := rawdb.ReadCanonicalHash(builder.L2.ExecNode.ExecutionDB, uint64(i))
		t.Logf("i: %d, missingBlock: %s", i, missingBlock.Hex())
	}

	newLastBlock, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)

	t.Logf("lastBlock: %d, lastBlockBeforeCleanup: %d, newLastBlock: %d", lastBlock, lastBlockBeforeCleanup, newLastBlock)

	_, err = builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(int64(newLastBlock)))
	Require(t, err)

	// This call succeeds on master and fails on this branch!!
	_, err = builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(int64(newLastBlock-1)))
	Require(t, err)

	// This call fails!!
	_, err = builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(int64(newLastBlock-2)))
	Require(t, err)

	// Both calls below fail
	// Make sure we can still retrieve balance at last finalized block number
	// _, err = builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("Owner"), big.NewInt(int64(finalizedBlock.Number().Uint64())))
	// Require(t, err)
	// _, err = builder.L2.Client.BalanceAt(ctx, builder.L2Info.GetAddress("User2"), big.NewInt(int64(finalizedBlock.Number().Uint64())))
	// Require(t, err)
}
