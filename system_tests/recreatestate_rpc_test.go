package arbtest

import (
	"context"
	"errors"
	"math/big"
	"strings"
	"testing"
	"time"

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
	"github.com/offchainlabs/nitro/util"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func prepareNodeWithHistory(t *testing.T, ctx context.Context, maxRecreateStateDepth int64, txCount uint64) (node *arbnode.Node, bc *core.BlockChain, db ethdb.Database, l2client *ethclient.Client, l2info info, cancel func()) {
	t.Helper()
	nodeConfig := arbnode.ConfigDefaultL1Test()
	nodeConfig.RPC.MaxRecreateStateDepth = maxRecreateStateDepth
	nodeConfig.Sequencer.MaxBlockSpeed = 0
	nodeConfig.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
	cacheConfig := &core.CacheConfig{
		// Arbitrum Config Options
		TriesInMemory: 128,
		TrieRetention: 30 * time.Minute,

		// disable caching of states in BlockChain.stateCache
		TrieCleanLimit: 0,
		TrieDirtyLimit: 0,

		TrieDirtyDisabled: true,

		TrieTimeLimit: 5 * time.Minute,
		SnapshotLimit: 256,
		SnapshotWait:  true,
	}
	l2info, node, l2client, _, _, _, _, l1stack := createTestNodeOnL1WithConfigImpl(t, ctx, true, nodeConfig, nil, nil, cacheConfig, nil)
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
		testhelpers.RequireImpl(t, err)
	}
	for _, tx := range txs {
		_, err := EnsureTxSucceeded(ctx, l2client, tx)
		testhelpers.RequireImpl(t, err)
	}
	bc = node.Execution.Backend.ArbInterface().BlockChain()
	db = node.Execution.Backend.ChainDb()

	return
}

func fillHeaderCache(t *testing.T, bc *core.BlockChain, from, to uint64) {
	t.Helper()
	for i := from; i <= to; i++ {
		header := bc.GetHeaderByNumber(i)
		if header == nil {
			testhelpers.FailImpl(t, "internal test error - failed to get header while trying to fill headerCache, header:", i)
		}
	}
}

func fillBlockCache(t *testing.T, bc *core.BlockChain, from, to uint64) {
	t.Helper()
	for i := from; i <= to; i++ {
		block := bc.GetBlockByNumber(i)
		if block == nil {
			testhelpers.FailImpl(t, "internal test error - failed to get block while trying to fill blockCache, block:", i)
		}
	}
}

func removeStatesFromDb(t *testing.T, bc *core.BlockChain, db ethdb.Database, from, to uint64) {
	t.Helper()
	for i := from; i <= to; i++ {
		header := bc.GetHeaderByNumber(i)
		if header == nil {
			testhelpers.FailImpl(t, "failed to get last block header")
		}
		hash := header.Root
		err := db.Delete(hash.Bytes())
		testhelpers.RequireImpl(t, err)
	}
	for i := from; i <= to; i++ {
		header := bc.GetHeaderByNumber(i)
		_, err := bc.StateAt(header.Root)
		if err == nil {
			testhelpers.FailImpl(t, "internal test error - failed to remove state from db")
		}
		expectedErr := &trie.MissingNodeError{}
		if !errors.As(err, &expectedErr) {
			testhelpers.FailImpl(t, "internal test error - failed to remove state from db, err: ", err)
		}
	}
}

func TestRecreateStateForRPCNoDepthLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, bc, db, l2client, _, cancelNode := prepareNodeWithHistory(t, ctx, arbitrum.InfiniteMaxRecreateStateDepth, 32)
	defer cancelNode()

	lastBlock, err := l2client.BlockNumber(ctx)
	testhelpers.RequireImpl(t, err)
	middleBlock := lastBlock / 2

	expectedBalance, err := l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	testhelpers.RequireImpl(t, err)

	removeStatesFromDb(t, bc, db, middleBlock, lastBlock)

	balance, err := l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	testhelpers.RequireImpl(t, err)
	if balance.Cmp(expectedBalance) != 0 {
		testhelpers.FailImpl(t, "unexpected balance result for last block, want: ", expectedBalance, " have: ", balance)
	}

}

func TestRecreateStateForRPCBigEnoughDepthLimit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	depthGasLimit := int64(256 * util.NormalizeL2GasForL1GasInitial(800_000, params.GWei))
	_, bc, db, l2client, _, cancelNode := prepareNodeWithHistory(t, ctx, depthGasLimit, 32)
	defer cancelNode()

	lastBlock, err := l2client.BlockNumber(ctx)
	testhelpers.RequireImpl(t, err)
	middleBlock := lastBlock / 2

	expectedBalance, err := l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	testhelpers.RequireImpl(t, err)

	removeStatesFromDb(t, bc, db, middleBlock, lastBlock)

	balance, err := l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	testhelpers.RequireImpl(t, err)
	if balance.Cmp(expectedBalance) != 0 {
		testhelpers.FailImpl(t, "unexpected balance result for last block, want: ", expectedBalance, " have: ", balance)
	}

}

func TestRecreateStateForRPCDepthLimitExceeded(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	depthGasLimit := int64(200)
	_, bc, db, l2client, _, cancelNode := prepareNodeWithHistory(t, ctx, depthGasLimit, 32)
	defer cancelNode()

	lastBlock, err := l2client.BlockNumber(ctx)
	testhelpers.RequireImpl(t, err)
	middleBlock := lastBlock / 2

	removeStatesFromDb(t, bc, db, middleBlock, lastBlock)

	_, err = l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	if err == nil {
		testhelpers.FailImpl(t, "Didn't fail as expected")
	}
	if err.Error() != arbitrum.ErrDepthLimitExceeded.Error() {
		testhelpers.FailImpl(t, "Failed with unexpected error:", err)
	}
}

func TestRecreateStateForRPCMissingBlockParent(t *testing.T) {
	// HeaderChain.headerCache size limit is currently core.headerCacheLimit = 512
	var headerCacheLimit uint64 = 512
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, bc, db, l2client, _, cancelNode := prepareNodeWithHistory(t, ctx, arbitrum.InfiniteMaxRecreateStateDepth, headerCacheLimit+5)
	defer cancelNode()

	lastBlock, err := l2client.BlockNumber(ctx)
	testhelpers.RequireImpl(t, err)
	if lastBlock < headerCacheLimit+4 {
		testhelpers.FailImpl(t, "Internal test error - not enough blocks produced during preparation, want:", headerCacheLimit, "have:", lastBlock)
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
			testhelpers.FailImpl(t, "Didn't fail to get balance at block:", i, " with hash:", hash, ", lastBlock:", lastBlock)
		}
		if !strings.Contains(err.Error(), "chain doesn't contain parent of block") {
			testhelpers.FailImpl(t, "Failed with unexpected error: \"", err, "\", at block:", i, "lastBlock:", lastBlock)
		}
	}
}

func TestRecreateStateForRPCBeyondGenesis(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, bc, db, l2client, _, cancelNode := prepareNodeWithHistory(t, ctx, arbitrum.InfiniteMaxRecreateStateDepth, 32)
	defer cancelNode()

	lastBlock, err := l2client.BlockNumber(ctx)
	testhelpers.RequireImpl(t, err)

	genesis := bc.Config().ArbitrumChainParams.GenesisBlockNum
	removeStatesFromDb(t, bc, db, genesis, lastBlock)

	_, err = l2client.BalanceAt(ctx, GetTestAddressForAccountName(t, "User2"), new(big.Int).SetUint64(lastBlock))
	if err == nil {
		hash := rawdb.ReadCanonicalHash(db, lastBlock)
		testhelpers.FailImpl(t, "Didn't fail to get balance at block:", lastBlock, " with hash:", hash, ", lastBlock:", lastBlock)
	}
	if !strings.Contains(err.Error(), "moved beyond genesis") {
		testhelpers.FailImpl(t, "Failed with unexpected error: \"", err, "\", at block:", lastBlock, "lastBlock:", lastBlock)
	}
}

func TestRecreateStateForRPCBlockNotFoundWhileRecreating(t *testing.T) {
	// BlockChain.blockCache size limit is currently core.blockCacheLimit = 256
	var blockCacheLimit uint64 = 256
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, bc, db, l2client, _, cancelNode := prepareNodeWithHistory(t, ctx, arbitrum.InfiniteMaxRecreateStateDepth, blockCacheLimit+4)
	defer cancelNode()

	lastBlock, err := l2client.BlockNumber(ctx)
	testhelpers.RequireImpl(t, err)
	if lastBlock < blockCacheLimit+4 {
		testhelpers.FailImpl(t, "Internal test error - not enough blocks produced during preparation, want:", blockCacheLimit, "have:", lastBlock)
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
		testhelpers.FailImpl(t, "Didn't fail to get balance at block:", lastBlock, " with hash:", hash, ", lastBlock:", lastBlock)
	}
	if !strings.Contains(err.Error(), "block not found while recreating") {
		testhelpers.FailImpl(t, "Failed with unexpected error: \"", err, "\", at block:", lastBlock, "lastBlock:", lastBlock)
	}
}
