// Copyright 2025-2026, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE.md

package arbtest

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

// Shared Helpers

type deployConfig struct {
	fragmentCount    uint16
	mutateDict       func(arbcompress.Dictionary) arbcompress.Dictionary
	mutateAddrs      func([]common.Address)
	mutateSize       func(int) int
	expectActivation bool
	expectedErr      string
}

func defaultDeployConfig() deployConfig {
	return deployConfig{
		fragmentCount:    2,
		expectActivation: true,
	}
}

// deployAndActivateFragmentedContract handles the common flow of reading rust files,
// splitting them, deploying fragments, constructing the root, and activating it.
func deployAndActivateFragmentedContract(
	t *testing.T,
	ctx context.Context,
	auth bind.TransactOpts,
	l2client *ethclient.Client,
	cfg deployConfig,
) (common.Address, []byte, *types.Receipt) {
	file := rustFile("storage")
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))

	// Read and fragment
	fragments, sourceWasm, dictType := readFragmentedContractFile(t, file, cfg.fragmentCount)
	require.Len(t, fragments, int(cfg.fragmentCount))

	// Apply dictionary mutations if any
	if cfg.mutateDict != nil {
		dictType = cfg.mutateDict(dictType)
	}

	// Deploy fragments
	auth.GasLimit = 32000000 // skip gas estimation
	addresses := make([]common.Address, 0, len(fragments))
	for i, fragment := range fragments {
		fragmentAddress := deployContract(t, ctx, auth, l2client, fragment)
		colors.PrintGrey(name, ": fragment contract", i, " deployed to ", fragmentAddress.Hex())
		addresses = append(addresses, fragmentAddress)
	}

	// Apply address mutations if any
	if cfg.mutateAddrs != nil {
		cfg.mutateAddrs(addresses)
	}

	// Calculate size
	// #nosec G115
	decompressedSize := uint32(len(sourceWasm))
	if cfg.mutateSize != nil {
		// #nosec G115
		decompressedSize = uint32(cfg.mutateSize(len(sourceWasm)))
	}

	// Deploy root contract
	rootContract := constructRootContract(t, decompressedSize, addresses, dictType)
	rootAddress := deployContract(t, ctx, auth, l2client, rootContract)
	colors.PrintGrey(name, ": root contract deployed to ", rootAddress.Hex())

	// Activate
	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)
	auth.Value = oneEth
	tx, err := arbWasm.ActivateProgram(&auth, rootAddress)
	Require(t, err)

	receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	if cfg.expectActivation {
		Require(t, err)
		return rootAddress, sourceWasm, receipt
	}

	require.Error(t, err, cfg.expectedErr)
	return rootAddress, sourceWasm, nil
}

// Validation Tests

func TestFragmentedContractValidation(t *testing.T) {
	tests := []struct {
		name string
		cfg  deployConfig
	}{
		{
			name: "Valid 2 Fragments",
			cfg:  defaultDeployConfig(),
		},
		{
			name: "Valid 1 Fragment",
			cfg: deployConfig{
				fragmentCount:    1,
				expectActivation: true,
			},
		},
		{
			name: "Zero Fragments",
			cfg: deployConfig{
				fragmentCount:    0,
				expectActivation: false,
				expectedErr:      "We can't deploy fragmented contracts which have zero fragments",
			},
		},
		{
			name: "Too Many Fragments (3)",
			cfg: deployConfig{
				fragmentCount:    3,
				expectActivation: false,
				expectedErr:      "more fragments then the current limit",
			},
		},
		{
			name: "Decompression Size Too Small",
			cfg: deployConfig{
				fragmentCount:    2,
				expectActivation: false,
				expectedErr:      "smaller decompression size then the actual wasm size",
				mutateSize:       func(i int) int { return i - 1 },
			},
		},
		{
			name: "Decompression Size Too Big",
			cfg: deployConfig{
				fragmentCount:    2,
				expectActivation: false,
				expectedErr:      "bigger decompression size then the actual wasm size",
				mutateSize:       func(i int) int { return i + 1 },
			},
		},
		{
			name: "Incorrect Dictionary Type",
			cfg: deployConfig{
				fragmentCount:    2,
				expectActivation: false,
				expectedErr:      "incorrect dictionary type",
				mutateDict: func(d arbcompress.Dictionary) arbcompress.Dictionary {
					return (d + 1) % 2
				},
			},
		},
		{
			name: "Invalid Address Order",
			cfg: deployConfig{
				fragmentCount:    2,
				expectActivation: false,
				expectedErr:      "fragment addresses in the wrong order",
				mutateAddrs: func(addrs []common.Address) {
					slices.Reverse(addrs)
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
				b.WithExtraArchs(allWasmTargets)
				b.WithArbOSVersion(params.ArbosVersion_StylusContractLimit)
			})
			defer cleanup()

			// If testing 0 fragments, readFragmentedContractFile returns empty sourceWasm,
			// creating an issue for constructRootContract size calculation if not handled.
			// The helper handles typical cases, edge cases are covered by logic inside deployAndActivate.
			deployAndActivateFragmentedContract(t, builder.ctx, auth, builder.L2.Client, tt.cfg)
		})
	}
}

func TestFragmentActivationChargesPerFragmentCodeRead(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithArbOSVersion(params.ArbosVersion_StylusContractLimit)
	})
	defer cleanup()

	file := rustFile("storage")
	fragmentsOne, _, _ := readFragmentedContractFile(t, file, 1)
	fragmentsTwo, _, _ := readFragmentedContractFile(t, file, 2)
	require.Len(t, fragmentsOne, 1)
	require.Len(t, fragmentsTwo, 2)

	minDelta := fragmentReadCostWarmOnly(uint64(len(fragmentsTwo[0]))) + fragmentReadCostWarmOnly(uint64(len(fragmentsTwo[1]))) - fragmentReadCostWarmOnly(uint64(len(fragmentsOne[0])))
	maxDelta := fragmentReadCost(uint64(len(fragmentsTwo[0]))) + fragmentReadCost(uint64(len(fragmentsTwo[1]))) - fragmentReadCost(uint64(len(fragmentsOne[0])))

	_, _, receiptOne := deployAndActivateFragmentedContract(t, builder.ctx, auth, builder.L2.Client, deployConfig{
		fragmentCount:    1,
		expectActivation: true,
	})
	_, _, receiptTwo := deployAndActivateFragmentedContract(t, builder.ctx, auth, builder.L2.Client, deployConfig{
		fragmentCount:    2,
		expectActivation: true,
	})

	require.GreaterOrEqual(t, receiptTwo.GasUsed, receiptOne.GasUsed)
	actualDelta := receiptTwo.GasUsed - receiptOne.GasUsed
	require.GreaterOrEqual(t, actualDelta, minDelta)
	require.LessOrEqual(t, actualDelta, maxDelta)
}

// Specific Edge Case Tests

func TestThatWeCantActivateStylusFragmentContract(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithArbOSVersion(params.ArbosVersion_StylusContractLimit)
	})
	defer cleanup()

	file := rustFile("storage")
	fragments, _, _ := readFragmentedContractFile(t, file, 1)
	require.Len(t, fragments, 1)
	auth.GasLimit = 32000000
	fragmentAddress := deployContract(t, builder.ctx, auth, builder.L2.Client, fragments[0])

	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, builder.L2.Client)
	Require(t, err)
	auth.Value = oneEth
	tx, err := arbWasm.ActivateProgram(&auth, fragmentAddress)
	Require(t, err)
	_, err = EnsureTxSucceeded(builder.ctx, builder.L2.Client, tx)
	require.Error(t, err, "We can't activate a stylus fragment contract directly")
}

func TestDeployStylusRootContractGreaterThanMaxCodeSize(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithArbOSVersion(params.ArbosVersion_StylusContractLimit)
		b.chainConfig.ArbitrumChainParams.MaxCodeSize = 4500
	})
	defer cleanup()

	// 1. Classic contract fail
	file := rustFile("storage")
	wasm, _ := readWasmFile(t, file)
	auth.GasLimit = 32000000
	_, err := deployContractForwardError(t, builder.ctx, auth, builder.L2.Client, wasm)
	require.Error(t, err, "We can't activate a classic stylus contract greater than the MaxCodeSize")

	// 2. Fragmented contract success
	deployAndActivateFragmentedContract(t, builder.ctx, auth, builder.L2.Client, defaultDeployConfig())
}

// ArbOwner Limit Modification Tests

func TestCantActivateRootContractBiggerThanMaxWasmSize(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithArbOSVersion(params.ArbosVersion_StylusContractLimit)
	})
	defer cleanup()

	// Deploy manually to inject custom logic before activation
	file := rustFile("storage")
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	fragments, sourceWasm, dictType := readFragmentedContractFile(t, file, 2)
	auth.GasLimit = 32000000

	addresses := make([]common.Address, 0, len(fragments))
	for i, fragment := range fragments {
		addr := deployContract(t, builder.ctx, auth, builder.L2.Client, fragment)
		addresses = append(addresses, addr)
		colors.PrintGrey(name, ": fragment", i, addr.Hex())
	}

	// #nosec G115
	rootContract := constructRootContract(t, uint32(len(sourceWasm)), addresses, dictType)
	rootAddress := deployContract(t, builder.ctx, auth, builder.L2.Client, rootContract)

	// Decrease limit
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	// #nosec G115
	tx, err := arbOwner.SetWasmMaxSize(&auth, uint32(len(sourceWasm)-1))
	Require(t, err)
	_, err = EnsureTxSucceeded(builder.ctx, builder.L2.Client, tx)
	Require(t, err)

	// Attempt activation
	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, builder.L2.Client)
	Require(t, err)
	auth.Value = oneEth
	tx, err = arbWasm.ActivateProgram(&auth, rootAddress)
	Require(t, err)
	_, err = EnsureTxSucceeded(builder.ctx, builder.L2.Client, tx)
	require.Error(t, err, "We can't activate a fragmented contract greater than the MaxWasmSize")
}

func TestArbOwnerModifyingMaxFragmentCount(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithArbOSVersion(params.ArbosVersion_StylusContractLimit)
	})
	defer cleanup()

	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, builder.L2.Client)
	Require(t, err)
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	callOpts := &bind.CallOpts{Context: builder.ctx}

	// Verify initial
	count, err := arbOwnerPublic.GetMaxStylusContractFragments(callOpts)
	Require(t, err)
	require.Equal(t, uint8(2), count)

	// Change to 1
	tx, err := arbOwner.SetMaxStylusContractFragments(&auth, 1)
	Require(t, err)
	_, err = EnsureTxSucceeded(builder.ctx, builder.L2.Client, tx)
	Require(t, err)

	count, err = arbOwnerPublic.GetMaxStylusContractFragments(callOpts)
	Require(t, err)
	require.Equal(t, uint8(1), count)

	// Change to 3
	tx, err = arbOwner.SetMaxStylusContractFragments(&auth, 3)
	Require(t, err)
	_, err = EnsureTxSucceeded(builder.ctx, builder.L2.Client, tx)
	Require(t, err)

	count, err = arbOwnerPublic.GetMaxStylusContractFragments(callOpts)
	Require(t, err)
	require.Equal(t, uint8(3), count)
}

func TestArbOwnerPublicReturnsCorrectMaxFragmentCount(t *testing.T) {
	builder, _, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithArbOSVersion(params.ArbosVersion_StylusContractLimit)
	})
	defer cleanup()

	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, builder.L2.Client)
	Require(t, err)

	count, err := arbOwnerPublic.GetMaxStylusContractFragments(&bind.CallOpts{Context: builder.ctx})
	Require(t, err)
	require.Equal(t, uint8(2), count)
}

// Generic Runners for Limit Decrease Scenarios
// These generic functions handle the heavy lifting for Rebuild, Execute, Cache, and Deploy tests.
// They accept a `setLimitFunc` to toggle between testing MaxWasmSize and MaxFragmentCount.

type limitSetter func(t *testing.T, ctx context.Context, auth *bind.TransactOpts, client *ethclient.Client)

func runRebuildWasmStoreTest(t *testing.T, setLimit limitSetter) {
	databaseEngine := rawdb.DBLeveldb
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithArbOSVersion(params.ArbosVersion_StylusContractLimit)
		b.WithDatabase(databaseEngine)
	})
	ctx := builder.ctx
	defer cleanup()

	rootAddress, _, _ := deployAndActivateFragmentedContract(t, ctx, auth, builder.L2.Client, defaultDeployConfig())

	// Store value
	zero := common.Hash{}
	val := common.HexToHash("0x121233445566")
	storeTx := builder.L2Info.PrepareTxTo("Owner", &rootAddress, builder.L2Info.TransferGas, nil, argsForStorageWrite(zero, val))
	Require(t, builder.L2.Client.SendTransaction(ctx, storeTx))
	_, err := EnsureTxSucceeded(ctx, builder.L2.Client, storeTx)
	Require(t, err)

	// Decrease Limit
	auth.GasLimit = 0
	auth.Value = nil
	setLimit(t, ctx, &auth, builder.L2.Client)

	// Build 2nd Node
	testDir := t.TempDir()
	nodeBStack := testhelpers.CreateStackConfigForTest(testDir)
	nodeBStack.DBEngine = databaseEngine
	nodeB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{stackConfig: nodeBStack})

	// Ensure tx succeeds on nodeB
	_, err = EnsureTxSucceeded(ctx, nodeB.Client, storeTx)
	Require(t, err)

	// Verify read
	loadTx := builder.L2Info.PrepareTxTo("Owner", &rootAddress, builder.L2Info.TransferGas, nil, argsForStorageRead(zero))
	result, err := arbutil.SendTxAsCall(ctx, nodeB.Client, loadTx, builder.L2Info.GetAddress("Owner"), nil, true)
	Require(t, err)
	require.Equal(t, val, common.BytesToHash(result))

	// Verify Wasm Store
	wasmDb := nodeB.ExecNode.Backend.ArbInterface().BlockChain().StateCache().WasmStore()
	checkWasmStoreContent(t, wasmDb, builder.execConfig.StylusTarget.WasmTargets(), 1)

	// Rebuild Test: Close, Delete Wasm, Reopen, Rebuild
	cleanupB()
	wasmPath := filepath.Join(testDir, nodeBStack.Name, "wasm")
	dirContents, err := os.ReadDir(wasmPath)
	Require(t, err)
	require.NotEmpty(t, dirContents)
	os.RemoveAll(wasmPath)

	nodeB, cleanupB = builder.Build2ndNode(t, &SecondNodeParams{stackConfig: nodeBStack})

	// Verify empty before rebuild
	wasmDbAfterDelete := nodeB.ExecNode.Backend.ArbInterface().BlockChain().StateCache().WasmStore()
	storeMapAfterDelete, err := createMapFromDb(wasmDbAfterDelete)
	Require(t, err)
	require.Empty(t, storeMapAfterDelete)

	log.Info("starting rebuilding of wasm store")
	execConfig := builder.execConfig
	bc := nodeB.ExecNode.Backend.ArbInterface().BlockChain()
	Require(t, gethexec.RebuildWasmStore(ctx, wasmDbAfterDelete, nodeB.ExecNode.ExecutionDB, execConfig.RPC.MaxRecreateStateDepth, &execConfig.StylusTarget, bc, common.Hash{}, bc.CurrentBlock().Hash()))

	wasmDbAfterRebuild := nodeB.ExecNode.Backend.ArbInterface().BlockChain().StateCache().WasmStore()
	checkWasmStoreContent(t, wasmDbAfterRebuild, builder.execConfig.StylusTarget.WasmTargets(), 1)
	cleanupB()
}

func runExecuteWasmTest(t *testing.T, setLimit limitSetter, deleteWasm bool) {
	databaseEngine := rawdb.DBLeveldb
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithArbOSVersion(params.ArbosVersion_StylusContractLimit)
		b.WithDatabase(databaseEngine)
	})
	ctx := builder.ctx
	defer cleanup()

	rootAddress, _, receipt := deployAndActivateFragmentedContract(t, ctx, auth, builder.L2.Client, defaultDeployConfig())

	// Decrease Limit
	auth.GasLimit = 0
	auth.Value = nil
	setLimit(t, ctx, &auth, builder.L2.Client)

	if deleteWasm {
		arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, builder.L2.Client)
		Require(t, err)
		l, err := arbWasm.ParseProgramActivated(*receipt.Logs[0])
		Require(t, err)

		wasmStore := builder.L2.ExecNode.Backend.ArbInterface().BlockChain().StateCache().WasmStore()
		Require(t, deleteAnyKeysContainingModuleHash(wasmStore, l.ModuleHash))
	}

	// Execute Wasm
	zero := common.Hash{}
	val := common.HexToHash("0x121233445566")
	storeTx := builder.L2Info.PrepareTxTo("Owner", &rootAddress, builder.L2Info.TransferGas, nil, argsForStorageWrite(zero, val))
	Require(t, builder.L2.Client.SendTransaction(ctx, storeTx))
	_, err := EnsureTxSucceeded(ctx, builder.L2.Client, storeTx)
	Require(t, err)
}

func runCacheProgramTest(t *testing.T, setLimit limitSetter) {
	databaseEngine := rawdb.DBLeveldb
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithArbOSVersion(params.ArbosVersion_StylusContractLimit)
		b.WithDatabase(databaseEngine)
	})
	ctx := builder.ctx
	defer cleanup()

	rootAddress, _, receipt := deployAndActivateFragmentedContract(t, ctx, auth, builder.L2.Client, defaultDeployConfig())

	// Decrease Limit
	auth.GasLimit = 0
	auth.Value = nil
	setLimit(t, ctx, &auth, builder.L2.Client)

	// Identify module hash and delete from store
	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, builder.L2.Client)
	Require(t, err)
	l, err := arbWasm.ParseProgramActivated(*receipt.Logs[0])
	Require(t, err)
	wasmStore := builder.L2.ExecNode.Backend.ArbInterface().BlockChain().StateCache().WasmStore()
	Require(t, deleteAnyKeysContainingModuleHash(wasmStore, l.ModuleHash))

	// Cache
	arbWasmCache, err := precompilesgen.NewArbWasmCache(types.ArbWasmCacheAddress, builder.L2.Client)
	Require(t, err)
	_, err = arbWasmCache.CacheProgram(&auth, rootAddress)
	Require(t, err)
}

func runDeployAfterLimitTest(t *testing.T, setLimit limitSetter) {
	databaseEngine := rawdb.DBLeveldb
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithArbOSVersion(params.ArbosVersion_StylusContractLimit)
		b.WithDatabase(databaseEngine)
	})
	ctx := builder.ctx
	defer cleanup()

	// Initial Deploy
	rootAddress, _, _ := deployAndActivateFragmentedContract(t, ctx, auth, builder.L2.Client, defaultDeployConfig())

	// Write
	zero := common.Hash{}
	val := common.HexToHash("0x121233445566")
	storeTx := builder.L2Info.PrepareTxTo("Owner", &rootAddress, builder.L2Info.TransferGas, nil, argsForStorageWrite(zero, val))
	Require(t, builder.L2.Client.SendTransaction(ctx, storeTx))
	_, err := EnsureTxSucceeded(ctx, builder.L2.Client, storeTx)
	Require(t, err)

	// Decrease Limit
	auth.GasLimit = 0
	auth.Value = nil
	setLimit(t, ctx, &auth, builder.L2.Client)

	// Second Deploy (should fail)
	failCfg := defaultDeployConfig()
	failCfg.expectActivation = false
	deployAndActivateFragmentedContract(t, ctx, auth, builder.L2.Client, failCfg)
}

// Specific Implementations of Limit Tests

// Shared Limit Setters
func setWasmLimitTo10k(t *testing.T, ctx context.Context, auth *bind.TransactOpts, client *ethclient.Client) {
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, client)
	Require(t, err)
	tx, err := arbOwner.SetWasmMaxSize(auth, 10000)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
}

func setFragmentLimitTo1(t *testing.T, ctx context.Context, auth *bind.TransactOpts, client *ethclient.Client) {
	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, client)
	Require(t, err)
	tx, err := arbOwner.SetMaxStylusContractFragments(auth, 1)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, client, tx)
	Require(t, err)
}

// Tests: Decrease Max Wasm Size
func TestRebuildWasmStoreWithDecreasedMaxWasmSize(t *testing.T) {
	runRebuildWasmStoreTest(t, setWasmLimitTo10k)
}
func TestExecuteWasmWithDecreasedMaxWasmSizeWasmPresent(t *testing.T) {
	runExecuteWasmTest(t, setWasmLimitTo10k, false)
}
func TestExecuteWasmWithDecreasedMaxWasmSizeRecoverWasm(t *testing.T) {
	runExecuteWasmTest(t, setWasmLimitTo10k, true)
}
func TestCacheProgramWithDecreasedMaxWasmSizeRecoverWasm(t *testing.T) {
	runCacheProgramTest(t, setWasmLimitTo10k)
}
func TestDeployingContractBeforeAndAfterDecreaseMaxWasmSize(t *testing.T) {
	runDeployAfterLimitTest(t, setWasmLimitTo10k)
}

// Tests: Decrease Max Fragment Count
func TestRebuildWasmStoreWithDecreasedMaxFragmentCount(t *testing.T) {
	runRebuildWasmStoreTest(t, setFragmentLimitTo1)
}
func TestExecuteWasmWithDecreasedMaxFragmentCountWasmPresent(t *testing.T) {
	runExecuteWasmTest(t, setFragmentLimitTo1, false)
}
func TestExecuteWasmWithDecreasedMaxFragmentCountRecoverWasm(t *testing.T) {
	runExecuteWasmTest(t, setFragmentLimitTo1, true)
}
func TestCacheProgramWithDecreasedMaxFragmentCountRecoverWasm(t *testing.T) {
	runCacheProgramTest(t, setFragmentLimitTo1)
}
func TestDeployingContractBeforeAndAfterDecreaseMaxFragmentCount(t *testing.T) {
	runDeployAfterLimitTest(t, setFragmentLimitTo1)
}

// Test that fragmented contracts fail on ArbOS versions before the feature is active

func TestFragmentedContractFailsOnArbOS50(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithArbOSVersion(params.ArbosVersion_50)
	})
	defer cleanup()

	fragments, _, _ := readFragmentedContractFile(t, rustFile("storage"), 2)
	require.Len(t, fragments, 2)

	auth.GasLimit = 32_000_000
	_, err := deployContractForwardError(t, builder.ctx, auth, builder.L2.Client, fragments[0])
	require.ErrorContains(t, err, vm.ErrInvalidCode.Error())
}

func TestArbOwnerPublicGetMaxFragmentCountFailsOnArbOS50(t *testing.T) {
	builder, _, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithArbOSVersion(params.ArbosVersion_50)
	})
	defer cleanup()

	arbOwnerPublic, err := precompilesgen.NewArbOwnerPublic(types.ArbOwnerPublicAddress, builder.L2.Client)
	Require(t, err)

	_, err = arbOwnerPublic.GetMaxStylusContractFragments(&bind.CallOpts{Context: builder.ctx})
	require.Error(t, err, "GetMaxStylusContractFragments should fail on ArbOS 50 because the feature is not yet active")
}

func TestArbOwnerSetMaxFragmentCountFailsOnArbOS50(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
		b.WithArbOSVersion(params.ArbosVersion_50)
	})
	defer cleanup()

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)

	tx, err := arbOwner.SetMaxStylusContractFragments(&auth, 10)
	if err == nil {
		_, err = EnsureTxSucceeded(builder.ctx, builder.L2.Client, tx)
	}
	require.Error(t, err, "SetMaxStylusContractFragments should fail on ArbOS 50")
}

// Utils

func fragmentReadCost(codeSize uint64) uint64 {
	if codeSize > 0x1FFFFFFFE0 {
		return 0
	}
	words := (codeSize + 31) / 32
	copyGas := words * params.CopyGas
	memoryGas := memoryExpansionCost(codeSize)
	coldDelta := params.ColdAccountAccessCostEIP2929 - params.WarmStorageReadCostEIP2929
	return params.WarmStorageReadCostEIP2929 + coldDelta + copyGas + memoryGas
}

func fragmentReadCostWarmOnly(codeSize uint64) uint64 {
	if codeSize > 0x1FFFFFFFE0 {
		return 0
	}
	words := (codeSize + 31) / 32
	return params.WarmStorageReadCostEIP2929 + words*params.CopyGas
}

func memoryExpansionCost(size uint64) uint64 {
	if size == 0 {
		return 0
	}
	words := (size + 31) / 32
	linearCost := words * params.MemoryGas
	squareCost := (words * words) / params.QuadCoeffDiv
	return linearCost + squareCost
}

// readFragmentedContractFile reads, compiles, compresses, and fragments a contract.
func readFragmentedContractFile(t *testing.T, file string, fragmentCount uint16) ([][]byte, []byte, arbcompress.Dictionary) {
	t.Helper()
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	source, err := os.ReadFile(file)
	Require(t, err)

	// #nosec G115
	randDict := arbcompress.Dictionary((len(file) + len(t.Name())) % 2)

	wasmSource, err := programs.Wat2Wasm(source)
	Require(t, err)

	fragments := make([][]byte, 0, fragmentCount)
	if fragmentCount == 0 {
		return fragments, wasmSource, randDict
	}

	compressedWasm, err := arbcompress.Compress(wasmSource, arbcompress.LEVEL_WELL, randDict)
	Require(t, err)

	toKb := func(data []byte) float64 { return float64(len(data)) / 1024.0 }
	colors.PrintGrey(fmt.Sprintf("%v: len %.2fK vs %.2fK", name, toKb(compressedWasm), toKb(wasmSource)))

	prefix := state.NewStylusFragmentPrefix()
	payloadLen := len(compressedWasm)
	chunkSize := (payloadLen + int(fragmentCount) - 1) / int(fragmentCount)

	for i := 0; i < int(fragmentCount); i++ {
		start := i * chunkSize
		if start >= payloadLen {
			break
		}
		end := start + chunkSize
		if end > payloadLen {
			end = payloadLen
		}
		frag := make([]byte, 0, len(prefix)+(end-start))
		frag = append(frag, prefix...)
		frag = append(frag, compressedWasm[start:end]...)
		fragments = append(fragments, frag)
	}

	return fragments, wasmSource, randDict
}

func constructRootContract(
	t *testing.T,
	dictionaryTypeUncompressedWasmSize uint32,
	addresses []common.Address,
	dictionaryType arbcompress.Dictionary,
) []byte {
	t.Helper()
	// prefix 3 bytes + dict 1 byte + length 4 bytes + len(address) * 20 bytes
	contract := make([]byte, 0, 3+1+4+len(addresses)*common.AddressLength)
	contract = append(contract, state.NewStylusRootPrefix(byte(dictionaryType))...)
	var sizeBuf [4]byte
	binary.BigEndian.PutUint32(sizeBuf[:], dictionaryTypeUncompressedWasmSize)
	contract = append(contract, sizeBuf[:]...)
	for _, addr := range addresses {
		contract = append(contract, addr.Bytes()...)
	}
	return contract
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
