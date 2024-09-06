// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbtest

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/execution/gethexec"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	pgen "github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/validator/valnode"
)

var oneEth = arbmath.UintToBig(1e18)

var allWasmTargets = []string{string(rawdb.TargetWavm), string(rawdb.TargetArm64), string(rawdb.TargetAmd64), string(rawdb.TargetHost)}

func TestProgramKeccak(t *testing.T) {
	t.Parallel()
	t.Run("WithDefaultWasmTargets", func(t *testing.T) {
		keccakTest(t, true)
	})

	t.Run("WithAllWasmTargets", func(t *testing.T) {
		keccakTest(t, true, func(builder *NodeBuilder) {
			builder.WithExtraArchs(allWasmTargets)
		})
	})
}

func keccakTest(t *testing.T, jit bool, builderOpts ...func(*NodeBuilder)) {
	builder, auth, cleanup := setupProgramTest(t, jit, builderOpts...)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()
	programAddress := deployWasm(t, ctx, auth, l2client, rustFile("keccak"))

	wasmDb := builder.L2.ExecNode.Backend.ArbInterface().BlockChain().StateCache().WasmStore()
	checkWasmStoreContent(t, wasmDb, builder.execConfig.StylusTarget.ExtraArchs, 1)

	wasm, _ := readWasmFile(t, rustFile("keccak"))
	otherAddressSameCode := deployContract(t, ctx, auth, l2client, wasm)
	arbWasm, err := pgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)

	colors.PrintBlue("program deployed to ", programAddress.Hex())
	timed(t, "activate same code", func() {
		if _, err := arbWasm.ActivateProgram(&auth, otherAddressSameCode); err == nil || !strings.Contains(err.Error(), "ProgramUpToDate") {
			Fatal(t, "activate should have failed with ProgramUpToDate", err)
		}
	})
	checkWasmStoreContent(t, wasmDb, builder.execConfig.StylusTarget.ExtraArchs, 1)

	if programAddress == otherAddressSameCode {
		Fatal(t, "expected to deploy at two separate program addresses")
	}

	stylusVersion, err := arbWasm.StylusVersion(nil)
	Require(t, err)
	statedb, err := builder.L2.ExecNode.Backend.ArbInterface().BlockChain().State()
	Require(t, err)
	codehashVersion, err := arbWasm.CodehashVersion(nil, statedb.GetCodeHash(programAddress))
	Require(t, err)
	if codehashVersion != stylusVersion || stylusVersion == 0 {
		Fatal(t, "unexpected versions", stylusVersion, codehashVersion)
	}
	programVersion, err := arbWasm.ProgramVersion(nil, programAddress)
	Require(t, err)
	if programVersion != stylusVersion || stylusVersion == 0 {
		Fatal(t, "unexpected versions", stylusVersion, programVersion)
	}
	otherVersion, err := arbWasm.ProgramVersion(nil, otherAddressSameCode)
	Require(t, err)
	if otherVersion != programVersion {
		Fatal(t, "mismatched versions", stylusVersion, programVersion)
	}

	preimage := []byte("°º¤ø,¸,ø¤°º¤ø,¸,ø¤°º¤ø,¸ nyan nyan ~=[,,_,,]:3 nyan nyan")
	correct := crypto.Keccak256Hash(preimage)

	args := []byte{0x01} // keccak the preimage once
	args = append(args, preimage...)

	timed(t, "execute", func() {
		result := sendContractCall(t, ctx, programAddress, l2client, args)
		if len(result) != 32 {
			Fatal(t, "unexpected return result: ", "result", result)
		}
		hash := common.BytesToHash(result)
		if hash != correct {
			Fatal(t, "computed hash mismatch", hash, correct)
		}
		colors.PrintGrey("keccak(x) = ", hash)
	})
	timed(t, "execute same code, different address", func() {
		result := sendContractCall(t, ctx, otherAddressSameCode, l2client, args)
		if len(result) != 32 {
			Fatal(t, "unexpected return result: ", "result", result)
		}
		hash := common.BytesToHash(result)
		if hash != correct {
			Fatal(t, "computed hash mismatch", hash, correct)
		}
		colors.PrintGrey("keccak(x) = ", hash)
	})

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	// do a mutating call for proving's sake
	_, tx, mock, err := mocksgen.DeployProgramTest(&auth, l2client)
	ensure(tx, err)
	ensure(mock.CallKeccak(&auth, programAddress, args))
	ensure(mock.CallKeccak(&auth, otherAddressSameCode, args))

	validateBlocks(t, 1, jit, builder)
}

func TestProgramActivateTwice(t *testing.T) {
	t.Parallel()
	t.Run("WithDefaultWasmTargets", func(t *testing.T) {
		testActivateTwice(t, true)
	})
	t.Run("WithAllWasmTargets", func(t *testing.T) {
		testActivateTwice(t, true, func(builder *NodeBuilder) {
			builder.WithExtraArchs(allWasmTargets)
		})
	})
}

func testActivateTwice(t *testing.T, jit bool, builderOpts ...func(*NodeBuilder)) {
	builder, auth, cleanup := setupProgramTest(t, jit, builderOpts...)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	arbOwner, err := pgen.NewArbOwner(types.ArbOwnerAddress, l2client)
	Require(t, err)
	ensure(arbOwner.SetInkPrice(&auth, 1))

	wasm, _ := readWasmFile(t, rustFile("keccak"))
	keccakA := deployContract(t, ctx, auth, l2client, wasm)
	keccakB := deployContract(t, ctx, auth, l2client, wasm)

	colors.PrintBlue("keccak program A deployed to ", keccakA)
	colors.PrintBlue("keccak program B deployed to ", keccakB)

	multiAddr := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))

	wasmDb := builder.L2.ExecNode.Backend.ArbInterface().BlockChain().StateCache().WasmStore()
	checkWasmStoreContent(t, wasmDb, builder.execConfig.StylusTarget.ExtraArchs, 1)

	preimage := []byte("it's time to du-du-du-du d-d-d-d-d-d-d de-duplicate")

	keccakArgs := []byte{0x01} // keccak the preimage once
	keccakArgs = append(keccakArgs, preimage...)

	checkReverts := func() {
		msg := ethereum.CallMsg{
			To:   &keccakA,
			Data: keccakArgs,
		}
		_, err = l2client.CallContract(ctx, msg, nil)
		if err == nil || !strings.Contains(err.Error(), "ProgramNotActivated") {
			Fatal(t, "call should have failed with ProgramNotActivated")
		}

		// execute onchain for proving's sake
		tx := l2info.PrepareTxTo("Owner", &keccakA, 1e9, nil, keccakArgs)
		Require(t, l2client.SendTransaction(ctx, tx))
		EnsureTxFailed(t, ctx, l2client, tx)
	}

	// Calling the contract pre-activation should fail.
	checkReverts()
	checkWasmStoreContent(t, wasmDb, builder.execConfig.StylusTarget.ExtraArchs, 1)

	// mechanisms for creating calldata
	activateProgram, _ := util.NewCallParser(pgen.ArbWasmABI, "activateProgram")
	legacyError, _ := util.NewCallParser(pgen.ArbDebugABI, "legacyError")
	callKeccak, _ := util.NewCallParser(mocksgen.ProgramTestABI, "callKeccak")
	pack := func(data []byte, err error) []byte {
		Require(t, err)
		return data
	}
	mockAddr, tx, _, err := mocksgen.DeployProgramTest(&auth, l2client)
	ensure(tx, err)

	// Successfully activate, but then revert
	args := argsForMulticall(vm.CALL, types.ArbWasmAddress, nil, pack(activateProgram(keccakA)))
	args = multicallAppend(args, vm.CALL, types.ArbDebugAddress, pack(legacyError()))

	tx = l2info.PrepareTxTo("Owner", &multiAddr, 1e9, oneEth, args)
	Require(t, l2client.SendTransaction(ctx, tx))
	EnsureTxFailed(t, ctx, l2client, tx)

	// Ensure the revert also reverted keccak's activation
	checkReverts()
	checkWasmStoreContent(t, wasmDb, builder.execConfig.StylusTarget.ExtraArchs, 1)

	// Activate keccak program A, then call into B, which should succeed due to being the same codehash
	args = argsForMulticall(vm.CALL, types.ArbWasmAddress, oneEth, pack(activateProgram(keccakA)))
	args = multicallAppend(args, vm.CALL, mockAddr, pack(callKeccak(keccakB, keccakArgs)))

	tx = l2info.PrepareTxTo("Owner", &multiAddr, 1e9, oneEth, args)
	ensure(tx, l2client.SendTransaction(ctx, tx))
	checkWasmStoreContent(t, wasmDb, builder.execConfig.StylusTarget.ExtraArchs, 2)

	validateBlocks(t, 7, jit, builder)
}

func TestStylusUpgrade(t *testing.T) {
	t.Parallel()
	testStylusUpgrade(t, true)
}

func testStylusUpgrade(t *testing.T, jit bool) {
	builder, auth, cleanup := setupProgramTest(t, false, func(b *NodeBuilder) { b.WithArbOSVersion(params.ArbosVersion_Stylus) })
	defer cleanup()

	ctx := builder.ctx

	l2info := builder.L2Info
	l2client := builder.L2.Client

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	arbOwner, err := pgen.NewArbOwner(types.ArbOwnerAddress, l2client)
	Require(t, err)
	ensure(arbOwner.SetInkPrice(&auth, 1))

	wasm, _ := readWasmFile(t, rustFile("keccak"))
	keccakAddr := deployContract(t, ctx, auth, l2client, wasm)

	colors.PrintBlue("keccak program deployed to ", keccakAddr)

	preimage := []byte("hello, you fool")

	keccakArgs := []byte{0x01} // keccak the preimage once
	keccakArgs = append(keccakArgs, preimage...)

	checkFailWith := func(errMessage string) uint64 {
		msg := ethereum.CallMsg{
			To:   &keccakAddr,
			Data: keccakArgs,
		}
		_, err = l2client.CallContract(ctx, msg, nil)
		if err == nil || !strings.Contains(err.Error(), errMessage) {
			Fatal(t, "call should have failed with "+errMessage, " got: "+err.Error())
		}

		// execute onchain for proving's sake
		tx := l2info.PrepareTxTo("Owner", &keccakAddr, 1e9, nil, keccakArgs)
		Require(t, l2client.SendTransaction(ctx, tx))
		return EnsureTxFailed(t, ctx, l2client, tx).BlockNumber.Uint64()
	}

	checkSucceeds := func() uint64 {
		msg := ethereum.CallMsg{
			To:   &keccakAddr,
			Data: keccakArgs,
		}
		_, err = l2client.CallContract(ctx, msg, nil)
		if err != nil {
			Fatal(t, err)
		}

		// execute onchain for proving's sake
		tx := l2info.PrepareTxTo("Owner", &keccakAddr, 1e9, nil, keccakArgs)
		Require(t, l2client.SendTransaction(ctx, tx))
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		if err != nil {
			Fatal(t, err)
		}
		return receipt.BlockNumber.Uint64()
	}

	// Calling the contract pre-activation should fail.
	blockFail1 := checkFailWith("ProgramNotActivated")

	activateWasm(t, ctx, auth, l2client, keccakAddr, "keccak")

	blockSuccess1 := checkSucceeds()

	tx, err := arbOwner.ScheduleArbOSUpgrade(&auth, 31, 0)
	Require(t, err)
	_, err = builder.L2.EnsureTxSucceeded(tx)
	Require(t, err)

	// generate traffic to perform the upgrade
	TransferBalance(t, "Owner", "Owner", big.NewInt(1), builder.L2Info, builder.L2.Client, ctx)

	blockFail2 := checkFailWith("ProgramNeedsUpgrade")

	activateWasm(t, ctx, auth, l2client, keccakAddr, "keccak")

	blockSuccess2 := checkSucceeds()

	validateBlockRange(t, []uint64{blockFail1, blockSuccess1, blockFail2, blockSuccess2}, jit, builder)
}

func TestProgramErrors(t *testing.T) {
	t.Parallel()
	errorTest(t, true)
}

func errorTest(t *testing.T, jit bool) {
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	programAddress := deployWasm(t, ctx, auth, l2client, rustFile("fallible"))
	multiAddr := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))

	// ensure tx passes
	tx := l2info.PrepareTxTo("Owner", &programAddress, l2info.TransferGas, nil, []byte{0x01})
	Require(t, l2client.SendTransaction(ctx, tx))
	_, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	// ensure tx fails
	tx = l2info.PrepareTxTo("Owner", &programAddress, l2info.TransferGas, nil, []byte{0x00})
	Require(t, l2client.SendTransaction(ctx, tx))
	receipt, err := WaitForTx(ctx, l2client, tx.Hash(), 5*time.Second)
	Require(t, err)
	if receipt.Status != types.ReceiptStatusFailed {
		Fatal(t, "call should have failed")
	}

	// ensure tx recovery is correct after failing in a deeply nested call
	args := []byte{}
	for i := 0; i < 32; i++ {
		args = argsForMulticall(vm.CALL, multiAddr, nil, args)
	}
	tx = l2info.PrepareTxTo("Owner", &multiAddr, 1e9, nil, args)
	Require(t, l2client.SendTransaction(ctx, tx))
	EnsureTxFailed(t, ctx, l2client, tx)

	validateBlocks(t, 7, jit, builder)
}

func TestProgramStorage(t *testing.T) {
	t.Parallel()
	storageTest(t, true)
}

func storageTest(t *testing.T, jit bool) {
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()
	programAddress := deployWasm(t, ctx, auth, l2client, rustFile("storage"))

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	key := testhelpers.RandomHash()
	value := testhelpers.RandomHash()
	tx := l2info.PrepareTxTo("Owner", &programAddress, l2info.TransferGas, nil, argsForStorageWrite(key, value))
	ensure(tx, l2client.SendTransaction(ctx, tx))
	assertStorageAt(t, ctx, l2client, programAddress, key, value)

	validateBlocks(t, 2, jit, builder)
}

func TestProgramTransientStorage(t *testing.T) {
	transientStorageTest(t, true)
}

func transientStorageTest(t *testing.T, jit bool) {
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	storage := deployWasm(t, ctx, auth, l2client, rustFile("storage"))
	multicall := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))

	trans := func(args []byte) []byte {
		args[0] += 2
		return args
	}

	zero := common.Hash{}
	keys := []common.Hash{}
	values := []common.Hash{}
	stored := []common.Hash{}
	args := argsForMulticall(vm.CALL, storage, nil, trans(argsForStorageWrite(zero, zero)))

	for i := 0; i < 8; i++ {
		keys = append(keys, testhelpers.RandomHash())
		values = append(values, testhelpers.RandomHash())
		if i%2 == 0 {
			args = multicallAppend(args, vm.CALL, storage, argsForStorageWrite(keys[i], values[i]))
			args = multicallAppend(args, vm.CALL, storage, argsForStorageRead(keys[i]))
			stored = append(stored, values[i])
		} else {
			args = multicallAppend(args, vm.CALL, storage, trans(argsForStorageWrite(keys[i], values[i])))
			args = multicallAppend(args, vm.CALL, storage, trans(argsForStorageRead(keys[i])))
			stored = append(stored, zero)
		}
	}

	// do an onchain call
	tx := l2info.PrepareTxTo("Owner", &multicall, l2info.TransferGas, nil, args)
	Require(t, l2client.SendTransaction(ctx, tx))
	_, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	// do an equivalent eth_call
	msg := ethereum.CallMsg{
		To:   &multicall,
		Data: args,
	}
	outs, err := l2client.CallContract(ctx, msg, nil)
	Require(t, err)

	for i, key := range keys {
		offset := i * 32
		value := common.BytesToHash(outs[offset : offset+32])
		if values[i] != value {
			Fatal(t, "unexpected value in transient storage", i, values[i], value)
		}
		assertStorageAt(t, ctx, l2client, storage, key, stored[i])
		assertStorageAt(t, ctx, l2client, multicall, key, zero)
	}

	validateBlocks(t, 7, jit, builder)
}

func TestProgramMath(t *testing.T) {
	t.Parallel()
	fastMathTest(t, true)
}

func fastMathTest(t *testing.T, jit bool) {
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	program := deployWasm(t, ctx, auth, l2client, rustFile("math"))

	_, tx, mock, err := mocksgen.DeployProgramTest(&auth, l2client)
	ensure(tx, err)
	ensure(mock.MathTest(&auth, program))

	validateBlocks(t, 6, jit, builder)
}

func TestProgramCalls(t *testing.T) {
	t.Parallel()
	testCalls(t, true)
}

func testCalls(t *testing.T, jit bool) {
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()
	callsAddr := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	expectFailure := func(to common.Address, data []byte, errMsg string) {
		t.Helper()
		msg := ethereum.CallMsg{
			To:   &to,
			Data: data,
		}
		_, err := l2client.CallContract(ctx, msg, nil)
		if err == nil {
			Fatal(t, "call should have failed with", errMsg)
		}
		expected := fmt.Sprintf("execution reverted%v", errMsg)
		if err.Error() != expected {
			Fatal(t, "wrong error", err.Error(), " ", expected)
		}

		// execute onchain for proving's sake
		tx := l2info.PrepareTxTo("Owner", &callsAddr, 1e9, nil, data)
		Require(t, l2client.SendTransaction(ctx, tx))
		EnsureTxFailed(t, ctx, l2client, tx)
	}

	storeAddr := deployWasm(t, ctx, auth, l2client, rustFile("storage"))
	keccakAddr := deployWasm(t, ctx, auth, l2client, rustFile("keccak"))
	mockAddr, tx, _, err := mocksgen.DeployProgramTest(&auth, l2client)
	ensure(tx, err)

	colors.PrintGrey("multicall.wasm ", callsAddr)
	colors.PrintGrey("storage.wasm   ", storeAddr)
	colors.PrintGrey("keccak.wasm    ", keccakAddr)
	colors.PrintGrey("mock.evm       ", mockAddr)

	kinds := make(map[vm.OpCode]byte)
	kinds[vm.CALL] = 0x00
	kinds[vm.DELEGATECALL] = 0x01
	kinds[vm.STATICCALL] = 0x02

	checkTree := func(opcode vm.OpCode, dest common.Address) map[common.Hash]common.Hash {
		colors.PrintBlue("Checking storage after call tree with ", opcode)
		slots := make(map[common.Hash]common.Hash)
		zeroHashBytes := common.BigToHash(common.Big0).Bytes()

		var nest func(level uint) []uint8
		nest = func(level uint) []uint8 {
			args := []uint8{}

			if level == 0 {
				// call storage.wasm
				args = append(args, kinds[opcode])
				if opcode == vm.CALL {
					args = append(args, zeroHashBytes...)
				}
				args = append(args, storeAddr[:]...)

				key := testhelpers.RandomHash()
				value := testhelpers.RandomHash()
				slots[key] = value

				// insert value @ key
				args = append(args, argsForStorageWrite(key, value)...)
				return args
			}

			// do the two following calls
			args = append(args, kinds[opcode])
			if opcode == vm.CALL {
				args = append(args, zeroHashBytes...)
			}
			args = append(args, callsAddr[:]...)
			args = append(args, 2)

			for i := 0; i < 2; i++ {
				inner := nest(level - 1)
				// #nosec G115
				args = append(args, arbmath.Uint32ToBytes(uint32(len(inner)))...)
				args = append(args, inner...)
			}
			return args
		}
		var tree []uint8
		if opcode == vm.CALL {
			tree = nest(3)[53:]
		} else {
			tree = nest(3)[21:]
		}
		tx = l2info.PrepareTxTo("Owner", &callsAddr, 1e9, nil, tree)
		ensure(tx, l2client.SendTransaction(ctx, tx))

		for key, value := range slots {
			assertStorageAt(t, ctx, l2client, dest, key, value)
		}
		return slots
	}

	slots := checkTree(vm.CALL, storeAddr)
	checkTree(vm.DELEGATECALL, callsAddr)

	colors.PrintBlue("Checking static call")
	calldata := []byte{0}
	expected := []byte{}
	for key, value := range slots {
		calldata = multicallAppend(calldata, vm.STATICCALL, storeAddr, argsForStorageRead(key))
		expected = append(expected, value[:]...)
	}
	values := sendContractCall(t, ctx, callsAddr, l2client, calldata)
	if !bytes.Equal(expected, values) {
		Fatal(t, "wrong results static call", common.Bytes2Hex(expected), common.Bytes2Hex(values))
	}
	tx = l2info.PrepareTxTo("Owner", &callsAddr, 1e9, nil, calldata)
	ensure(tx, l2client.SendTransaction(ctx, tx))

	colors.PrintBlue("Checking static call write protection")
	writeKey := append([]byte{0x1}, testhelpers.RandomHash().Bytes()...)
	writeKey = append(writeKey, testhelpers.RandomHash().Bytes()...)
	expectFailure(callsAddr, argsForMulticall(vm.STATICCALL, storeAddr, nil, writeKey), "")

	// mechanisms for creating calldata
	burnArbGas, _ := util.NewCallParser(pgen.ArbosTestABI, "burnArbGas")
	customRevert, _ := util.NewCallParser(pgen.ArbDebugABI, "customRevert")
	legacyError, _ := util.NewCallParser(pgen.ArbDebugABI, "legacyError")
	callKeccak, _ := util.NewCallParser(mocksgen.ProgramTestABI, "callKeccak")
	pack := func(data []byte, err error) []byte {
		Require(t, err)
		return data
	}

	colors.PrintBlue("Calling the ArbosTest precompile (Rust => precompile)")
	testPrecompile := func(gas uint64) uint64 {
		// Call the burnArbGas() precompile from Rust
		// #nosec G115
		burn := pack(burnArbGas(big.NewInt(int64(gas))))
		args := argsForMulticall(vm.CALL, types.ArbosTestAddress, nil, burn)
		tx := l2info.PrepareTxTo("Owner", &callsAddr, 1e9, nil, args)
		receipt := ensure(tx, l2client.SendTransaction(ctx, tx))
		return receipt.GasUsed - receipt.GasUsedForL1
	}

	smallGas := testhelpers.RandomUint64(2000, 8000)
	largeGas := smallGas + testhelpers.RandomUint64(2000, 8000)
	small := testPrecompile(smallGas)
	large := testPrecompile(largeGas)

	if !arbmath.Within(large-small, largeGas-smallGas, 2) {
		// #nosec G115
		ratio := float64(int64(large)-int64(small)) / float64(int64(largeGas)-int64(smallGas))
		Fatal(t, "inconsistent burns", large, small, largeGas, smallGas, ratio)
	}

	colors.PrintBlue("Checking consensus revert data (Rust => precompile)")
	args := argsForMulticall(vm.CALL, types.ArbDebugAddress, nil, pack(customRevert(uint64(32))))
	spider := ": error Custom(32, This spider family wards off bugs: /\\oo/\\ //\\(oo)//\\ /\\oo/\\, true)"
	expectFailure(callsAddr, args, spider)

	colors.PrintBlue("Checking non-consensus revert data (Rust => precompile)")
	args = argsForMulticall(vm.CALL, types.ArbDebugAddress, nil, pack(legacyError()))
	expectFailure(callsAddr, args, "")

	colors.PrintBlue("Checking success (Rust => Solidity => Rust)")
	rustArgs := append([]byte{0x01}, []byte(spider)...)
	mockArgs := argsForMulticall(vm.CALL, mockAddr, nil, pack(callKeccak(keccakAddr, rustArgs)))
	tx = l2info.PrepareTxTo("Owner", &callsAddr, 1e9, nil, mockArgs)
	ensure(tx, l2client.SendTransaction(ctx, tx))

	colors.PrintBlue("Checking call with value (Rust => EOA)")
	eoa := testhelpers.RandomAddress()
	value := testhelpers.RandomCallValue(1e12)
	args = argsForMulticall(vm.CALL, eoa, value, []byte{})
	tx = l2info.PrepareTxTo("Owner", &callsAddr, 1e9, value, args)
	ensure(tx, l2client.SendTransaction(ctx, tx))
	balance := GetBalance(t, ctx, l2client, eoa)
	if !arbmath.BigEquals(balance, value) {
		Fatal(t, balance, value)
	}

	blocks := []uint64{10}
	validateBlockRange(t, blocks, jit, builder)
}

func TestProgramReturnData(t *testing.T) {
	t.Parallel()
	testReturnData(t, true)
}

func testReturnData(t *testing.T, jit bool) {
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	ensure := func(tx *types.Transaction, err error) {
		t.Helper()
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
	}

	readReturnDataAddr := deployWasm(t, ctx, auth, l2client, rustFile("read-return-data"))

	colors.PrintGrey("read-return-data.evm ", readReturnDataAddr)
	colors.PrintBlue("checking calls with partial return data")

	dataToSend := [4]byte{0, 1, 2, 3}
	testReadReturnData := func(callType uint32, offset uint32, size uint32, expectedSize uint32, count uint32) {
		parameters := [20]byte{}
		binary.BigEndian.PutUint32(parameters[0:4], callType)
		binary.BigEndian.PutUint32(parameters[4:8], offset)
		binary.BigEndian.PutUint32(parameters[8:12], size)
		binary.BigEndian.PutUint32(parameters[12:16], expectedSize)
		binary.BigEndian.PutUint32(parameters[16:20], count)
		callData := append(parameters[:], dataToSend[:]...)

		tx := l2info.PrepareTxTo("Owner", &readReturnDataAddr, 1e9, nil, callData)
		ensure(tx, l2client.SendTransaction(ctx, tx))
	}

	testReadReturnData(1, 0, 5, 4, 2)
	testReadReturnData(1, 0, 1, 1, 2)
	testReadReturnData(1, 5, 1, 0, 2)
	testReadReturnData(1, 0, 0, 0, 2)
	testReadReturnData(1, 0, 4, 4, 2)

	testReadReturnData(2, 0, 5, 4, 1)
	testReadReturnData(2, 0, 1, 1, 1)
	testReadReturnData(2, 5, 1, 0, 1)
	testReadReturnData(2, 0, 0, 0, 1)
	testReadReturnData(2, 0, 4, 4, 1)

	validateBlocks(t, 11, jit, builder)
}

func TestProgramLogs(t *testing.T) {
	t.Parallel()
	testLogs(t, true, false)
}

func TestProgramLogsWithTracing(t *testing.T) {
	t.Parallel()
	testLogs(t, true, true)
}

func testLogs(t *testing.T, jit, tracing bool) {
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()
	logAddr := deployWasm(t, ctx, auth, l2client, rustFile("log"))
	multiAddr := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))

	type traceLog struct {
		Address common.Address `json:"address"`
		Topics  []common.Hash  `json:"topics"`
		Data    hexutil.Bytes  `json:"data"`
	}
	traceTx := func(tx *types.Transaction) []traceLog {
		type traceLogs struct {
			Logs []traceLog `json:"logs"`
		}
		var trace traceLogs
		traceConfig := map[string]interface{}{
			"tracer": "callTracer",
			"tracerConfig": map[string]interface{}{
				"withLog": true,
			},
		}
		rpc := l2client.Client()
		err := rpc.CallContext(ctx, &trace, "debug_traceTransaction", tx.Hash(), traceConfig)
		Require(t, err)
		return trace.Logs
	}
	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}
	encode := func(topics []common.Hash, data []byte) []byte {
		args := []byte{byte(len(topics))}
		for _, topic := range topics {
			args = append(args, topic[:]...)
		}
		args = append(args, data...)
		return args
	}
	randBytes := func(min, max uint64) []byte {
		return testhelpers.RandomSlice(testhelpers.RandomUint64(min, max))
	}

	for i := 0; i <= 4; i++ {
		colors.PrintGrey("Emitting ", i, " topics")
		topics := make([]common.Hash, i)
		for j := 0; j < i; j++ {
			topics[j] = testhelpers.RandomHash()
		}
		data := randBytes(0, 48)
		verifyLogTopicsAndData := func(logData []byte, logTopics []common.Hash) {
			if !bytes.Equal(logData, data) {
				Fatal(t, "data mismatch", logData, data)
			}
			if len(logTopics) != len(topics) {
				Fatal(t, "topics mismatch", len(logTopics), len(topics))
			}
			for j := 0; j < i; j++ {
				if logTopics[j] != topics[j] {
					Fatal(t, "topic mismatch", logTopics, topics)
				}
			}
		}

		args := encode(topics, data)
		tx := l2info.PrepareTxTo("Owner", &logAddr, 1e9, nil, args)
		receipt := ensure(tx, l2client.SendTransaction(ctx, tx))

		if len(receipt.Logs) != 1 {
			Fatal(t, "wrong number of logs", len(receipt.Logs))
		}
		log := receipt.Logs[0]
		verifyLogTopicsAndData(log.Data, log.Topics)
		if tracing {
			logs := traceTx(tx)
			if len(logs) != 1 {
				Fatal(t, "wrong number of logs in trace", len(logs))
			}
			log := logs[0]
			verifyLogTopicsAndData(log.Data, log.Topics)
		}
	}

	tooMany := encode([]common.Hash{{}, {}, {}, {}, {}}, []byte{})
	tx := l2info.PrepareTxTo("Owner", &logAddr, 1e9, nil, tooMany)
	Require(t, l2client.SendTransaction(ctx, tx))
	EnsureTxFailed(t, ctx, l2client, tx)

	delegate := argsForMulticall(vm.DELEGATECALL, logAddr, nil, []byte{0x00})
	tx = l2info.PrepareTxTo("Owner", &multiAddr, 1e9, nil, delegate)
	receipt := ensure(tx, l2client.SendTransaction(ctx, tx))
	if receipt.Logs[0].Address != multiAddr {
		Fatal(t, "wrong address", receipt.Logs[0].Address)
	}

	validateBlocks(t, 11, jit, builder)
}

func TestProgramCreate(t *testing.T) {
	t.Parallel()
	testCreate(t, true)
}

func testCreate(t *testing.T, jit bool) {
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()
	createAddr := deployWasm(t, ctx, auth, l2client, rustFile("create"))
	activateAuth := auth
	activateAuth.Value = oneEth

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	deployWasm, _ := readWasmFile(t, rustFile("storage"))
	deployCode := deployContractInitCode(deployWasm, false)
	startValue := testhelpers.RandomCallValue(1e12)
	salt := testhelpers.RandomHash()

	create := func(createArgs []byte, correctStoreAddr common.Address) {
		tx := l2info.PrepareTxTo("Owner", &createAddr, 1e9, startValue, createArgs)
		receipt := ensure(tx, l2client.SendTransaction(ctx, tx))
		storeAddr := common.BytesToAddress(receipt.Logs[0].Topics[0][:])
		if storeAddr == (common.Address{}) {
			Fatal(t, "failed to deploy storage.wasm")
		}
		colors.PrintBlue("deployed keccak to ", storeAddr.Hex())
		balance, err := l2client.BalanceAt(ctx, storeAddr, nil)
		Require(t, err)
		if !arbmath.BigEquals(balance, startValue) {
			Fatal(t, "storage.wasm has the wrong balance", balance, startValue)
		}

		// activate the program
		arbWasm, err := pgen.NewArbWasm(types.ArbWasmAddress, l2client)
		Require(t, err)
		tx, err = arbWasm.ActivateProgram(&activateAuth, storeAddr)
		if err != nil {
			if !strings.Contains(err.Error(), "ProgramUpToDate") {
				Fatal(t, err)
			}
		} else {
			_, succeedErr := EnsureTxSucceeded(ctx, l2client, tx)
			Require(t, succeedErr)
		}

		// check the program works
		key := testhelpers.RandomHash()
		value := testhelpers.RandomHash()
		tx = l2info.PrepareTxTo("Owner", &storeAddr, 1e9, nil, argsForStorageWrite(key, value))
		ensure(tx, l2client.SendTransaction(ctx, tx))
		assertStorageAt(t, ctx, l2client, storeAddr, key, value)

		if storeAddr != correctStoreAddr {
			Fatal(t, "program deployed to the wrong address", storeAddr, correctStoreAddr)
		}
	}

	create1Args := []byte{0x01}
	create1Args = append(create1Args, common.BigToHash(startValue).Bytes()...)
	create1Args = append(create1Args, deployCode...)

	create2Args := []byte{0x02}
	create2Args = append(create2Args, common.BigToHash(startValue).Bytes()...)
	create2Args = append(create2Args, salt[:]...)
	create2Args = append(create2Args, deployCode...)

	create1Addr := crypto.CreateAddress(createAddr, 1)
	create2Addr := crypto.CreateAddress2(createAddr, salt, crypto.Keccak256(deployCode))
	create(create1Args, create1Addr)
	create(create2Args, create2Addr)

	revertData := []byte("✌(✰‿✰)✌ ┏(✰‿✰)┛ ┗(✰‿✰)┓ ┗(✰‿✰)┛ ┏(✰‿✰)┓ ✌(✰‿✰)✌")
	revertArgs := []byte{0x01}
	revertArgs = append(revertArgs, common.BigToHash(startValue).Bytes()...)
	revertArgs = append(revertArgs, deployContractInitCode(revertData, true)...)

	_, tx, mock, err := mocksgen.DeployProgramTest(&auth, l2client)
	ensure(tx, err)
	auth.Value = startValue
	ensure(mock.CheckRevertData(&auth, createAddr, revertArgs, revertData))

	// validate just the opcodes
	blocks := []uint64{5, 6}
	validateBlockRange(t, blocks, jit, builder)
}

func TestProgramMemory(t *testing.T) {
	t.Parallel()
	testMemory(t, true)
}

func testMemory(t *testing.T, jit bool) {
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	arbOwner, err := pgen.NewArbOwner(types.ArbOwnerAddress, l2client)
	Require(t, err)
	arbWasm, err := pgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)

	ensure(arbOwner.SetInkPrice(&auth, 1e4))
	ensure(arbOwner.SetMaxTxGasLimit(&auth, 34000000))

	memoryAddr := deployWasm(t, ctx, auth, l2client, watFile("memory"))
	multiAddr := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))
	growCallAddr := deployWasm(t, ctx, auth, l2client, watFile("grow/grow-and-call"))
	growFixed := deployWasm(t, ctx, auth, l2client, watFile("grow/fixed"))
	memWrite := deployWasm(t, ctx, auth, l2client, watFile("grow/mem-write"))

	expectFailure := func(to common.Address, data []byte, value *big.Int) {
		t.Helper()
		msg := ethereum.CallMsg{
			To:    &to,
			Value: big.NewInt(0),
			Data:  data,
			Gas:   32000000,
		}
		_, err := l2client.CallContract(ctx, msg, nil)
		if err == nil {
			Fatal(t, "call should have failed")
		}

		// execute onchain for proving's sake
		tx := l2info.PrepareTxTo("Owner", &to, 1e9, value, data)
		Require(t, l2client.SendTransaction(ctx, tx))
		EnsureTxFailed(t, ctx, l2client, tx)
	}

	model := programs.NewMemoryModel(programs.InitialFreePages, programs.InitialPageGas)

	// expand to 128 pages, retract, then expand again to 128.
	//   - multicall takes 1 page to init, and then 1 more at runtime.
	//   - grow-and-call takes 1 page, then grows to the first arg by second arg steps.
	args := argsForMulticall(vm.CALL, memoryAddr, nil, []byte{126, 50})
	args = multicallAppend(args, vm.CALL, memoryAddr, []byte{126, 80})

	tx := l2info.PrepareTxTo("Owner", &multiAddr, 1e9, nil, args)
	receipt := ensure(tx, l2client.SendTransaction(ctx, tx))
	gasCost := receipt.GasUsedForL2()
	memCost := model.GasCost(128, 0, 0) + model.GasCost(126, 2, 128)
	logical := uint64(32000000 + 126*programs.InitialPageGas)
	if !arbmath.WithinRange(gasCost, memCost, memCost+2e5) || !arbmath.WithinRange(gasCost, logical, logical+2e5) {
		Fatal(t, "unexpected cost", gasCost, memCost, logical)
	}

	// check that we'd normally run out of gas
	ensure(arbOwner.SetMaxTxGasLimit(&auth, 32000000))
	expectFailure(multiAddr, args, oneEth)

	// check that activation fails when out of memory
	wasm, _ := readWasmFile(t, watFile("grow/grow-120"))
	growHugeAddr := deployContract(t, ctx, auth, l2client, wasm)
	colors.PrintGrey("memory.wat        ", memoryAddr)
	colors.PrintGrey("multicall.rs      ", multiAddr)
	colors.PrintGrey("grow-and-call.wat ", growCallAddr)
	colors.PrintGrey("grow-120.wat      ", growHugeAddr)
	activate, _ := util.NewCallParser(pgen.ArbWasmABI, "activateProgram")
	pack := func(data []byte, err error) []byte {
		Require(t, err)
		return data
	}
	args = arbmath.ConcatByteSlices([]byte{60}, types.ArbWasmAddress[:], pack(activate(growHugeAddr)))
	expectFailure(growCallAddr, args, oneEth) // consumes 64, then tries to compile something 120

	// check that activation then succeeds
	args[0] = 0x00
	tx = l2info.PrepareTxTo("Owner", &growCallAddr, 1e9, oneEth, args)
	receipt = ensure(tx, l2client.SendTransaction(ctx, tx))
	if receipt.GasUsedForL2() < 1659168 {
		Fatal(t, "activation unexpectedly cheap")
	}

	// check footprint can induce a revert
	args = arbmath.ConcatByteSlices([]byte{122}, growCallAddr[:], []byte{0}, common.Address{}.Bytes())
	expectFailure(growCallAddr, args, oneEth)

	// check same call would have succeeded with fewer pages
	args = arbmath.ConcatByteSlices([]byte{119}, growCallAddr[:], []byte{0}, common.Address{}.Bytes())
	tx = l2info.PrepareTxTo("Owner", &growCallAddr, 1e9, nil, args)
	receipt = ensure(tx, l2client.SendTransaction(ctx, tx))
	gasCost = receipt.GasUsedForL2()
	memCost = model.GasCost(127, 0, 0)
	if !arbmath.WithinRange(gasCost, memCost, memCost+1e5) {
		Fatal(t, "unexpected cost", gasCost, memCost)
	}

	// check huge memory footprint
	programMemoryFootprint, err := arbWasm.ProgramMemoryFootprint(nil, growHugeAddr)
	Require(t, err)
	if programMemoryFootprint != 120 {
		Fatal(t, "unexpected memory footprint", programMemoryFootprint)
	}

	// check edge case where memory doesn't require `pay_for_memory_grow`
	tx = l2info.PrepareTxTo("Owner", &growFixed, 1e9, nil, args)
	ensure(tx, l2client.SendTransaction(ctx, tx))

	// check memory boundary conditions
	type Case struct {
		pass bool
		size uint8
		spot uint32
		data uint32
	}
	cases := []Case{
		{true, 0, 0, 0},
		{true, 1, 4, 0},
		{true, 1, 65536, 0},
		{false, 1, 65536, 1}, // 1st byte out of bounds
		{false, 1, 65537, 0}, // 2nd byte out of bounds
		{true, 1, 65535, 1},  // last byte in bounds
		{false, 1, 65535, 2}, // 1st byte over-run
		{true, 2, 131072, 0},
		{false, 2, 131073, 0},
	}
	for _, test := range cases {
		args := []byte{}
		if test.size > 0 {
			args = append(args, test.size)
			args = binary.LittleEndian.AppendUint32(args, test.spot)
			args = binary.LittleEndian.AppendUint32(args, test.data)
		}
		if test.pass {
			tx = l2info.PrepareTxTo("Owner", &memWrite, 1e9, nil, args)
			ensure(tx, l2client.SendTransaction(ctx, tx))
		} else {
			expectFailure(memWrite, args, nil)
		}
	}

	validateBlocks(t, 3, jit, builder)
}

func TestProgramActivateFails(t *testing.T) {
	t.Parallel()
	testActivateFails(t, true)
}

func testActivateFails(t *testing.T, jit bool) {
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	arbWasm, err := pgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)

	badExportWasm, _ := readWasmFile(t, watFile("bad-mods/bad-export"))
	auth.GasLimit = 32000000 // skip gas estimation
	badExportAddr := deployContract(t, ctx, auth, l2client, badExportWasm)

	blockToValidate := uint64(0)
	timed(t, "activate bad-export", func() {
		auth.Value = oneEth
		tx, err := arbWasm.ActivateProgram(&auth, badExportAddr)
		Require(t, err)
		txRes, err := WaitForTx(ctx, l2client, tx.Hash(), time.Second*5)
		Require(t, err)
		if txRes.Status != 0 {
			Fatal(t, "bad-export transaction did not fail")
		}
		gotError := arbutil.DetailTxError(ctx, l2client, tx, txRes)
		if !strings.Contains(gotError.Error(), "reserved symbol") {
			Fatal(t, "unexpected error: ", gotError)
		}
		Require(t, err)
		blockToValidate = txRes.BlockNumber.Uint64()
	})

	validateBlockRange(t, []uint64{blockToValidate}, jit, builder)
}

func TestProgramSdkStorage(t *testing.T) {
	t.Parallel()
	testSdkStorage(t, true)
}

func testSdkStorage(t *testing.T, jit bool) {
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	rust := deployWasm(t, ctx, auth, l2client, rustFile("sdk-storage"))

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}

	solidity, tx, mock, err := mocksgen.DeploySdkStorage(&auth, l2client)
	ensure(tx, err)
	tx, err = mock.Populate(&auth)
	receipt := ensure(tx, err)
	solCost := receipt.GasUsedForL2()

	tx = l2info.PrepareTxTo("Owner", &rust, 1e9, nil, tx.Data())
	receipt = ensure(tx, l2client.SendTransaction(ctx, tx))
	rustCost := receipt.GasUsedForL2()

	check := func() {
		colors.PrintBlue("rust ", rustCost, " sol ", solCost)

		// ensure txes are sequenced before checking state
		waitForSequencer(t, builder, receipt.BlockNumber.Uint64())

		bc := builder.L2.ExecNode.Backend.ArbInterface().BlockChain()
		statedb, err := bc.State()
		Require(t, err)

		solTrie := statedb.GetStorageRoot(solidity)
		rustTrie := statedb.GetStorageRoot(rust)
		if solTrie != rustTrie {
			Fatal(t, solTrie, rustTrie)
		}
	}

	check()

	colors.PrintBlue("checking removal")
	tx, err = mock.Remove(&auth)
	receipt = ensure(tx, err)
	solCost = receipt.GasUsedForL2()

	tx = l2info.PrepareTxTo("Owner", &rust, 1e9, nil, tx.Data())
	receipt = ensure(tx, l2client.SendTransaction(ctx, tx))
	rustCost = receipt.GasUsedForL2()
	check()
}

func TestProgramActivationLogs(t *testing.T) {
	t.Parallel()
	builder, auth, cleanup := setupProgramTest(t, true)
	l2client := builder.L2.Client
	ctx := builder.ctx
	defer cleanup()

	wasm, _ := readWasmFile(t, watFile("memory"))
	arbWasm, err := pgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)

	nolimitAuth := auth
	nolimitAuth.GasLimit = 32000000

	programAddress := deployContract(t, ctx, nolimitAuth, l2client, wasm)

	auth.Value = oneEth
	tx, err := arbWasm.ActivateProgram(&auth, programAddress)
	Require(t, err)
	receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	if len(receipt.Logs) != 1 {
		Fatal(t, "expected 1 log while activating, got ", len(receipt.Logs))
	}
	log, err := arbWasm.ParseProgramActivated(*receipt.Logs[0])
	if err != nil {
		Fatal(t, "parsing activated log: ", err)
	}
	if log.Version == 0 {
		Fatal(t, "activated program with version 0")
	}
	if log.Program != programAddress {
		Fatal(t, "unexpected program in activation log: ", log.Program)
	}
	if crypto.Keccak256Hash(wasm) != log.Codehash {
		Fatal(t, "unexpected codehash in activation log: ", log.Codehash)
	}
}

func TestProgramEarlyExit(t *testing.T) {
	t.Parallel()
	testEarlyExit(t, true)
}

func testEarlyExit(t *testing.T, jit bool) {
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	earlyAddress := deployWasm(t, ctx, auth, l2client, "../arbitrator/stylus/tests/exit-early/exit-early.wat")
	panicAddress := deployWasm(t, ctx, auth, l2client, "../arbitrator/stylus/tests/exit-early/panic-after-write.wat")

	ensure := func(tx *types.Transaction, err error) {
		t.Helper()
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
	}

	_, tx, mock, err := mocksgen.DeployProgramTest(&auth, l2client)
	ensure(tx, err)

	// revert with the following data
	data := append([]byte{0x01}, []byte("private key: https://www.youtube.com/watch?v=dQw4w9WgXcQ")...)

	ensure(mock.CheckRevertData(&auth, earlyAddress, data, data))
	ensure(mock.CheckRevertData(&auth, panicAddress, data, []byte{}))

	validateBlocks(t, 8, jit, builder)
}

func TestProgramCacheManager(t *testing.T) {
	builder, ownerAuth, cleanup := setupProgramTest(t, true)
	ctx := builder.ctx
	l2client := builder.L2.Client
	l2info := builder.L2Info
	defer cleanup()

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
		return receipt
	}
	denytx := func(tx *types.Transaction, err error) {
		t.Helper()
		Require(t, err)
		signer := types.LatestSignerForChainID(tx.ChainId())
		from, err := signer.Sender(tx)
		Require(t, err)
		msg := ethereum.CallMsg{
			To:    tx.To(),
			Value: big.NewInt(0),
			Data:  tx.Data(),
			From:  from,
		}
		_, err = l2client.CallContract(ctx, msg, nil)
		if err == nil {
			Fatal(t, "call should have failed")
		}
	}
	assert := func(cond bool, err error, msg ...interface{}) {
		t.Helper()
		Require(t, err)
		if !cond {
			Fatal(t, msg...)
		}
	}

	// precompiles we plan to use
	arbWasm, err := pgen.NewArbWasm(types.ArbWasmAddress, builder.L2.Client)
	Require(t, err)
	arbWasmCache, err := pgen.NewArbWasmCache(types.ArbWasmCacheAddress, builder.L2.Client)
	Require(t, err)
	arbOwner, err := pgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	ensure(arbOwner.SetInkPrice(&ownerAuth, 10_000))
	parseLog := logParser[pgen.ArbWasmCacheUpdateProgramCache](t, pgen.ArbWasmCacheABI, "UpdateProgramCache")

	// fund a user account we'll use to probe access-restricted methods
	l2info.GenerateAccount("Anyone")
	userAuth := l2info.GetDefaultTransactOpts("Anyone", ctx)
	userAuth.GasLimit = 3e6
	TransferBalance(t, "Owner", "Anyone", arbmath.BigMulByUint(oneEth, 32), l2info, l2client, ctx)

	// deploy without activating a wasm
	wasm, _ := readWasmFile(t, rustFile("keccak"))
	program := deployContract(t, ctx, userAuth, l2client, wasm)
	codehash := crypto.Keccak256Hash(wasm)

	// try to manage the cache without authorization
	manager, tx, mock, err := mocksgen.DeploySimpleCacheManager(&ownerAuth, l2client)
	ensure(tx, err)
	denytx(mock.CacheProgram(&userAuth, program))
	denytx(mock.EvictProgram(&userAuth, program))

	// check non-membership
	isManager, err := arbWasmCache.IsCacheManager(nil, manager)
	assert(!isManager, err)

	// athorize the manager
	ensure(arbOwner.AddWasmCacheManager(&ownerAuth, manager))
	assert(arbWasmCache.IsCacheManager(nil, manager))
	all, err := arbWasmCache.AllCacheManagers(nil)
	assert(len(all) == 1 && all[0] == manager, err)

	// try to cache something inactive
	denytx(mock.CacheProgram(&userAuth, program))
	ensure(mock.EvictProgram(&userAuth, program))
	denytx(mock.CacheProgram(&userAuth, testhelpers.RandomAddress()))
	ensure(mock.EvictProgram(&userAuth, testhelpers.RandomAddress()))

	// cache the active program
	activateWasm(t, ctx, userAuth, l2client, program, "keccak")
	ensure(mock.CacheProgram(&userAuth, program))
	assert(arbWasmCache.CodehashIsCached(nil, codehash))

	// compare gas costs
	keccak := func() uint64 {
		tx := l2info.PrepareTxTo("Owner", &program, 1e9, nil, []byte{0x00})
		return ensure(tx, l2client.SendTransaction(ctx, tx)).GasUsedForL2()
	}
	ensure(mock.EvictProgram(&userAuth, program))
	miss := keccak()
	ensure(mock.CacheProgram(&userAuth, program))
	hits := keccak()
	cost, err := arbWasm.ProgramInitGas(nil, program)
	assert(hits-cost.GasWhenCached == miss-cost.Gas, err)

	// check logs
	empty := len(ensure(mock.CacheProgram(&userAuth, program)).Logs)
	evict := parseLog(ensure(mock.EvictProgram(&userAuth, program)).Logs[0])
	cache := parseLog(ensure(mock.CacheProgram(&userAuth, program)).Logs[0])
	assert(empty == 0 && evict.Manager == manager && !evict.Cached && cache.Codehash == codehash && cache.Cached, nil)

	// check ownership
	assert(arbOwner.IsChainOwner(nil, ownerAuth.From))
	ensure(arbWasmCache.EvictCodehash(&ownerAuth, codehash))
	ensure(arbWasmCache.CacheProgram(&ownerAuth, program))

	// de-authorize manager
	ensure(arbOwner.RemoveWasmCacheManager(&ownerAuth, manager))
	denytx(mock.EvictProgram(&userAuth, program))
	assert(arbWasmCache.CodehashIsCached(nil, codehash))
	all, err = arbWasmCache.AllCacheManagers(nil)
	assert(len(all) == 0, err)
}

func testReturnDataCost(t *testing.T, arbosVersion uint64) {
	builder, auth, cleanup := setupProgramTest(t, false, func(b *NodeBuilder) { b.WithArbOSVersion(arbosVersion) })
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	// use a consistent ink price
	arbOwner, err := pgen.NewArbOwner(types.ArbOwnerAddress, l2client)
	Require(t, err)
	tx, err := arbOwner.SetInkPrice(&auth, 10000)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, builder.L2.Client, tx)
	Require(t, err)

	returnSize := big.NewInt(1024 * 1024) // 1MiB
	returnSizeBytes := arbmath.U256Bytes(returnSize)

	testCall := func(to common.Address) uint64 {
		msg := ethereum.CallMsg{
			To:             &to,
			Data:           returnSizeBytes,
			SkipL1Charging: true,
		}
		ret, err := l2client.CallContract(ctx, msg, nil)
		Require(t, err)

		if !arbmath.BigEquals(big.NewInt(int64(len(ret))), returnSize) {
			Fatal(t, "unexpected return length", len(ret), "expected", returnSize)
		}

		gas, err := l2client.EstimateGas(ctx, msg)
		Require(t, err)

		return gas
	}

	stylusReturnSizeAddr := deployWasm(t, ctx, auth, l2client, watFile("return-size"))

	stylusGas := testCall(stylusReturnSizeAddr)

	// PUSH32 [returnSizeBytes]
	evmBytecode := append([]byte{0x7F}, returnSizeBytes...)
	// PUSH0 RETURN
	evmBytecode = append(evmBytecode, 0x5F, 0xF3)
	evmReturnSizeAddr := deployContract(t, ctx, auth, l2client, evmBytecode)

	evmGas := testCall(evmReturnSizeAddr)

	colors.PrintGrey(fmt.Sprintf("arbosVersion=%v stylusGas=%v evmGas=%v", arbosVersion, stylusGas, evmGas))
	// a bit of gas difference is expected due to EVM PUSH32 and PUSH0 cost (in practice this is 5 gas)
	similarGas := math.Abs(float64(stylusGas)-float64(evmGas)) <= 100
	if arbosVersion >= params.ArbosVersion_StylusFixes {
		if !similarGas {
			Fatal(t, "unexpected gas difference for return data: stylus", stylusGas, ", evm", evmGas)
		}
	} else if similarGas {
		Fatal(t, "gas unexpectedly similar for return data: stylus", stylusGas, ", evm", evmGas)
	}
}

func TestReturnDataCost(t *testing.T) {
	testReturnDataCost(t, params.ArbosVersion_Stylus)
	testReturnDataCost(t, params.ArbosVersion_StylusFixes)
}

func setupProgramTest(t *testing.T, jit bool, builderOpts ...func(*NodeBuilder)) (
	*NodeBuilder, bind.TransactOpts, func(),
) {
	ctx, cancel := context.WithCancel(context.Background())

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)

	for _, opt := range builderOpts {
		opt(builder)
	}

	// setupProgramTest is being called by tests that validate blocks.
	// For now validation only works with HashScheme set.
	builder.execConfig.Caching.StateScheme = rawdb.HashScheme
	builder.nodeConfig.BlockValidator.Enable = false
	builder.nodeConfig.Staker.Enable = true
	builder.nodeConfig.BatchPoster.Enable = true
	builder.nodeConfig.ParentChainReader.Enable = true
	builder.nodeConfig.ParentChainReader.OldHeaderTimeout = 10 * time.Minute

	valConf := valnode.TestValidationConfig
	valConf.UseJit = jit
	_, valStack := createTestValidationNode(t, ctx, &valConf)
	configByValidationNode(builder.nodeConfig, valStack)

	builder.execConfig.Sequencer.MaxRevertGasReject = 0

	builderCleanup := builder.Build(t)

	cleanup := func() {
		builderCleanup()
		cancel()
	}

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	arbOwner, err := pgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	arbDebug, err := pgen.NewArbDebug(types.ArbDebugAddress, builder.L2.Client)
	Require(t, err)

	ensure := func(tx *types.Transaction, err error) *types.Receipt {
		t.Helper()
		Require(t, err)
		receipt, err := EnsureTxSucceeded(ctx, builder.L2.Client, tx)
		Require(t, err)
		return receipt
	}

	// Set random pricing params
	inkPrice := testhelpers.RandomUint32(1, 20000) // evm to ink
	colors.PrintGrey(fmt.Sprintf("ink price=%d", inkPrice))

	ensure(arbDebug.BecomeChainOwner(&auth))
	ensure(arbOwner.SetInkPrice(&auth, inkPrice))
	return builder, auth, cleanup
}

func readWasmFile(t *testing.T, file string) ([]byte, []byte) {
	t.Helper()
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	source, err := os.ReadFile(file)
	Require(t, err)

	// chose a random dictionary for testing, but keep the same files consistent
	// #nosec G115
	randDict := arbcompress.Dictionary((len(file) + len(t.Name())) % 2)

	wasmSource, err := programs.Wat2Wasm(source)
	Require(t, err)
	wasm, err := arbcompress.Compress(wasmSource, arbcompress.LEVEL_WELL, randDict)
	Require(t, err)

	toKb := func(data []byte) float64 { return float64(len(data)) / 1024.0 }
	colors.PrintGrey(fmt.Sprintf("%v: len %.2fK vs %.2fK", name, toKb(wasm), toKb(wasmSource)))

	wasm = append(state.NewStylusPrefix(byte(randDict)), wasm...)
	return wasm, wasmSource
}

func deployWasm(
	t *testing.T, ctx context.Context, auth bind.TransactOpts, l2client *ethclient.Client, file string,
) common.Address {
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	wasm, _ := readWasmFile(t, file)
	auth.GasLimit = 32000000 // skip gas estimation
	program := deployContract(t, ctx, auth, l2client, wasm)
	colors.PrintGrey(name, ": deployed to ", program.Hex())
	activateWasm(t, ctx, auth, l2client, program, name)
	return program
}

func activateWasm(
	t *testing.T,
	ctx context.Context,
	auth bind.TransactOpts,
	l2client *ethclient.Client,
	program common.Address,
	name string,
) {
	arbWasm, err := pgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)

	timed(t, "activate "+name, func() {
		auth.Value = oneEth
		tx, err := arbWasm.ActivateProgram(&auth, program)
		Require(t, err)
		_, err = EnsureTxSucceeded(ctx, l2client, tx)
		Require(t, err)
	})
}

func argsForStorageRead(key common.Hash) []byte {
	args := []byte{0x00}
	args = append(args, key[:]...)
	return args
}

func argsForStorageWrite(key, value common.Hash) []byte {
	args := []byte{0x01}
	args = append(args, key[:]...)
	args = append(args, value[:]...)
	return args
}

func argsForMulticall(opcode vm.OpCode, address common.Address, value *big.Int, calldata []byte) []byte {
	kinds := make(map[vm.OpCode]byte)
	kinds[vm.CALL] = 0x00
	kinds[vm.DELEGATECALL] = 0x01
	kinds[vm.STATICCALL] = 0x02

	args := []byte{0x01}
	length := 21 + len(calldata)
	if opcode == vm.CALL {
		length += 32
	}
	// #nosec G115
	args = append(args, arbmath.Uint32ToBytes(uint32(length))...)
	args = append(args, kinds[opcode])
	if opcode == vm.CALL {
		if value == nil {
			value = common.Big0
		}
		args = append(args, common.BigToHash(value).Bytes()...)
	}
	args = append(args, address.Bytes()...)
	args = append(args, calldata...)
	return args
}

func multicallAppend(calls []byte, opcode vm.OpCode, address common.Address, inner []byte) []byte {
	calls[0] += 1 // add another call
	calls = append(calls, argsForMulticall(opcode, address, nil, inner)[1:]...)
	return calls
}

func multicallEmptyArgs() []byte {
	return []byte{0} // number of actions
}

func multicallAppendStore(args []byte, key, value common.Hash, emitLog bool) []byte {
	var action byte = 0x10
	if emitLog {
		action |= 0x08
	}
	args[0] += 1
	args = binary.BigEndian.AppendUint32(args, 1+64) // length
	args = append(args, action)
	args = append(args, key.Bytes()...)
	args = append(args, value.Bytes()...)
	return args
}

func multicallAppendLoad(args []byte, key common.Hash, emitLog bool) []byte {
	var action byte = 0x11
	if emitLog {
		action |= 0x08
	}
	args[0] += 1
	args = binary.BigEndian.AppendUint32(args, 1+32) // length
	args = append(args, action)
	args = append(args, key.Bytes()...)
	return args
}

func assertStorageAt(
	t *testing.T, ctx context.Context, l2client *ethclient.Client, contract common.Address, key, value common.Hash,
) {
	t.Helper()
	storedBytes, err := l2client.StorageAt(ctx, contract, key, nil)
	Require(t, err)
	storedValue := common.BytesToHash(storedBytes)
	if value != storedValue {
		Fatal(t, "wrong value", value, storedValue)
	}
}

func rustFile(name string) string {
	return fmt.Sprintf("../arbitrator/stylus/tests/%v/target/wasm32-unknown-unknown/release/%v.wasm", name, name)
}

func watFile(name string) string {
	return fmt.Sprintf("../arbitrator/stylus/tests/%v.wat", name)
}

func waitForSequencer(t *testing.T, builder *NodeBuilder, block uint64) {
	t.Helper()
	msgCount := arbutil.BlockNumberToMessageCount(block, 0)
	doUntil(t, 20*time.Millisecond, 500, func() bool {
		batchCount, err := builder.L2.ConsensusNode.InboxTracker.GetBatchCount()
		Require(t, err)
		meta, err := builder.L2.ConsensusNode.InboxTracker.GetBatchMetadata(batchCount - 1)
		Require(t, err)
		msgExecuted, err := builder.L2.ExecNode.ExecEngine.HeadMessageNumber()
		Require(t, err)
		return msgExecuted+1 >= msgCount && meta.MessageCount >= msgCount
	})
}

func timed(t *testing.T, message string, lambda func()) {
	t.Helper()
	now := time.Now()
	lambda()
	passed := time.Since(now)
	colors.PrintGrey("Time to ", message, ": ", passed.String())
}

func formatTime(duration time.Duration) string {
	span := float64(duration.Nanoseconds())
	unit := 0
	units := []string{"ns", "μs", "ms", "s", "min", "h", "d", "w", "mo", "yr", "dec", "cent", "mill", "eon"}
	scale := []float64{1000., 1000., 1000., 60., 60., 24., 7., 4.34, 12., 10., 10., 10., 1000000.}
	for span >= scale[unit] && unit < len(scale) {
		span /= scale[unit]
		unit += 1
	}
	return fmt.Sprintf("%.2f%s", span, units[unit])
}

func testWasmRecreate(t *testing.T, builder *NodeBuilder, storeTx *types.Transaction, loadTx *types.Transaction, want []byte) {
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client

	// do an onchain call - store value
	Require(t, l2client.SendTransaction(ctx, storeTx))
	_, err := EnsureTxSucceeded(ctx, l2client, storeTx)
	Require(t, err)

	testDir := t.TempDir()
	nodeBStack := testhelpers.CreateStackConfigForTest(testDir)
	nodeB, cleanupB := builder.Build2ndNode(t, &SecondNodeParams{stackConfig: nodeBStack})

	_, err = EnsureTxSucceeded(ctx, nodeB.Client, storeTx)
	Require(t, err)

	// make sure reading 2nd value succeeds from 2nd node
	result, err := arbutil.SendTxAsCall(ctx, nodeB.Client, loadTx, l2info.GetAddress("Owner"), nil, true)
	Require(t, err)
	if !bytes.Equal(result, want) {
		t.Fatalf("got wrong value, got %x, want %x", result, want)
	}
	// close nodeB
	cleanupB()

	// delete wasm dir of nodeB

	wasmPath := filepath.Join(testDir, "system_tests.test", "wasm")
	dirContents, err := os.ReadDir(wasmPath)
	Require(t, err)
	if len(dirContents) == 0 {
		Fatal(t, "not contents found before delete")
	}
	os.RemoveAll(wasmPath)

	// recreate nodeB - using same source dir (wasm deleted)
	nodeB, cleanupB = builder.Build2ndNode(t, &SecondNodeParams{stackConfig: nodeBStack})

	// test nodeB - sees existing transaction
	_, err = EnsureTxSucceeded(ctx, nodeB.Client, storeTx)
	Require(t, err)

	// test nodeB - answers eth_call (requires reloading wasm)
	result, err = arbutil.SendTxAsCall(ctx, nodeB.Client, loadTx, l2info.GetAddress("Owner"), nil, true)
	Require(t, err)
	if !bytes.Equal(result, want) {
		t.Fatalf("got wrong value, got %x, want %x", result, want)
	}

	// send new tx (requires wasm) and check nodeB sees it as well
	Require(t, l2client.SendTransaction(ctx, loadTx))

	_, err = EnsureTxSucceeded(ctx, l2client, loadTx)
	Require(t, err)

	_, err = EnsureTxSucceeded(ctx, nodeB.Client, loadTx)
	Require(t, err)

	cleanupB()
	dirContents, err = os.ReadDir(wasmPath)
	Require(t, err)
	if len(dirContents) == 0 {
		Fatal(t, "not contents found before delete")
	}
	os.RemoveAll(wasmPath)
}

func TestWasmRecreate(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	storage := deployWasm(t, ctx, auth, l2client, rustFile("storage"))

	zero := common.Hash{}
	val := common.HexToHash("0x121233445566")

	storeTx := l2info.PrepareTxTo("Owner", &storage, l2info.TransferGas, nil, argsForStorageWrite(zero, val))
	loadTx := l2info.PrepareTxTo("Owner", &storage, l2info.TransferGas, nil, argsForStorageRead(zero))

	testWasmRecreate(t, builder, storeTx, loadTx, val[:])
}

func TestWasmRecreateWithDelegatecall(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	storage := deployWasm(t, ctx, auth, l2client, rustFile("storage"))
	multicall := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))

	zero := common.Hash{}
	val := common.HexToHash("0x121233445566")

	data := argsForMulticall(vm.DELEGATECALL, storage, big.NewInt(0), argsForStorageWrite(zero, val))
	storeTx := l2info.PrepareTxTo("Owner", &multicall, l2info.TransferGas, nil, data)

	data = argsForMulticall(vm.DELEGATECALL, storage, big.NewInt(0), argsForStorageRead(zero))
	loadTx := l2info.PrepareTxTo("Owner", &multicall, l2info.TransferGas, nil, data)

	testWasmRecreate(t, builder, storeTx, loadTx, val[:])
}

// createMapFromDb is used in verifying if wasm store rebuilding works
func createMapFromDb(db ethdb.KeyValueStore) (map[string][]byte, error) {
	iter := db.NewIterator(nil, nil)
	defer iter.Release()

	dataMap := make(map[string][]byte)

	for iter.Next() {
		key := iter.Key()
		value := iter.Value()

		dataMap[string(key)] = value
	}

	if err := iter.Error(); err != nil {
		return nil, fmt.Errorf("iterator error: %w", err)
	}

	return dataMap, nil
}

func TestWasmStoreRebuilding(t *testing.T) {
	builder, auth, cleanup := setupProgramTest(t, true, func(b *NodeBuilder) {
		b.WithExtraArchs(allWasmTargets)
	})
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	storage := deployWasm(t, ctx, auth, l2client, rustFile("storage"))

	zero := common.Hash{}
	val := common.HexToHash("0x121233445566")

	// do an onchain call - store value
	storeTx := l2info.PrepareTxTo("Owner", &storage, l2info.TransferGas, nil, argsForStorageWrite(zero, val))
	Require(t, l2client.SendTransaction(ctx, storeTx))
	_, err := EnsureTxSucceeded(ctx, l2client, storeTx)
	Require(t, err)

	testDir := t.TempDir()
	nodeBStack := testhelpers.CreateStackConfigForTest(testDir)
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

	storeMap, err := createMapFromDb(wasmDb)
	Require(t, err)

	checkWasmStoreContent(t, wasmDb, builder.execConfig.StylusTarget.ExtraArchs, 1)
	// close nodeB
	cleanupB()

	// delete wasm dir of nodeB
	wasmPath := filepath.Join(testDir, "system_tests.test", "wasm")
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
	execConfig := nodeB.ExecNode.ConfigFetcher()
	Require(t, gethexec.RebuildWasmStore(ctx, wasmDbAfterDelete, nodeB.ExecNode.ChainDB, execConfig.RPC.MaxRecreateStateDepth, &execConfig.StylusTarget, bc, common.Hash{}, bc.CurrentBlock().Hash()))

	wasmDbAfterRebuild := nodeB.ExecNode.Backend.ArbInterface().BlockChain().StateCache().WasmStore()

	// Before comparing, check if rebuilding was set to done and then delete the keys that are used to track rebuilding status
	status, err := gethexec.ReadFromKeyValueStore[common.Hash](wasmDbAfterRebuild, gethexec.RebuildingPositionKey)
	Require(t, err)
	if status != gethexec.RebuildingDone {
		Fatal(t, "rebuilding was not set to done after successful completion")
	}
	Require(t, wasmDbAfterRebuild.Delete(gethexec.RebuildingPositionKey))
	Require(t, wasmDbAfterRebuild.Delete(gethexec.RebuildingStartBlockHashKey))

	rebuiltStoreMap, err := createMapFromDb(wasmDbAfterRebuild)
	Require(t, err)

	// Check if rebuilding worked
	if len(storeMap) != len(rebuiltStoreMap) {
		Fatal(t, "size mismatch while rebuilding wasm store:", "want", len(storeMap), "got", len(rebuiltStoreMap))
	}
	for key, value1 := range storeMap {
		value2, exists := rebuiltStoreMap[key]
		if !exists {
			Fatal(t, "rebuilt wasm store doesn't have key from original")
		}
		if !bytes.Equal(value1, value2) {
			Fatal(t, "rebuilt wasm store has incorrect value from original")
		}
	}

	checkWasmStoreContent(t, wasmDbAfterRebuild, builder.execConfig.StylusTarget.ExtraArchs, 1)
	cleanupB()
}

func readModuleHashes(t *testing.T, wasmDb ethdb.KeyValueStore) []common.Hash {
	modulesSet := make(map[common.Hash]struct{})
	asmPrefix := []byte{0x00, 'w'}
	it := wasmDb.NewIterator(asmPrefix, nil)
	defer it.Release()
	for it.Next() {
		key := it.Key()
		if len(key) != rawdb.WasmKeyLen {
			t.Fatalf("unexpected activated module key length, len: %d, key: %v", len(key), key)
		}
		moduleHash := key[rawdb.WasmPrefixLen:]
		if len(moduleHash) != common.HashLength {
			t.Fatalf("Invalid moduleHash length in key: %v, moduleHash: %v", key, moduleHash)
		}
		modulesSet[common.BytesToHash(moduleHash)] = struct{}{}
	}
	modules := make([]common.Hash, 0, len(modulesSet))
	for module := range modulesSet {
		modules = append(modules, module)
	}
	return modules
}

func checkWasmStoreContent(t *testing.T, wasmDb ethdb.KeyValueStore, targets []string, numModules int) {
	modules := readModuleHashes(t, wasmDb)
	if len(modules) != numModules {
		t.Fatalf("Unexpected number of module hashes found in wasm store, want: %d, have: %d", numModules, len(modules))
	}
	for _, module := range modules {
		for _, target := range targets {
			wasmTarget := ethdb.WasmTarget(target)
			if !rawdb.IsSupportedWasmTarget(wasmTarget) {
				t.Fatalf("internal test error - unsupported target passed to checkWasmStoreContent: %v", target)
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("Failed to read activated asm for target: %v, module: %v", target, module)
					}
				}()
				_ = rawdb.ReadActivatedAsm(wasmDb, wasmTarget, module)
			}()
		}
	}
}
