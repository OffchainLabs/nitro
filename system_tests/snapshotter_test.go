package arbtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/offchainlabs/nitro/cmd/conf"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/snapshotter"
)

func TestDatabsaseSnapshotter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.l2StackConfig.Name = "testl2" // set Name to recreate instanceDir later on
	// snapshotter supports only HashScheme
	builder.execConfig.Caching.StateScheme = rawdb.HashScheme
	// for simplicity we use archive node as snapshotter reads state from disk
	builder.execConfig.Caching.Archive = true
	_ = builder.Build(t)
	l2cleanupDone := false
	defer func() {
		if !l2cleanupDone {
			builder.L2.cleanup()
		}
		builder.L1.cleanup()
	}()
	var txs []*types.Transaction
	threadsRunning := 0
	var wg sync.WaitGroup
	for i := 0; i < 127; i++ {
		user := fmt.Sprintf("user-%d", i)
		builder.L2Info.GenerateAccount(user)
		wg.Add(1)
		threadsRunning++
		go func() {
			defer wg.Done()
			for j := 0; j < 3; j++ {
				tx := builder.L2Info.PrepareTx("Owner", user, builder.L2Info.TransferGas, common.Big1, nil)
				txs = append(txs, tx)
				err := builder.L2.Client.SendTransaction(ctx, tx)
				Require(t, err)
			}
		}()
		if threadsRunning > 16 {
			wg.Wait()
		}
	}
	wg.Wait()

	for _, tx := range txs {
		_, err := builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}
	lastBlock, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)
	l2cleanupDone = true
	builder.L2.cleanup()
	t.Log("stopped l2 node")

	snapshotDir := t.TempDir()

	func() {
		stack, err := node.New(builder.l2StackConfig)
		Require(t, err)
		defer stack.Close()
		chainDb, err := stack.OpenDatabaseWithExtraOptions("l2chaindata", 0, 0, "l2chaindata/", false, conf.PersistentConfigDefault.Pebble.ExtraOptions("l2chaindata"))
		Require(t, err)
		defer chainDb.Close()
		coreCacheConfig := gethexec.DefaultCacheConfigFor(stack, &builder.execConfig.Caching)
		bc, err := gethexec.GetBlockChain(chainDb, coreCacheConfig, builder.chainConfig, builder.execConfig.TxLookupLimit)
		Require(t, err)

		snapshotterConfig := snapshotter.DatabaseSnapshotterConfigDefault
		snapshotterConfig.Enable = true
		snapshotterConfig.Threads = 16
		snapshotterConfig.GethExporter.Output.Data = snapshotDir

		trigger := make(chan common.Hash)
		result := make(chan error)
		snapshotter := snapshotter.NewDatabaseSnapshotter(chainDb, bc, &snapshotterConfig, trigger, result)
		snapshotter.Start(ctx)
		trigger <- common.Hash{}
		err = <-result
		Require(t, err)

	}()

	// replace l2chaindata database with snapshot
	instanceDir := filepath.Join(builder.dataDir, builder.l2StackConfig.Name)
	l2ChainDataDir := filepath.Join(instanceDir, "l2chaindata")
	err = os.RemoveAll(l2ChainDataDir)
	Require(t, err)
	err = os.Rename(snapshotDir, l2ChainDataDir)
	Require(t, err)

	// we pass original l2StackConfig to 2nd node to start from the same data dir
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
	if currentBlock != lastBlock {
		Fatal(t, "unexpected current block, want:", lastBlock, " have:", currentBlock)
	}

	bc := testClient.ExecNode.Backend.ArbInterface().BlockChain()
	triedb := bc.StateCache().TrieDB()

	checkBlock := func(number uint64) {
		header := bc.GetHeaderByNumber(number)
		if header == nil {
			Fatal(t, "header not found for block:", number)
		}
		body := bc.GetBody(header.Hash())
		if body == nil {
			Fatal(t, "header not found for block, hash:", header.Hash(), "number:", number)
		}
		receipts := bc.GetReceiptsByHash(header.Hash())
		if receipts == nil {
			Fatal(t, "receipts not found for block, hash:", header.Hash(), "number:", number)
		}
	}
	checkStateExists := func(number uint64) {
		header := bc.GetHeaderByNumber(number)
		if header == nil {
			Fatal(t, "header not found for block:", number)
		}
		_, err := bc.StateAt(header.Root)
		Require(t, err)
		tr, err := trie.New(trie.TrieID(header.Root), triedb)
		Require(t, err)
		accountIt, err := tr.NodeIterator(nil)
		Require(t, err)
		for accountIt.Next(true) {
			if accountIt.Hash() != (common.Hash{}) {
				blob := accountIt.NodeBlob()
				if len(blob) == 0 {
					Fatal(t, "missing trie node blob, path:", fmt.Sprintf("%x", accountIt.Path()), "key:", accountIt.Hash())
				}
			}
			if accountIt.Leaf() {
				keyBytes := accountIt.LeafKey()
				if len(keyBytes) != len(common.Hash{}) {
					Fatal(t, "invalid account leaf key length")
				}
				key := common.BytesToHash(keyBytes)
				var data types.StateAccount
				if err := rlp.DecodeBytes(accountIt.LeafBlob(), &data); err != nil {
					Fatal(t, "failed to decode account data:", err)
				}
				if data.Root != (common.Hash{}) {
					trieID := trie.StorageTrieID(data.Root, key, data.Root)
					storageTr, err := trie.NewStateTrie(trieID, triedb)
					Require(t, err)
					storageIt, err := storageTr.NodeIterator(nil)
					Require(t, err)
					for storageIt.Next(true) {
					}
					Require(t, storageIt.Error())
				}
			}
		}
		Require(t, accountIt.Error())
	}
	checkStateDoesNotExist := func(number uint64) {
		header := bc.GetHeaderByNumber(number)
		if header == nil {
			Fatal(t, "header not found for block:", number)
		}
		_, err := bc.StateAt(header.Root)
		if err == nil {
			Fatal(t, "state shouldn't be found for block:", number)
		}
	}
	// check genesis and head block state
	checkBlock(0)
	checkStateExists(0)
	checkBlock(lastBlock)
	checkStateExists(lastBlock)
	for i := uint64(1); i < lastBlock; i++ {
		checkBlock(i)
		checkStateDoesNotExist(i)
	}

	tx := builder.L2Info.PrepareTx("Owner", "user-0", builder.L2Info.TransferGas, common.Big1, nil)
	err = testClient.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = testClient.EnsureTxSucceeded(tx)
	Require(t, err)
}
