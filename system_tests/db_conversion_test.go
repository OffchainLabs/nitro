package arbtest

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/offchainlabs/nitro/cmd/dbconv/dbconv"
)

func TestDatabaseConversion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var dataDir string
	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.l2StackConfig.DBEngine = "leveldb"
	builder.l2StackConfig.Name = "testl2"
	builder.execConfig.Caching.Archive = true
	cleanup := builder.Build(t)
	dataDir = builder.dataDir
	cleanupDone := false
	defer func() {
		if !cleanupDone { // TODO we should be able to call cleanup twice, rn it gets stuck then
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
	cleanupDone = true
	cleanup()
	t.Log("stopped first node")

	instanceDir := filepath.Join(dataDir, builder.l2StackConfig.Name)
	err := os.Rename(filepath.Join(instanceDir, "chaindb"), filepath.Join(instanceDir, "chaindb_old"))
	Require(t, err)
	t.Log("converting chaindb...")
	func() {
		oldDBConfig := dbconv.DBConfigDefault
		oldDBConfig.Data = path.Join(instanceDir, "chaindb_old")
		oldDBConfig.DBEngine = "leveldb"
		newDBConfig := dbconv.DBConfigDefault
		newDBConfig.Data = path.Join(instanceDir, "chaindb")
		newDBConfig.DBEngine = "pebble"
		convConfig := dbconv.DefaultDBConvConfig
		convConfig.Src = oldDBConfig
		convConfig.Dst = newDBConfig
		convConfig.Threads = 32
		conv := dbconv.NewDBConverter(&convConfig)
		err := conv.Convert(ctx)
		Require(t, err)
	}()
	t.Log("converting arbitrumdata...")
	err = os.Rename(filepath.Join(instanceDir, "arbitrumdata"), filepath.Join(instanceDir, "arbitrumdata_old"))
	Require(t, err)
	func() {
		oldDBConfig := dbconv.DBConfigDefault
		oldDBConfig.Data = path.Join(instanceDir, "arbitrumdata_old")
		oldDBConfig.DBEngine = "leveldb"
		newDBConfig := dbconv.DBConfigDefault
		newDBConfig.Data = path.Join(instanceDir, "arbitrumdata")
		newDBConfig.DBEngine = "pebble"
		convConfig := dbconv.DefaultDBConvConfig
		convConfig.Src = oldDBConfig
		convConfig.Dst = newDBConfig
		convConfig.Threads = 32
		conv := dbconv.NewDBConverter(&convConfig)
		err := conv.Convert(ctx)
		Require(t, err)
	}()

	builder = NewNodeBuilder(ctx).DefaultConfig(t, true)
	builder.l2StackConfig.Name = "testl2"
	builder.l2StackConfig.DBEngine = "pebble"
	builder.dataDir = dataDir
	cleanup = builder.Build(t)
	defer cleanup()

	builder.L2Info.GenerateAccount("User2")
	t.Log("sending test tx")
	tx := builder.L2Info.PrepareTx("Owner", "User2", builder.L2Info.TransferGas, common.Big1, nil)
	err = builder.L2.Client.SendTransaction(ctx, tx)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)
}
