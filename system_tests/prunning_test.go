package arbtest

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/cmd/prunning"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func countDbEntries(db ethdb.Iteratee) int {
	entries := 0
	it := db.NewIterator(nil, nil)
	for it.Next() {
		entries++
	}
	it.Release()
	return entries
}

func TestPrunning(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var dataDir string

	func() {
		builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
		_ = builder.Build(t)
		dataDir = builder.dataDir
		l2cleanupDone := false
		defer func() {
			if !l2cleanupDone {
				builder.L2.cleanup()
			}
			builder.L1.cleanup()
		}()
		builder.L2Info.GenerateAccount("User2")
		var txs []*types.Transaction
		for i := uint64(0); i < 1000; i++ {
			tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, common.Big1, nil)
			txs = append(txs, tx)
			err := builder.L2.Client.SendTransaction(ctx, tx)
			Require(t, err)
		}
		for _, tx := range txs {
			_, err := builder.L2.EnsureTxSucceeded(tx)
			Require(t, err)
		}
		l2cleanupDone = true
		builder.L2.cleanup()
		t.Log("stopped l2 node")

		stack, err := node.New(builder.l2StackConfig)
		Require(t, err)
		defer stack.Close()
		chainDb, err := stack.OpenDatabase("chaindb", 0, 0, "", false)
		Require(t, err)
		defer chainDb.Close()
		entriesBeforePrunning := countDbEntries(chainDb)

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
		err = prunning.PruneChainDb(ctx, chainDb, stack, &initConfig, coreCacheConfig, builder.L1.Client, *builder.L2.ConsensusNode.DeployInfo, false)
		Require(t, err)

		for _, key := range testKeys {
			if has, _ := chainDb.Has(key); has {
				Fatal(t, "test key hasn't been prunned as expected")
			}
		}

		entriesAfterPrunning := countDbEntries(chainDb)
		t.Log("db entries pre-prunning:", entriesBeforePrunning)
		t.Log("db entries post-prunning:", entriesAfterPrunning)

		if entriesAfterPrunning >= entriesBeforePrunning {
			Fatal(t, "The db doesn't have less entires after prunning then before. Before:", entriesBeforePrunning, "After:", entriesAfterPrunning)
		}
	}()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.dataDir = dataDir
	cancel = builder.Build(t)
	defer cancel()

	builder.L2Info.GenerateAccount("User2")
	var txs []*types.Transaction
	for i := uint64(0); i < 10; i++ {
		tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, common.Big1, nil)
		txs = append(txs, tx)
		err := builder.L2.Client.SendTransaction(ctx, tx)
		Require(t, err)
	}
	for _, tx := range txs {
		_, err := builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}
}
