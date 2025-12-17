// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

func TestRebuildWasmStoreWithDecreasedMaxWasmSize(t *testing.T) {
	databaseEngine := rawdb.DBLeveldb
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithDatabase(databaseEngine)
	})
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)

	storage := deployWasm(t, ctx, auth, l2client, rustFile("storage"))

	zero := common.Hash{}
	val := common.HexToHash("0x121233445566")

	// do an onchain call - store value
	storeTx := l2info.PrepareTxTo("Owner", &storage, l2info.TransferGas, nil, argsForStorageWrite(zero, val))
	Require(t, l2client.SendTransaction(ctx, storeTx))
	_, err = EnsureTxSucceeded(ctx, l2client, storeTx)
	Require(t, err)

	tx, err := arbOwner.SetWasmMaxSize(&auth, 10000)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	testDir := t.TempDir()
	nodeBStack := testhelpers.CreateStackConfigForTest(testDir)
	nodeBStack.DBEngine = databaseEngine
	nodeB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{stackConfig: nodeBStack})

	_, err = EnsureTxSucceeded(ctx, nodeB.Client, storeTx)
	Require(t, err)

	// make sure reading 2nd value succeeds from 2nd node
	loadTx := l2info.PrepareTxTo("Owner", &storage, l2info.TransferGas, nil, argsForStorageRead(zero))
	result, err := arbutil.SendTxAsCall(ctx, nodeB.Client, loadTx, l2info.GetAddress("Owner"), nil, true)
	Require(t, err)
	if common.BytesToHash(result) != val {
		Fatal(t, "got wrong value")
	}

	wasmDb := nodeB.ExecNode.Backend.ArbInterface().BlockChain().StateCache().WasmStore()

	checkWasmStoreContent(t, wasmDb, builder.execConfig.StylusTarget.WasmTargets(), 1)

	// close nodeB
	cleanupB()

	// delete wasm dir of nodeB
	wasmPath := filepath.Join(testDir, nodeBStack.Name, "wasm")
	dirContents, err := os.ReadDir(wasmPath)
	Require(t, err)
	if len(dirContents) == 0 {
		Fatal(t, "not contents found before delete")
	}
	os.RemoveAll(wasmPath)

	// recreate nodeB - using same source dir (wasm deleted)
	nodeB, cleanupB = builder.Build2ndNode(t, &SecondNodeParams{stackConfig: nodeBStack})
	bc := nodeB.ExecNode.Backend.ArbInterface().BlockChain()

	wasmDbAfterDelete := nodeB.ExecNode.Backend.ArbInterface().BlockChain().StateCache().WasmStore()
	storeMapAfterDelete, err := createMapFromDb(wasmDbAfterDelete)
	Require(t, err)
	if len(storeMapAfterDelete) != 0 {
		Fatal(t, "non-empty wasm store after it was previously deleted")
	}

	// Start rebuilding and wait for it to finish
	log.Info("starting rebuilding of wasm store")
	execConfig := builder.execConfig
	Require(t, gethexec.RebuildWasmStore(ctx, wasmDbAfterDelete, nodeB.ExecNode.ChainDB, execConfig.RPC.MaxRecreateStateDepth, &execConfig.StylusTarget, bc, common.Hash{}, bc.CurrentBlock().Hash()))
	wasmDbAfterRebuild := nodeB.ExecNode.Backend.ArbInterface().BlockChain().StateCache().WasmStore()
	checkWasmStoreContent(t, wasmDbAfterRebuild, builder.execConfig.StylusTarget.WasmTargets(), 1)
	cleanupB()
}

func TestExecuteWasmWithDecreasedMaxWasmSizeWasmPresent(t *testing.T) {
	databaseEngine := rawdb.DBLeveldb
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithDatabase(databaseEngine)
	})
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)

	// deploy stylus contract
	file := rustFile("storage")
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	wasm, _ := readWasmFile(t, file)
	auth.GasLimit = 32000000 // skip gas estimation
	program := deployContract(t, ctx, auth, l2client, wasm)
	colors.PrintGrey(name, ": deployed to ", program.Hex())
	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)
	auth.Value = oneEth
	tx, err := arbWasm.ActivateProgram(&auth, program)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	// decrease MaxWasmSize
	auth.GasLimit = 0
	auth.Value = nil
	tx, err = arbOwner.SetWasmMaxSize(&auth, 10000)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	// try to execute wasm, will trigger `getWasmFromContractCode`
	zero := common.Hash{}
	val := common.HexToHash("0x121233445566")
	storeTx := l2info.PrepareTxTo("Owner", &program, l2info.TransferGas, nil, argsForStorageWrite(zero, val))
	Require(t, l2client.SendTransaction(ctx, storeTx))
	_, err = EnsureTxSucceeded(ctx, l2client, storeTx)
	Require(t, err)
}

func TestExecuteWasmWithDecreasedMaxWasmSizeRecoverWasm(t *testing.T) {
	databaseEngine := rawdb.DBLeveldb
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithDatabase(databaseEngine)
	})
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)

	// deploy stylus contract
	file := rustFile("storage")
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	wasm, _ := readWasmFile(t, file)
	auth.GasLimit = 32000000 // skip gas estimation
	program := deployContract(t, ctx, auth, l2client, wasm)
	colors.PrintGrey(name, ": deployed to ", program.Hex())
	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)
	auth.Value = oneEth
	tx, err := arbWasm.ActivateProgram(&auth, program)
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)
	l, err := arbWasm.ParseProgramActivated(*receipt.Logs[0])
	Require(t, err)

	// decrease MaxWasmSize
	auth.GasLimit = 0
	auth.Value = nil
	tx, err = arbOwner.SetWasmMaxSize(&auth, 10000)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	// delete targets
	wasmStore := builder.L2.ExecNode.Backend.ArbInterface().BlockChain().StateCache().WasmStore()
	Require(t, deleteAnyKeysContainingModuleHash(wasmStore, l.ModuleHash))

	// try to execute wasm, will trigger `getWasmFromContractCode`
	zero := common.Hash{}
	val := common.HexToHash("0x121233445566")
	storeTx := l2info.PrepareTxTo("Owner", &program, l2info.TransferGas, nil, argsForStorageWrite(zero, val))
	Require(t, l2client.SendTransaction(ctx, storeTx))
	_, err = EnsureTxSucceeded(ctx, l2client, storeTx)
	Require(t, err)
}

func deleteAnyKeysContainingModuleHash(db ethdb.KeyValueStore, moduleHash common.Hash) error {
	it := db.NewIterator(nil, nil)
	defer it.Release()

	batch := db.NewBatch()
	mh := moduleHash.Bytes()

	for it.Next() {
		k := it.Key()
		if bytes.Contains(k, mh) {
			kk := append([]byte(nil), k...)
			err := batch.Delete(kk)
			if err != nil {
				return err
			}
		}
	}
	if err := it.Error(); err != nil {
		return err
	}
	return batch.Write()
}

func TestCacheProgramWithDecreasedMaxWasmSizeRecoverWasm(t *testing.T) {
	databaseEngine := rawdb.DBLeveldb
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithDatabase(databaseEngine)
	})
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)

	// deploy stylus contract
	file := rustFile("storage")
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	wasm, _ := readWasmFile(t, file)
	auth.GasLimit = 32000000 // skip gas estimation
	program := deployContract(t, ctx, auth, l2client, wasm)
	colors.PrintGrey(name, ": deployed to ", program.Hex())
	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)
	auth.Value = oneEth
	tx, err := arbWasm.ActivateProgram(&auth, program)
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)
	l, err := arbWasm.ParseProgramActivated(*receipt.Logs[0])
	Require(t, err)

	// decrease MaxWasmSize
	auth.GasLimit = 0
	auth.Value = nil
	tx, err = arbOwner.SetWasmMaxSize(&auth, 10000)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	// delete targets
	wasmStore := builder.L2.ExecNode.Backend.ArbInterface().BlockChain().StateCache().WasmStore()
	Require(t, deleteAnyKeysContainingModuleHash(wasmStore, l.ModuleHash))

	arbWasmCache, err := precompilesgen.NewArbWasmCache(types.ArbWasmCacheAddress, builder.L2.Client)
	Require(t, err)
	_, err = arbWasmCache.CacheProgram(&auth, program)
	Require(t, err)
}

func TestDeployingContractBeforeAndAfterDecrease(t *testing.T) {
	databaseEngine := rawdb.DBLeveldb
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithDatabase(databaseEngine)
	})
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)

	// deploy stylus contract
	file := rustFile("storage")
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	wasm, _ := readWasmFile(t, file)
	auth.GasLimit = 32000000 // skip gas estimation
	program := deployContract(t, ctx, auth, l2client, wasm)
	colors.PrintGrey(name, ": deployed to ", program.Hex())
	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)
	auth.Value = oneEth
	tx, err := arbWasm.ActivateProgram(&auth, program)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	zero := common.Hash{}
	val := common.HexToHash("0x121233445566")
	storeTx := l2info.PrepareTxTo("Owner", &program, l2info.TransferGas, nil, argsForStorageWrite(zero, val))
	Require(t, l2client.SendTransaction(ctx, storeTx))
	_, err = EnsureTxSucceeded(ctx, l2client, storeTx)
	Require(t, err)

	// decrease MaxWasmSize
	auth.GasLimit = 0
	auth.Value = nil
	tx, err = arbOwner.SetWasmMaxSize(&auth, 10000)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	// deploying the contract should fail now
	file = rustFile("sdk-storage")
	name = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	wasm, _ = readWasmFile(t, file)
	auth.GasLimit = 32000000 // skip gas estimation
	program = deployContract(t, ctx, auth, l2client, wasm)
	colors.PrintGrey(name, ": deployed to ", program.Hex())
	arbWasm, err = precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)
	auth.Value = oneEth
	tx, err = arbWasm.ActivateProgram(&auth, program)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	require.Error(t, err)
}
