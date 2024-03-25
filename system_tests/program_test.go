// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbtest

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/nitro/arbcompress"
	"github.com/offchainlabs/nitro/arbos/programs"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
	"github.com/offchainlabs/nitro/validator/valnode"
	"github.com/wasmerio/wasmer-go/wasmer"
)

var oneEth = arbmath.UintToBig(1e18)

func TestProgramKeccak(t *testing.T) {
	t.Parallel()
	keccakTest(t, true)
}

func keccakTest(t *testing.T, jit bool) {
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()
	programAddress := deployWasm(t, ctx, auth, l2client, rustFile("keccak"))

	wasm, _ := readWasmFile(t, rustFile("keccak"))
	otherAddressSameCode := deployContract(t, ctx, auth, l2client, wasm)
	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)

	colors.PrintBlue("program deployed to ", programAddress.Hex())
	timed(t, "activate same code", func() {
		if _, err := arbWasm.ActivateProgram(&auth, otherAddressSameCode); err == nil || !strings.Contains(err.Error(), "ProgramUpToDate") {
			Fatal(t, "activate should have failed with ProgramUpToDate", err)
		}
	})

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
	testActivateTwice(t, true)
}

func testActivateTwice(t *testing.T, jit bool) {
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

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, l2client)
	Require(t, err)
	ensure(arbOwner.SetInkPrice(&auth, 1))

	wasm, _ := readWasmFile(t, rustFile("keccak"))
	keccakA := deployContract(t, ctx, auth, l2client, wasm)
	keccakB := deployContract(t, ctx, auth, l2client, wasm)

	colors.PrintBlue("keccak program A deployed to ", keccakA)
	colors.PrintBlue("keccak program B deployed to ", keccakB)

	multiAddr := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))
	preimage := []byte("it's time to du-du-du-du d-d-d-d-d-d-d de-duplicate")

	keccakArgs := []byte{0x01} // keccak the preimage once
	keccakArgs = append(keccakArgs, preimage...)

	checkReverts := func() {
		msg := ethereum.CallMsg{
			To:    &keccakA,
			Value: big.NewInt(0),
			Data:  keccakArgs,
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

	// mechanisms for creating calldata
	activateProgram, _ := util.NewCallParser(precompilesgen.ArbWasmABI, "activateProgram")
	legacyError, _ := util.NewCallParser(precompilesgen.ArbDebugABI, "legacyError")
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

	// Activate keccak program A, then call into B, which should succeed due to being the same codehash
	args = argsForMulticall(vm.CALL, types.ArbWasmAddress, oneEth, pack(activateProgram(keccakA)))
	args = multicallAppend(args, vm.CALL, mockAddr, pack(callKeccak(keccakB, keccakArgs)))

	tx = l2info.PrepareTxTo("Owner", &multiAddr, 1e9, oneEth, args)
	ensure(tx, l2client.SendTransaction(ctx, tx))

	validateBlocks(t, 7, jit, builder)
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
			To:    &to,
			Value: big.NewInt(0),
			Data:  data,
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
	burnArbGas, _ := util.NewCallParser(precompilesgen.ArbosTestABI, "burnArbGas")
	customRevert, _ := util.NewCallParser(precompilesgen.ArbDebugABI, "customRevert")
	legacyError, _ := util.NewCallParser(precompilesgen.ArbDebugABI, "legacyError")
	callKeccak, _ := util.NewCallParser(mocksgen.ProgramTestABI, "callKeccak")
	pack := func(data []byte, err error) []byte {
		Require(t, err)
		return data
	}

	colors.PrintBlue("Calling the ArbosTest precompile (Rust => precompile)")
	testPrecompile := func(gas uint64) uint64 {
		// Call the burnArbGas() precompile from Rust
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
	testLogs(t, true)
}

func testLogs(t *testing.T, jit bool) {
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()
	logAddr := deployWasm(t, ctx, auth, l2client, rustFile("log"))
	multiAddr := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))

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
		args := encode(topics, data)
		tx := l2info.PrepareTxTo("Owner", &logAddr, 1e9, nil, args)
		receipt := ensure(tx, l2client.SendTransaction(ctx, tx))

		if len(receipt.Logs) != 1 {
			Fatal(t, "wrong number of logs", len(receipt.Logs))
		}
		log := receipt.Logs[0]
		if !bytes.Equal(log.Data, data) {
			Fatal(t, "data mismatch", log.Data, data)
		}
		if len(log.Topics) != len(topics) {
			Fatal(t, "topics mismatch", len(log.Topics), len(topics))
		}
		for j := 0; j < i; j++ {
			if log.Topics[j] != topics[j] {
				Fatal(t, "topic mismatch", log.Topics, topics)
			}
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
		arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
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

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, l2client)
	Require(t, err)
	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)

	ensure(arbOwner.SetInkPrice(&auth, 1e4))
	ensure(arbOwner.SetMaxTxGasLimit(&auth, 34000000))

	memoryAddr := deployWasm(t, ctx, auth, l2client, watFile("memory"))
	multiAddr := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))
	growCallAddr := deployWasm(t, ctx, auth, l2client, watFile("grow-and-call"))

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

	model := programs.NewMemoryModel(2, 1000)

	// expand to 128 pages, retract, then expand again to 128.
	//   - multicall takes 1 page to init, and then 1 more at runtime.
	//   - grow-and-call takes 1 page, then grows to the first arg by second arg steps.
	args := argsForMulticall(vm.CALL, memoryAddr, nil, []byte{126, 50})
	args = multicallAppend(args, vm.CALL, memoryAddr, []byte{126, 80})

	tx := l2info.PrepareTxTo("Owner", &multiAddr, 1e9, nil, args)
	receipt := ensure(tx, l2client.SendTransaction(ctx, tx))
	gasCost := receipt.GasUsedForL2()
	memCost := model.GasCost(128, 0, 0) + model.GasCost(126, 2, 128)
	logical := uint64(32000000 + 126*1000)
	if !arbmath.WithinRange(gasCost, memCost, memCost+2e5) || !arbmath.WithinRange(gasCost, logical, logical+2e5) {
		Fatal(t, "unexpected cost", gasCost, memCost)
	}

	// check that we'd normally run out of gas
	ensure(arbOwner.SetMaxTxGasLimit(&auth, 32000000))
	expectFailure(multiAddr, args, oneEth)

	// check that activation fails when out of memory
	wasm, _ := readWasmFile(t, watFile("grow-120"))
	growHugeAddr := deployContract(t, ctx, auth, l2client, wasm)
	colors.PrintGrey("memory.wat        ", memoryAddr)
	colors.PrintGrey("multicall.rs      ", multiAddr)
	colors.PrintGrey("grow-and-call.wat ", growCallAddr)
	colors.PrintGrey("grow-120.wat      ", growHugeAddr)
	activate, _ := util.NewCallParser(precompilesgen.ArbWasmABI, "activateProgram")
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

	validateBlocks(t, 2, jit, builder)
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

	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
	Require(t, err)

	badExportWasm, _ := readWasmFile(t, watFile("bad-export"))
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
		trieHash := func(addr common.Address) common.Hash {
			stateObject := statedb.GetOrNewStateObject(addr)
			return stateObject.Root()
			// .StorageTrie(addr)
		}

		solTrie := trieHash(solidity)
		rustTrie := trieHash(rust)
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
	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
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

func setupProgramTest(t *testing.T, jit bool) (
	*NodeBuilder, bind.TransactOpts, func(),
) {
	ctx, cancel := context.WithCancel(context.Background())

	builder := NewNodeBuilder(ctx).DefaultConfig(t, true)

	builder.nodeConfig.BlockValidator.Enable = false
	builder.nodeConfig.Staker.Enable = true
	builder.nodeConfig.BatchPoster.Enable = true
	builder.nodeConfig.ParentChainReader.Enable = true
	builder.nodeConfig.ParentChainReader.OldHeaderTimeout = 10 * time.Minute

	valConf := valnode.TestValidationConfig
	valConf.UseJit = jit
	_, valStack := createTestValidationNode(t, ctx, &valConf)
	configByValidationNode(t, builder.nodeConfig, valStack)

	builder.execConfig.Sequencer.MaxRevertGasReject = 0

	builderCleanup := builder.Build(t)

	cleanup := func() {
		builderCleanup()
		cancel()
	}

	auth := builder.L2Info.GetDefaultTransactOpts("Owner", ctx)

	arbOwner, err := precompilesgen.NewArbOwner(types.ArbOwnerAddress, builder.L2.Client)
	Require(t, err)
	arbDebug, err := precompilesgen.NewArbDebug(types.ArbDebugAddress, builder.L2.Client)
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
	name := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	source, err := os.ReadFile(file)
	Require(t, err)

	// chose a random dictionary for testing, but keep the same files consistent
	randDict := arbcompress.Dictionary((len(file) + len(t.Name())) % 2)

	wasmSource, err := wasmer.Wat2Wasm(string(source))
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
	arbWasm, err := precompilesgen.NewArbWasm(types.ArbWasmAddress, l2client)
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
