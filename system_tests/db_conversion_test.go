package arbtest

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/nitro/cmd/dbconv/dbconv"
)

func TestDatabaseConversion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.l2StackConfig.DBEngine = "leveldb"
	builder.l2StackConfig.Name = "testl2"
	builder.execConfig.Caching.Archive = true
	_ = builder.Build(t)
	dataDir := builder.dataDir
	l2CleanupDone := false
	defer func() { // TODO we should be able to call cleanup twice, rn it gets stuck then
		if !l2CleanupDone {
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
	l2CleanupDone = true
	builder.L2.cleanup()
	t.Log("stopped first node")

	instanceDir := filepath.Join(dataDir, builder.l2StackConfig.Name)
	for _, dbname := range []string{"l2chaindata", "arbitrumdata", "wasm"} {
		err := os.Rename(filepath.Join(instanceDir, dbname), filepath.Join(instanceDir, fmt.Sprintf("%s_old", dbname)))
		Require(t, err)
		t.Log("converting:", dbname)
		convConfig := dbconv.DefaultDBConvConfig
		convConfig.Src.Data = path.Join(instanceDir, fmt.Sprintf("%s_old", dbname))
		convConfig.Dst.Data = path.Join(instanceDir, dbname)
		conv := dbconv.NewDBConverter(&convConfig)
		err = conv.Convert(ctx)
		Require(t, err)
	}

	builder.l2StackConfig.DBEngine = "pebble"
	testClient, cleanup := builder.Build2ndNode(t, &SecondNodeParams{stackConfig: builder.l2StackConfig})
	defer cleanup()

	t.Log("sending test tx")
	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, common.Big1, nil)
	err := testClient.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = testClient.EnsureTxSucceeded(tx)
	Require(t, err)

	bc := testClient.ExecNode.Backend.ArbInterface().BlockChain()
	current := bc.CurrentBlock()
	if current == nil {
		Fatal(t, "failed to get current block header")
	}
	triedb := bc.StateCache().TrieDB()
	visited := 0
	for i := uint64(0); i <= current.Number.Uint64(); i++ {
		header := bc.GetHeaderByNumber(i)
		_, err := bc.StateAt(header.Root)
		Require(t, err)
		tr, err := trie.New(trie.TrieID(header.Root), triedb)
		Require(t, err)
		it, err := tr.NodeIterator(nil)
		Require(t, err)
		for it.Next(true) {
			visited++
		}
		Require(t, it.Error())
	}
	t.Log("visited nodes:", visited)
}
