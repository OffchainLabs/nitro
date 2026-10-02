// Copyright 2023-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package rpc

import (
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/arbitrum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/system_tests/v2/systest"
	"github.com/offchainlabs/nitro/util"
)

// #nosec G115
var depthGasLimit = int64(256 * util.NormalizeL2GasForL1GasInitial(800_000, params.GWei))

var recreateStateOpts = systest.Compose(
	systest.WithStateScheme(systest.StateSchemeHash),
	systest.WithoutChainOwner(),
	systest.WithExecConfigOverride(func(cfg *gethexec.Config) {
		cfg.Sequencer.MaxBlockSpeed = 5 * time.Millisecond
		cfg.Sequencer.MaxTxDataSize = 150 // 1 test tx ~= 110
		cfg.Caching.Archive = true
		// disable trie/Database.cleans cache, so as states removed from ChainDb won't be cached there
		cfg.Caching.TrieCleanCache = 0
		cfg.Caching.MaxNumberOfBlocksToSkipStateSaving = 0
		cfg.Caching.MaxAmountOfGasToSkipStateSaving = 0
	}),
)

func maxRecreateDepth(depth int64) systest.TestOption {
	return systest.WithExecConfigOverride(func(cfg *gethexec.Config) { cfg.RPC.MaxRecreateStateDepth = depth })
}

var recreateStateTests = []systest.Scenario{
	systest.Test(testRecreateStateForRPC(nil),
		systest.Named("RecreateStateForRPCNoDepthLimit"), recreateStateOpts,
		maxRecreateDepth(arbitrum.InfiniteMaxRecreateStateDepth),
		systest.WithExecConfigOverride(func(cfg *gethexec.Config) { cfg.Caching.SnapshotCache = 0 })), // disable snapshots
	systest.Test(testRecreateStateForRPC(nil),
		systest.Named("RecreateStateForRPCBigEnoughDepthLimit"), recreateStateOpts,
		maxRecreateDepth(depthGasLimit)),
	systest.Test(testRecreateStateForRPC(arbitrum.ErrDepthLimitExceeded),
		systest.Named("RecreateStateForRPCDepthLimitExceeded"), recreateStateOpts,
		maxRecreateDepth(200), systest.WithDBEngine(systest.DBEnginePebble)),
	systest.Test(testRunRecreateStateForRPCMissingBlockParent,
		recreateStateOpts, maxRecreateDepth(arbitrum.InfiniteMaxRecreateStateDepth)),
	systest.Test(testRunRecreateStateForRPCBeyondGenesis,
		recreateStateOpts, maxRecreateDepth(arbitrum.InfiniteMaxRecreateStateDepth)),
	systest.Test(testRunRecreateStateForRPCBlockNotFoundWhileRecreating,
		recreateStateOpts, maxRecreateDepth(arbitrum.InfiniteMaxRecreateStateDepth)),
}

func testRecreateStateForRPC(wantErr error) systest.Scenario {
	return func(env *systest.Env) {
		bc, db := makeSomeTransfers(env, 32)

		lastBlock, err := env.L2.Client.BlockNumber(env.Ctx)
		env.Require(err)
		middleBlock := lastBlock / 2

		user2 := env.L2.Info.GetAddress("User2")
		lastBlockBig := new(big.Int).SetUint64(lastBlock)

		var expectedBalance *big.Int
		if wantErr == nil {
			expectedBalance, err = env.L2.Client.BalanceAt(env.Ctx, user2, lastBlockBig)
			env.Require(err)
		}

		removeStatesFromDb(env, bc, db, middleBlock, lastBlock)

		balance, err := env.L2.Client.BalanceAt(env.Ctx, user2, lastBlockBig)
		if wantErr != nil {
			env.ErrorContains(err, wantErr.Error())
			return
		}
		env.Require(err)
		env.EqualBig(expectedBalance, balance, "unexpected balance result for last block")
	}
}

func testRunRecreateStateForRPCMissingBlockParent(env *systest.Env) {
	// HeaderChain.headerCache size limit is currently core.headerCacheLimit = 512
	var headerCacheLimit uint64 = 512
	bc, db := makeSomeTransfers(env, headerCacheLimit+5)

	lastBlock, err := env.L2.Client.BlockNumber(env.Ctx)
	env.Require(err)
	env.True(lastBlock >= headerCacheLimit+4, "not enough blocks produced during preparation, want: %d, have: %d", headerCacheLimit, lastBlock)
	user2 := env.L2.Info.GetAddress("User2")

	removeStatesFromDb(env, bc, db, lastBlock-4, lastBlock)

	headerToRemove := lastBlock - 4
	hash := rawdb.ReadCanonicalHash(db, headerToRemove)
	rawdb.DeleteHeader(db, hash, headerToRemove)

	firstBlock := lastBlock - headerCacheLimit - 5
	fillHeaderCache(env, bc, firstBlock, firstBlock+headerCacheLimit)

	for i := lastBlock; i > lastBlock-3; i-- {
		_, err = env.L2.Client.BalanceAt(env.Ctx, user2, new(big.Int).SetUint64(i))
		env.NotNil(err, "did not fail to get balance at block: %d with hash: %s, lastBlock: %d", i, rawdb.ReadCanonicalHash(db, i), lastBlock)
		env.ErrorContains(err, "chain doesn't contain parent of block", "unexpected error at block: %d, lastBlock: %d", i, lastBlock)
	}
}

func testRunRecreateStateForRPCBeyondGenesis(env *systest.Env) {
	bc, db := makeSomeTransfers(env, 32)

	lastBlock, err := env.L2.Client.BlockNumber(env.Ctx)
	env.Require(err)
	user2 := env.L2.Info.GetAddress("User2")

	genesis := bc.Config().ArbitrumChainParams.GenesisBlockNum
	removeStatesFromDb(env, bc, db, genesis, lastBlock)

	_, err = env.L2.Client.BalanceAt(env.Ctx, user2, new(big.Int).SetUint64(lastBlock))
	env.NotNil(err, "did not fail to get balance at block: %d with hash: %s", lastBlock, rawdb.ReadCanonicalHash(db, lastBlock))
	env.ErrorContains(err, "moved beyond genesis", "unexpected error at block: %d", lastBlock)
}

func testRunRecreateStateForRPCBlockNotFoundWhileRecreating(env *systest.Env) {
	// BlockChain.blockCache size limit is currently core.blockCacheLimit = 256
	var blockCacheLimit uint64 = 256
	bc, db := makeSomeTransfers(env, blockCacheLimit+4)

	lastBlock, err := env.L2.Client.BlockNumber(env.Ctx)
	env.Require(err)
	env.True(lastBlock >= blockCacheLimit+4, "not enough blocks produced during preparation, want: %d, have: %d", blockCacheLimit, lastBlock)
	user2 := env.L2.Info.GetAddress("User2")

	removeStatesFromDb(env, bc, db, lastBlock-4, lastBlock)

	blockBodyToRemove := lastBlock - 1
	hash := rawdb.ReadCanonicalHash(db, blockBodyToRemove)
	rawdb.DeleteBody(db, hash, blockBodyToRemove)

	firstBlock := lastBlock - blockCacheLimit - 4
	fillBlockCache(env, bc, firstBlock, firstBlock+blockCacheLimit)

	_, err = env.L2.Client.BalanceAt(env.Ctx, user2, new(big.Int).SetUint64(lastBlock))
	env.NotNil(err, "did not fail to get balance at block: %d with hash: %s", lastBlock, rawdb.ReadCanonicalHash(db, lastBlock))
	env.ErrorContains(err, fmt.Sprintf("block #%d not found", blockBodyToRemove), "unexpected error at block: %d", lastBlock)
}

// makeSomeTransfers seeds User2 and mines txCount transfer blocks, returning
// the exec node's blockchain and chain database.
func makeSomeTransfers(env *systest.Env, txCount uint64) (*core.BlockChain, ethdb.Database) {
	env.L2.Info.GenerateAccount("User2")
	txs := make([]*types.Transaction, 0, txCount)
	for range txCount {
		txs = append(txs, env.L2.Info.PrepareTx("Owner", "User2", env.L2.Info.TransferGas, common.Big1, nil))
	}
	env.L2.SendWaitTestTransactions(txs)
	return env.L2.ExecNode.Backend.ArbInterface().BlockChain(), env.L2.ExecNode.Backend.ChainDb()
}

func fillHeaderCache(env *systest.Env, bc *core.BlockChain, from, to uint64) {
	for i := from; i <= to; i++ {
		env.NotNil(bc.GetHeaderByNumber(i), "failed to get header while trying to fill headerCache, header: %d", i)
	}
}

func fillBlockCache(env *systest.Env, bc *core.BlockChain, from, to uint64) {
	for i := from; i <= to; i++ {
		env.NotNil(bc.GetBlockByNumber(i), "failed to get block while trying to fill blockCache, block: %d", i)
	}
}

func removeStatesFromDb(env *systest.Env, bc *core.BlockChain, db ethdb.Database, from, to uint64) {
	for i := from; i <= to; i++ {
		header := bc.GetHeaderByNumber(i)
		env.NotNil(header, "failed to get last block header")
		env.Require(db.Delete(header.Root.Bytes()))
	}
	for i := from; i <= to; i++ {
		header := bc.GetHeaderByNumber(i)
		_, err := bc.StateAt(header)
		env.NotNil(err, "failed to remove state from db")
		expectedErr := &trie.MissingNodeError{}
		env.True(errors.As(err, &expectedErr), "failed to remove state from db, err: %v", err)
	}
}
