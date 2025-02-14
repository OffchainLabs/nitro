package arbtest

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"

	"github.com/offchainlabs/nitro/snapshotter"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
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
	builder.execConfig.DatabaseSnapshotter.Enable = true
	builder.execConfig.DatabaseSnapshotter.Threads = 32
	snapshotDir := t.TempDir()
	builder.execConfig.DatabaseSnapshotter.GethExporter.Output.Data = snapshotDir
	_ = builder.Build(t)
	l2cleanupDone := false
	defer func() {
		if !l2cleanupDone {
			builder.L2.cleanup()
		}
		builder.L1.cleanup()
	}()
	var txes []*types.Transaction
	var users []string
	for i := 0; i < 16; i++ {
		user := fmt.Sprintf("user-%d", i)
		users = append(users, user)
		builder.L2Info.GenerateAccount(user)
		tx := builder.L2Info.PrepareTx("Owner", user, builder.L2Info.TransferGas, new(big.Int).Lsh(big.NewInt(1), 63), nil)
		Require(t, builder.L2.Client.SendTransaction(ctx, tx))
		txes = append(txes, tx)
	}
	for _, tx := range txes {
		_, err := builder.L2.EnsureTxSucceeded(tx)
		Require(t, err)
	}

	var wg sync.WaitGroup
	for _, user := range users {
		user := user
		wg.Add(1)
		go func() {
			defer wg.Done()
			auth := builder.L2Info.GetDefaultTransactOpts(user, ctx)
			for j := 0; j < 16; j++ {
				var txes []*types.Transaction
				_, tx, mock, err := mocksgen.DeploySdkStorage(&auth, builder.L2.Client)
				Require(t, err)
				txes = append(txes, tx)
				tx, err = mock.Populate(&auth)
				Require(t, err)
				txes = append(txes, tx)
				_, simple := builder.L2.DeploySimple(t, auth)
				tx, err = simple.LogAndIncrement(&auth, common.Big0) // we don't care about expected arg
				Require(t, err)
				txes = append(txes, tx)
				tx, err = simple.StoreDifficulty(&auth)
				Require(t, err)
				txes = append(txes, tx)
				for _, tx := range txes {
					_, err := builder.L2.EnsureTxSucceeded(tx)
					Require(t, err)
				}
			}
		}()
	}
	wg.Wait()

	lastBlock, err := builder.L2.Client.BlockNumber(ctx)
	Require(t, err)

	l2rpc := builder.L2.Stack.Attach()
	err = l2rpc.CallContext(ctx, nil, "snapshotter_snapshot", rpc.LatestBlockNumber)
	Require(t, err)

	var result snapshotter.SnapshotResult
	err = l2rpc.CallContext(ctx, &result, "snapshotter_result", false)
	for err != nil && strings.Contains(err.Error(), "not ready") {
		err = l2rpc.CallContext(ctx, &result, "snapshotter_result", false)
		time.Sleep(10 * time.Millisecond)
	}
	for err != nil && !strings.Contains(err.Error(), "not ready") {
		Fatal(t, "snapshotter_result returned unexpected error, err:", err)
	}
	// TODO validate SnapshotResult
	err = l2rpc.CallContext(ctx, &result, "snapshotter_snapshot", rpc.LatestBlockNumber)
	if err == nil {
		Fatal(t, "should fail when we already have a result")
	}
	if !strings.Contains(err.Error(), "needs rewind") {
		Fatal(t, "failed with unexpected error when output database already exists, err: ", err)
	}
	// rewind
	err = l2rpc.CallContext(ctx, &result, "snapshotter_result", true)
	Require(t, err)

	err = l2rpc.CallContext(ctx, &result, "snapshotter_snapshot", rpc.LatestBlockNumber)
	if err == nil {
		Fatal(t, "should fail when output database already exists")
	}
	if !strings.Contains(err.Error(), "already exists") {
		Fatal(t, "failed with unexpected error when output database already exists, err: ", err)
	}

	l2cleanupDone = true
	builder.L2.cleanup()
	t.Log("stopped l2 node")

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
		if err != nil {
			Fatal(t, "failed to get state for root:", header.Root, "number:", header.Number, "err:", err)
		}
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
						if storageIt.Hash() != (common.Hash{}) {
							if len(storageIt.NodeBlob()) == 0 {
								Fatal(t, "Missing node blob, node hash:", storageIt.Hash())
							}
						}
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
	currentHead := bc.CurrentBlock()
	// check genesis and head block state
	checkBlock(0)
	checkStateExists(0)
	checkBlock(currentHead.Number.Uint64())
	checkStateExists(currentHead.Number.Uint64())
	for i := uint64(1); i < lastBlock; i++ {
		checkBlock(i)
		checkStateDoesNotExist(i)
	}

	// make sure we use big enough GasFeeCap (deploying bunch of contracts in short time may bump up the gas price
	builder.L2Info.GasPrice = new(big.Int).Mul(builder.L2Info.GasPrice, big.NewInt(100))
	tx := builder.L2Info.PrepareTx("Owner", "user-0", builder.L2Info.TransferGas, common.Big1, nil)
	err = testClient.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = testClient.EnsureTxSucceeded(tx)
	Require(t, err)
}
