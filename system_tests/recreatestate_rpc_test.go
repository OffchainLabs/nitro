package arbtest

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/nitro/arbnode"
	"github.com/offchainlabs/nitro/arbnode/execution"
	"github.com/offchainlabs/nitro/util"
)

func prepareNodeWithHistory(t *testing.T, ctx context.Context, nodeConfig *arbnode.Config, txCount uint64) (node *arbnode.Node, executionNode *execution.ExecutionNode, l2client *ethclient.Client, cancel func()) {
	t.Helper()
	l2info, node, l2client, _, _, _, l1stack := createTestNodeOnL1WithConfig(t, ctx, true, nodeConfig, nil, nil)
	cancel = func() {
		defer requireClose(t, l1stack)
		defer node.StopAndWait()
	}
	l2info.GenerateAccount("User2")
	var txs []*types.Transaction
	for i := uint64(0); i < txCount; i++ {
		tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, common.Big1, nil)
		txs = append(txs, tx)
		err := l2client.SendTransaction(ctx, tx)
		Require(t, err)
	}
	for _, tx := range txs {
		_, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
	}
	return node, node.Execution, l2client, cancel
}

func fillHeaderCache(t *testing.T, bc *core.BlockChain, from, to uint64) {
	t.Helper()
	for i := from; i <= to; i++ {
		header := bc.GetHeaderByNumber(i)
		if header == nil {
			Fatal(t, "internal test error - failed to get header while trying to fill headerCache, header:", i)
		}
	}
}

func fillBlockCache(t *testing.T, bc *core.BlockChain, from, to uint64) {
	t.Helper()
	for i := from; i <= to; i++ {
		block := bc.GetBlockByNumber(i)
		if block == nil {
			Fatal(t, "internal test error - failed to get block while trying to fill blockCache, block:", i)
		}
	}
}

func removeStatesFromDb(t *testing.T, bc *core.BlockChain, db ethdb.Database, from, to uint64) {
	t.Helper()
	for i := from; i <= to; i++ {
		header := bc.GetHeaderByNumber(i)
		if header == nil {
			Fatal(t, "failed to get last block header")
		}
		hash := header.Root
		err := db.Delete(hash.Bytes())
		Require(t, err)
	}
	for i := from; i <= to; i++ {
		header := bc.GetHeaderByNumber(i)
		_, err := bc.StateAt(header.Root)
		if err == nil {
			Fatal(t, "internal test error - failed to remove state from db")
		}
		expectedErr := &trie.MissingNodeError{}
		if !errors.As(err, &expectedErr) {
			Fatal(t, "internal test error - failed to remove state from db, err: ", err)
		}
	}
}

func TestRecreateStateForRPCNoDepthLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.RPC.MaxRecreateStateDepth = arbitrum.InfiniteMaxRecreateStateDepth
	nodeConfig.Sequencer.MaxBlockSpeed = 0
	nodeConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	nodeConfig.Caching.Archive = true
	// disable caching of states in BlockChain.stateCache
	nodeConfig.Caching.TrieCleanCache = 0
	nodeConfig.Caching.TrieDirtyCache = 0
	nodeConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 0
	nodeConfig.Caching.MaxAmountOfGasToSkipStateSaving = 0
	_, execNode, l2client, cancelNode := prepareNodeWithHistory(t, ctx, nodeConfig, 32)
	defer cancelNode()
	bc := execNode.Backend.ArbInterface().BlockChain()
	db := execNode.Backend.ChainDb()

	lastBlock, err := l2client.BlockNumber(ctx)
	Require(t, err)
	middleBlock := lastBlock / 2

	expectedBalance, err := l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	Require(t, err)

	removeStatesFromDb(t, bc, db, middleBlock, lastBlock)

	balance, err := l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	Require(t, err)
	if balance.Cmp(expectedBalance) != 0 {
		Fatal(t, "unexpected balance result for last block, want: ", expectedBalance, " have: ", balance)
	}

}

func TestRecreateStateForRPCBigEnoughDepthLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	depthGasLimit := int64(256 * util.NormalizeL2GasForL1GasInitial(800_000, params.GWei))
	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.RPC.MaxRecreateStateDepth = depthGasLimit
	nodeConfig.Sequencer.MaxBlockSpeed = 0
	nodeConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	nodeConfig.Caching.Archive = true
	// disable caching of states in BlockChain.stateCache
	nodeConfig.Caching.TrieCleanCache = 0
	nodeConfig.Caching.TrieDirtyCache = 0
	nodeConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 0
	nodeConfig.Caching.MaxAmountOfGasToSkipStateSaving = 0
	_, execNode, l2client, cancelNode := prepareNodeWithHistory(t, ctx, nodeConfig, 32)
	defer cancelNode()
	bc := execNode.Backend.ArbInterface().BlockChain()
	db := execNode.Backend.ChainDb()

	lastBlock, err := l2client.BlockNumber(ctx)
	Require(t, err)
	middleBlock := lastBlock / 2

	expectedBalance, err := l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	Require(t, err)

	removeStatesFromDb(t, bc, db, middleBlock, lastBlock)

	balance, err := l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	Require(t, err)
	if balance.Cmp(expectedBalance) != 0 {
		Fatal(t, "unexpected balance result for last block, want: ", expectedBalance, " have: ", balance)
	}

}

func TestRecreateStateForRPCDepthLimitExceeded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.RPC.MaxRecreateStateDepth = int64(200)
	nodeConfig.Sequencer.MaxBlockSpeed = 0
	nodeConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	nodeConfig.Caching.Archive = true
	// disable caching of states in BlockChain.stateCache
	nodeConfig.Caching.TrieCleanCache = 0
	nodeConfig.Caching.TrieDirtyCache = 0
	nodeConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 0
	nodeConfig.Caching.MaxAmountOfGasToSkipStateSaving = 0
	_, execNode, l2client, cancelNode := prepareNodeWithHistory(t, ctx, nodeConfig, 32)
	defer cancelNode()
	bc := execNode.Backend.ArbInterface().BlockChain()
	db := execNode.Backend.ChainDb()

	lastBlock, err := l2client.BlockNumber(ctx)
	Require(t, err)
	middleBlock := lastBlock / 2

	removeStatesFromDb(t, bc, db, middleBlock, lastBlock)

	_, err = l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	if err == nil {
		Fatal(t, "Didn't fail as expected")
	}
	if err.Error() != arbitrum.ErrDepthLimitExceeded.Error() {
		Fatal(t, "Failed with unexpected error:", err)
	}
}

func TestRecreateStateForRPCMissingBlockParent(t *testing.T) {
	// HeaderChain.headerCache size limit is currently core.headerCacheLimit = 512
	var headerCacheLimit uint64 = 512
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.RPC.MaxRecreateStateDepth = arbitrum.InfiniteMaxRecreateStateDepth
	nodeConfig.Sequencer.MaxBlockSpeed = 0
	nodeConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	nodeConfig.Caching.Archive = true
	// disable caching of states in BlockChain.stateCache
	nodeConfig.Caching.TrieCleanCache = 0
	nodeConfig.Caching.TrieDirtyCache = 0
	nodeConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 0
	nodeConfig.Caching.MaxAmountOfGasToSkipStateSaving = 0
	_, execNode, l2client, cancelNode := prepareNodeWithHistory(t, ctx, nodeConfig, headerCacheLimit+5)
	defer cancelNode()
	bc := execNode.Backend.ArbInterface().BlockChain()
	db := execNode.Backend.ChainDb()

	lastBlock, err := l2client.BlockNumber(ctx)
	Require(t, err)
	if lastBlock < headerCacheLimit+4 {
		Fatal(t, "Internal test error - not enough blocks produced during preparation, want:", headerCacheLimit, "have:", lastBlock)
	}

	removeStatesFromDb(t, bc, db, lastBlock-4, lastBlock)

	headerToRemove := lastBlock - 4
	hash := rawdb.ReadCanonicalHash(db, headerToRemove)
	rawdb.DeleteHeader(db, hash, headerToRemove)

	firstBlock := lastBlock - headerCacheLimit - 5
	fillHeaderCache(t, bc, firstBlock, firstBlock+headerCacheLimit)

	for i := lastBlock; i > lastBlock-3; i-- {
		_, err = l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(i))
		if err == nil {
			hash := rawdb.ReadCanonicalHash(db, i)
			Fatal(t, "Didn't fail to get balance at block:", i, " with hash:", hash, ", lastBlock:", lastBlock)
		}
		if !strings.Contains(err.Error(), "chain doesn't contain parent of block") {
			Fatal(t, "Failed with unexpected error: \"", err, "\", at block:", i, "lastBlock:", lastBlock)
		}
	}
}

func TestRecreateStateForRPCBeyondGenesis(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.RPC.MaxRecreateStateDepth = arbitrum.InfiniteMaxRecreateStateDepth
	nodeConfig.Sequencer.MaxBlockSpeed = 0
	nodeConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	nodeConfig.Caching.Archive = true
	// disable caching of states in BlockChain.stateCache
	nodeConfig.Caching.TrieCleanCache = 0
	nodeConfig.Caching.TrieDirtyCache = 0
	nodeConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 0
	nodeConfig.Caching.MaxAmountOfGasToSkipStateSaving = 0
	_, execNode, l2client, cancelNode := prepareNodeWithHistory(t, ctx, nodeConfig, 32)
	defer cancelNode()
	bc := execNode.Backend.ArbInterface().BlockChain()
	db := execNode.Backend.ChainDb()

	lastBlock, err := l2client.BlockNumber(ctx)
	Require(t, err)

	genesis := bc.Config().ArbitrumChainParams.GenesisBlockNum
	removeStatesFromDb(t, bc, db, genesis, lastBlock)

	_, err = l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	if err == nil {
		hash := rawdb.ReadCanonicalHash(db, lastBlock)
		Fatal(t, "Didn't fail to get balance at block:", lastBlock, " with hash:", hash, ", lastBlock:", lastBlock)
	}
	if !strings.Contains(err.Error(), "moved beyond genesis") {
		Fatal(t, "Failed with unexpected error: \"", err, "\", at block:", lastBlock, "lastBlock:", lastBlock)
	}
}

func TestRecreateStateForRPCBlockNotFoundWhileRecreating(t *testing.T) {
	// BlockChain.blockCache size limit is currently core.blockCacheLimit = 256
	var blockCacheLimit uint64 = 256
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.RPC.MaxRecreateStateDepth = arbitrum.InfiniteMaxRecreateStateDepth
	nodeConfig.Sequencer.MaxBlockSpeed = 0
	nodeConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	nodeConfig.Caching.Archive = true
	// disable caching of states in BlockChain.stateCache
	nodeConfig.Caching.TrieCleanCache = 0
	nodeConfig.Caching.TrieDirtyCache = 0
	nodeConfig.Caching.MaxNumberOfBlocksToSkipStateSaving = 0
	nodeConfig.Caching.MaxAmountOfGasToSkipStateSaving = 0
	_, execNode, l2client, cancelNode := prepareNodeWithHistory(t, ctx, nodeConfig, blockCacheLimit+4)
	defer cancelNode()
	bc := execNode.Backend.ArbInterface().BlockChain()
	db := execNode.Backend.ChainDb()

	lastBlock, err := l2client.BlockNumber(ctx)
	Require(t, err)
	if lastBlock < blockCacheLimit+4 {
		Fatal(t, "Internal test error - not enough blocks produced during preparation, want:", blockCacheLimit, "have:", lastBlock)
	}

	removeStatesFromDb(t, bc, db, lastBlock-4, lastBlock)

	blockBodyToRemove := lastBlock - 1
	hash := rawdb.ReadCanonicalHash(db, blockBodyToRemove)
	rawdb.DeleteBody(db, hash, blockBodyToRemove)

	firstBlock := lastBlock - blockCacheLimit - 4
	fillBlockCache(t, bc, firstBlock, firstBlock+blockCacheLimit)

	_, err = l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	if err == nil {
		hash := rawdb.ReadCanonicalHash(db, lastBlock)
		Fatal(t, "Didn't fail to get balance at block:", lastBlock, " with hash:", hash, ", lastBlock:", lastBlock)
	}
	if !strings.Contains(err.Error(), "block not found while recreating") {
		Fatal(t, "Failed with unexpected error: \"", err, "\", at block:", lastBlock, "lastBlock:", lastBlock)
	}
}

func testSkippingSavingStateAndRecreatingAfterRestart(t *testing.T, cacheConfig *execution.CachingConfig, txCount int) {
	maxRecreateStateDepth := int64(30 * 1000 * 1000)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx1, cancel1 := context.WithCancel(ctx)
	nodeConfig := arbnode.ConfigDefaultL2Test()
	nodeConfig.RPC.MaxRecreateStateDepth = maxRecreateStateDepth
	nodeConfig.Sequencer.MaxBlockSpeed = 0
	nodeConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	nodeConfig.Caching = *cacheConfig

	skipBlocks := nodeConfig.Caching.MaxNumberOfBlocksToSkipStateSaving
	skipGas := nodeConfig.Caching.MaxAmountOfGasToSkipStateSaving

	feedErrChan := make(chan error, 10)
	AddDefaultValNode(t, ctx1, nodeConfig, true)
	l2info, stack, chainDb, arbDb, blockchain := createL2BlockChain(t, nil, t.TempDir(), params.ArbitrumDevTestChainConfig(), &nodeConfig.Caching)

	node, err := arbnode.CreateNode(ctx1, stack, chainDb, arbDb, NewFetcherFromConfig(nodeConfig), blockchain, nil, nil, nil, nil, nil, feedErrChan)
	Require(t, err)
	err = node.TxStreamer.AddFakeInitMessage()
	Require(t, err)
	Require(t, node.Start(ctx1))
	client := ClientForStack(t, stack)

	StartWatchChanErr(t, ctx, feedErrChan, node)
	dataDir := node.Stack.DataDir()

	l2info.GenerateAccount("User2")
	var txs []*types.Transaction
	for i := 0; i < txCount; i++ {
		tx := l2info.PrepareTx("Owner", "User2", l2info.TransferGas, common.Big1, nil)
		txs = append(txs, tx)
		err := client.SendTransaction(ctx, tx)
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, client, tx)
		Require(t, err)
		if have, want := receipt.BlockNumber.Uint64(), uint64(i)+1; have != want {
			Fatal(t, "internal test error - tx got included in unexpected block number, have:", have, "want:", want)
		}
	}
	genesis := uint64(0)
	lastBlock, err := client.BlockNumber(ctx)
	Require(t, err)
	if want := genesis + uint64(txCount); lastBlock < want {
		Fatal(t, "internal test error - not enough blocks produced during preparation, want:", want, "have:", lastBlock)
	}
	expectedBalance, err := client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	Require(t, err)

	node.StopAndWait()
	cancel1()
	t.Log("stopped first node")

	AddDefaultValNode(t, ctx, nodeConfig, true)
	l2info, stack, chainDb, arbDb, blockchain = createL2BlockChain(t, l2info, dataDir, params.ArbitrumDevTestChainConfig(), &nodeConfig.Caching)
	node, err = arbnode.CreateNode(ctx, stack, chainDb, arbDb, NewFetcherFromConfig(nodeConfig), blockchain, nil, node.DeployInfo, nil, nil, nil, feedErrChan)
	Require(t, err)
	Require(t, node.Start(ctx))
	client = ClientForStack(t, stack)
	defer node.StopAndWait()
	bc := node.Execution.Backend.ArbInterface().BlockChain()
	gas := skipGas
	blocks := skipBlocks
	for i := genesis + 1; i <= genesis+uint64(txCount); i++ {
		block := bc.GetBlockByNumber(i)
		if block == nil {
			Fatal(t, "header not found for block number:", i)
			continue
		}
		gas += block.GasUsed()
		blocks++
		_, err := bc.StateAt(block.Root())
		if (skipBlocks == 0 && skipGas == 0) || (skipBlocks != 0 && blocks > skipBlocks) || (skipGas != 0 && gas > skipGas) {
			if err != nil {
				t.Log("blocks:", blocks, "skipBlocks:", skipBlocks, "gas:", gas, "skipGas:", skipGas)
			}
			Require(t, err, "state not found, root:", block.Root(), "blockNumber:", i, "blockHash", block.Hash(), "err:", err)
			gas = 0
			blocks = 0
		} else {
			if err == nil {
				t.Log("blocks:", blocks, "skipBlocks:", skipBlocks, "gas:", gas, "skipGas:", skipGas)
				Fatal(t, "state shouldn't be available, root:", block.Root(), "blockNumber:", i, "blockHash", block.Hash())
			}
			expectedErr := &trie.MissingNodeError{}
			if !errors.As(err, &expectedErr) {
				Fatal(t, "getting state failed with unexpected error, root:", block.Root(), "blockNumber:", i, "blockHash", block.Hash())
			}
		}
	}
	for i := genesis + 1; i <= genesis+uint64(txCount); i += i % 10 {
		_, err = client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(i))
		Require(t, err)
	}

	balance, err := client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	Require(t, err)
	if balance.Cmp(expectedBalance) != 0 {
		Fatal(t, "unexpected balance result for last block, want: ", expectedBalance, " have: ", balance)
	}
}

func TestSkippingSavingStateAndRecreatingAfterRestart(t *testing.T) {
	cacheConfig := execution.DefaultCachingConfig
	cacheConfig.Archive = true
	// disable caching of states in BlockChain.stateCache
	cacheConfig.TrieCleanCache = 0
	cacheConfig.TrieDirtyCache = 0
	// test defaults
	testSkippingSavingStateAndRecreatingAfterRestart(t, &cacheConfig, 512)

	cacheConfig.MaxNumberOfBlocksToSkipStateSaving = 127
	cacheConfig.MaxAmountOfGasToSkipStateSaving = 0
	testSkippingSavingStateAndRecreatingAfterRestart(t, &cacheConfig, 512)

	cacheConfig.MaxNumberOfBlocksToSkipStateSaving = 0
	cacheConfig.MaxAmountOfGasToSkipStateSaving = 15 * 1000 * 1000
	testSkippingSavingStateAndRecreatingAfterRestart(t, &cacheConfig, 512)

	cacheConfig.MaxNumberOfBlocksToSkipStateSaving = 127
	cacheConfig.MaxAmountOfGasToSkipStateSaving = 15 * 1000 * 1000
	testSkippingSavingStateAndRecreatingAfterRestart(t, &cacheConfig, 512)

	// one test block ~ 925000 gas
	testBlockGas := uint64(925000)
	skipBlockValues := []uint64{0, 1, 2, 3, 5, 21, 51, 100, 101}
	var skipGasValues []uint64
	for _, i := range skipBlockValues {
		skipGasValues = append(skipGasValues, i*testBlockGas)
	}
	for _, skipGas := range skipGasValues {
		for _, skipBlocks := range skipBlockValues[:len(skipBlockValues)-2] {
			cacheConfig.MaxAmountOfGasToSkipStateSaving = skipGas
			cacheConfig.MaxNumberOfBlocksToSkipStateSaving = uint32(skipBlocks)
			testSkippingSavingStateAndRecreatingAfterRestart(t, &cacheConfig, 100)
		}
	}
}
