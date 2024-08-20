package arbtest

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/offchainlabs/nitro/cmd/dbconv/dbconv"
	"github.com/offchainlabs/nitro/util/arbmath"
)

func TestDatabaseConversion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.l2StackConfig.DBEngine = "leveldb"
	builder.l2StackConfig.Name = "testl2"
	// currently only HashScheme supports archive mode
	if builder.execConfig.Caching.StateScheme == rawdb.HashScheme {
		builder.execConfig.Caching.Archive = true
	}
	cleanup := builder.Build(t)
	dataDir := builder.dataDir
	cleanupDone := false
	defer func() { // TODO we should be able to call cleanup twice, rn it gets stuck then
		if !cleanupDone {
			cleanup()
		}
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
	block, err := builder.L2.Client.BlockByNumber(ctx, nil)
	Require(t, err)
	user2Balance := builder.L2.GetBalance(t, builder.L2Info.GetAddress("User2"))
	ownerBalance := builder.L2.GetBalance(t, builder.L2Info.GetAddress("Owner"))

	cleanup()
	cleanupDone = true
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
	builder.nodeConfig.ParentChainReader.Enable = false
	builder.withL1 = false
	builder.L2.cleanup = func() {}
	builder.RestartL2Node(t)
	t.Log("restarted the node")

	blockAfterRestart, err := builder.L2.Client.BlockByNumber(ctx, nil)
	Require(t, err)
	user2BalanceAfterRestart := builder.L2.GetBalance(t, builder.L2Info.GetAddress("User2"))
	ownerBalanceAfterRestart := builder.L2.GetBalance(t, builder.L2Info.GetAddress("Owner"))
	if block.Hash() != blockAfterRestart.Hash() {
		t.Fatal("block hash mismatch")
	}
	if !arbmath.BigEquals(user2Balance, user2BalanceAfterRestart) {
		t.Fatal("unexpected User2 balance, have:", user2BalanceAfterRestart, "want:", user2Balance)
	}
	if !arbmath.BigEquals(ownerBalance, ownerBalanceAfterRestart) {
		t.Fatal("unexpected Owner balance, have:", ownerBalanceAfterRestart, "want:", ownerBalance)
	}

	bc := builder.L2.ExecNode.Backend.ArbInterface().BlockChain()
	current := bc.CurrentBlock()
	if current == nil {
		Fatal(t, "failed to get current block header")
	}
	triedb := bc.StateCache().TrieDB()
	visited := 0
	i := uint64(0)
	// don't query historical blocks when PathSchem is used
	if builder.execConfig.Caching.StateScheme == rawdb.PathScheme {
		i = current.Number.Uint64()
	}
	for ; i <= current.Number.Uint64(); i++ {
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

	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, common.Big1, nil)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

}
