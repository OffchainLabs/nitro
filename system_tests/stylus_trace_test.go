// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbtest

import (
	"bytes"
	"encoding/binary"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/tracers/logger"
	"github.com/holiman/uint256"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/solgen/go/mocksgen"
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/colors"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

var skipCheck = []byte("skip")

func checkOpcode(t *testing.T, result logger.ExecutionResult, index int, wantOp vm.OpCode, wantStack ...[]byte) {
	CheckEqual(t, wantOp.String(), result.StructLogs[index].Op)
	CheckEqual(t, len(wantStack), len(*result.StructLogs[index].Stack))

	// reverse stack to canonical order
	for i, j := 0, len(wantStack)-1; i < j; i, j = i+1, j-1 {
		wantStack[i], wantStack[j] = wantStack[j], wantStack[i]

	}

	for i, wantBytes := range wantStack {
		if !bytes.Equal(wantBytes, skipCheck) {
			wantVal := uint256.NewInt(0).SetBytes(wantBytes).Hex()
			CheckEqual(t, wantVal, (*result.StructLogs[index].Stack)[i])
		}
	}
}

func sendAndTraceTransaction(
	t *testing.T,
	builder *NodeBuilder,
	program common.Address,
	value *big.Int,
	data []byte,
) logger.ExecutionResult {
	ctx := builder.ctx
	l2client := builder.L2.Client
	l2info := builder.L2Info
	rpcClient := builder.L2.ConsensusNode.Stack.Attach()

	tx := l2info.PrepareTxTo("Owner", &program, l2info.TransferGas, value, data)
	err := l2client.SendTransaction(ctx, tx)
	Require(t, err)

	var result logger.ExecutionResult
	err = rpcClient.CallContext(ctx, &result, "debug_traceTransaction", tx.Hash(), nil)
	Require(t, err, "failed to trace call")

	colors.PrintGrey("Call trace:")
	colors.PrintGrey("i\tdepth\topcode\tstack")
	for i, log := range result.StructLogs {
		if log.Stack == nil {
			stack := []string{}
			log.Stack = &stack
		}
		colors.PrintGrey(i, "\t", log.Depth, "\t", log.Op, "\t", *log.Stack)
	}

	return result
}

func intToBytes(v int) []byte {
	return binary.BigEndian.AppendUint64(nil, uint64(v))
}

func TestStylusTraceStorage(t *testing.T) {
	const jit = false
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	program := deployWasm(t, ctx, auth, l2client, rustFile("storage"))

	key := testhelpers.RandomHash()
	value := testhelpers.RandomHash()

	trans := func(data []byte) []byte {
		data[0] += 2
		return data
	}

	// storage_cache_bytes32
	result := sendAndTraceTransaction(t, builder, program, nil, argsForStorageWrite(key, value))
	checkOpcode(t, result, 3, vm.SSTORE, key[:], value[:])

	// storage_load_bytes32
	result = sendAndTraceTransaction(t, builder, program, nil, argsForStorageRead(key))
	checkOpcode(t, result, 3, vm.SLOAD, key[:])
	checkOpcode(t, result, 4, vm.POP, value[:])

	// transient_store_bytes32
	result = sendAndTraceTransaction(t, builder, program, nil, trans(argsForStorageWrite(key, value)))
	checkOpcode(t, result, 3, vm.TSTORE, key[:], value[:])

	// transient_load_bytes32
	result = sendAndTraceTransaction(t, builder, program, nil, trans(argsForStorageRead(key)))
	checkOpcode(t, result, 3, vm.TLOAD, key[:])
	checkOpcode(t, result, 4, vm.POP, nil)
}

func TestStylusTraceNativeKeccak(t *testing.T) {
	const jit = false
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	program := deployWasm(t, ctx, auth, l2client, watFile("timings/keccak"))

	args := binary.LittleEndian.AppendUint32(nil, 1) // rounds
	args = append(args, testhelpers.RandomSlice(123)...)
	hash := crypto.Keccak256Hash(args) // the keccak.wat program computes the hash of the whole args

	// native_keccak256
	result := sendAndTraceTransaction(t, builder, program, nil, args)
	checkOpcode(t, result, 3, vm.KECCAK256, nil, intToBytes(len(args)))
	checkOpcode(t, result, 4, vm.POP, hash[:])
}

func TestStylusTraceMath(t *testing.T) {
	const jit = false
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	program := deployWasm(t, ctx, auth, l2client, rustFile("math"))
	result := sendAndTraceTransaction(t, builder, program, nil, nil)

	value := common.Hex2Bytes("eddecf107b5740cef7f5a01e3ea7e287665c4e75a8eb6afae2fda2e3d4367786")
	unknown := common.Hex2Bytes("c6178c2de1078cd36c3bd302cde755340d7f17fcb3fcc0b9c333ba03b217029f")
	ed25519 := common.Hex2Bytes("fffffffffffffffffffffffffffffffffffffffffffffffffffffffefffffc2f")
	results := [][]byte{
		common.Hex2Bytes("b28a98598473836430b84078e55690d279cca19b9922f248c6a6ad6588d12494"),
		common.Hex2Bytes("265b7ffdc26469bd58409a734987e66a5ece71a2312970d5403f395d24a31b85"),
		common.Hex2Bytes("00000000000000002947e87fd2cf7e1eacd01ef1286c0d795168d90db4fc5bb3"),
		common.Hex2Bytes("c4b1cfcc1423392b29d826de0b3779a096d543ad2b71f34aa4596bd97f493fbb"),
		common.Hex2Bytes("00000000000000000000000000000000000000000000000015d41b922f2eafc5"),
	}

	// math_mul_mod
	checkOpcode(t, result, 3, vm.MULMOD, value, unknown, ed25519)
	checkOpcode(t, result, 4, vm.POP, results[0])

	// math_add_mod
	checkOpcode(t, result, 5, vm.ADDMOD, results[0], ed25519, unknown)
	checkOpcode(t, result, 6, vm.POP, results[1])

	// math_div
	checkOpcode(t, result, 7, vm.DIV, results[1], value[:8])
	checkOpcode(t, result, 8, vm.POP, results[2])

	// math_pow
	checkOpcode(t, result, 9, vm.EXP, results[2], ed25519[24:32])
	checkOpcode(t, result, 10, vm.POP, results[3])

	// math_mod
	checkOpcode(t, result, 11, vm.MOD, results[3], unknown[:8])
	checkOpcode(t, result, 12, vm.POP, results[4])
}

func TestStylusTraceExit(t *testing.T) {
	const jit = false
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	// normal exit with return value
	program := deployWasm(t, ctx, auth, l2client, rustFile("storage"))
	key := testhelpers.RandomHash()
	result := sendAndTraceTransaction(t, builder, program, nil, argsForStorageRead(key))
	size := intToBytes(32)
	checkOpcode(t, result, 5, vm.RETURN, nil, size)

	// stop with exit early
	program = deployWasm(t, ctx, auth, l2client, watFile("exit-early/exit-early"))
	result = sendAndTraceTransaction(t, builder, program, nil, nil)
	checkOpcode(t, result, 3, vm.STOP)

	// revert
	program = deployWasm(t, ctx, auth, l2client, watFile("exit-early/panic-after-write"))
	result = sendAndTraceTransaction(t, builder, program, nil, nil)
	size = intToBytes(len("execution reverted"))
	checkOpcode(t, result, 3, vm.REVERT, nil, size)
}

func TestStylusTraceEvmData(t *testing.T) {
	const jit = false
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2info := builder.L2Info
	l2client := builder.L2.Client
	defer cleanup()

	program := deployWasm(t, ctx, auth, l2client, rustFile("evm-data"))

	fundedAddr := l2info.GetAddress("Faucet")
	ethPrecompile := common.BigToAddress(big.NewInt(1))
	arbTestAddress := types.ArbosTestAddress
	burnArbGas, _ := util.NewCallParser(precompilesgen.ArbosTestABI, "burnArbGas")
	gasToBurn := uint64(1000000)
	callBurnData, err := burnArbGas(new(big.Int).SetUint64(gasToBurn))
	Require(t, err)

	data := []byte{}
	data = append(data, fundedAddr.Bytes()...)
	data = append(data, ethPrecompile.Bytes()...)
	data = append(data, arbTestAddress.Bytes()...)
	data = append(data, program.Bytes()...)
	data = append(data, callBurnData...)
	result := sendAndTraceTransaction(t, builder, program, nil, data)

	fundedBalance, err := l2client.BalanceAt(ctx, fundedAddr, nil)
	Require(t, err)
	programCode, err := l2client.CodeAt(ctx, program, nil)
	Require(t, err)
	programCodehash := crypto.Keccak256(programCode)
	owner := l2info.GetAddress("Owner")

	// read_args
	checkOpcode(t, result, 2, vm.CALLDATACOPY, nil, nil, intToBytes(len(data)))

	// account_balance
	checkOpcode(t, result, 3, vm.BALANCE, fundedAddr[:])
	checkOpcode(t, result, 4, vm.POP, fundedBalance.Bytes())

	// account_codehash
	checkOpcode(t, result, 9, vm.EXTCODEHASH, program[:])
	checkOpcode(t, result, 10, vm.POP, programCodehash)

	// account_code_size
	checkOpcode(t, result, 11, vm.EXTCODESIZE, program[:])
	checkOpcode(t, result, 12, vm.POP, intToBytes(len(programCode)))

	// account_code
	checkOpcode(t, result, 13, vm.EXTCODECOPY, program[:], nil, nil, intToBytes(len(programCode)))

	// block_basefee
	checkOpcode(t, result, 26, vm.BASEFEE)
	checkOpcode(t, result, 27, vm.POP, skipCheck)

	// chainid
	checkOpcode(t, result, 28, vm.CHAINID)
	checkOpcode(t, result, 29, vm.POP, intToBytes(412346))

	// block_coinbase
	checkOpcode(t, result, 30, vm.COINBASE)
	checkOpcode(t, result, 31, vm.POP, skipCheck)

	// block_gas_limit
	checkOpcode(t, result, 32, vm.GASLIMIT)
	checkOpcode(t, result, 33, vm.POP, skipCheck)

	// block_timestamp
	checkOpcode(t, result, 34, vm.TIMESTAMP)
	checkOpcode(t, result, 35, vm.POP, skipCheck)

	// contract_address
	checkOpcode(t, result, 36, vm.ADDRESS)
	checkOpcode(t, result, 37, vm.POP, program[:])

	// msg_sender
	checkOpcode(t, result, 38, vm.CALLER)
	checkOpcode(t, result, 39, vm.POP, owner[:])

	// msg_value
	checkOpcode(t, result, 40, vm.CALLVALUE)
	checkOpcode(t, result, 41, vm.POP, nil)

	// tx_origin
	checkOpcode(t, result, 42, vm.ORIGIN)
	checkOpcode(t, result, 43, vm.POP, owner[:])

	// tx_gas_price
	checkOpcode(t, result, 44, vm.GASPRICE)
	checkOpcode(t, result, 45, vm.POP, skipCheck)

	// tx_ink_price
	checkOpcode(t, result, 46, vm.GASPRICE)
	checkOpcode(t, result, 47, vm.POP, skipCheck)

	// block_number
	checkOpcode(t, result, 48, vm.NUMBER)
	checkOpcode(t, result, 49, vm.POP, skipCheck)

	// evm_gas_left
	checkOpcode(t, result, 50, vm.GAS)
	checkOpcode(t, result, 51, vm.POP, skipCheck)

	// evm_ink_left
	checkOpcode(t, result, 52, vm.GAS)
	checkOpcode(t, result, 53, vm.POP, skipCheck)
}

func TestStylusTraceLog(t *testing.T) {
	const jit = false
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	program := deployWasm(t, ctx, auth, l2client, rustFile("log"))

	const numTopics = 4
	const logSize = 123
	expectedStack := [][]byte{nil, intToBytes(logSize)}
	args := []byte{numTopics}
	for i := 0; i < numTopics; i++ {
		topic := testhelpers.RandomSlice(32)
		expectedStack = append(expectedStack, topic)
		args = append(args, topic...) // topic
	}
	args = append(args, testhelpers.RandomSlice(logSize)...) // log

	result := sendAndTraceTransaction(t, builder, program, nil, args)

	// emit_log
	checkOpcode(t, result, 3, vm.LOG4, expectedStack...)
}

func TestStylusTraceReturnDataSize(t *testing.T) {
	const jit = false
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	program := deployWasm(t, ctx, auth, l2client, watFile("timings/return_data_size"))
	args := binary.LittleEndian.AppendUint32(nil, 1) // rounds
	result := sendAndTraceTransaction(t, builder, program, nil, args)

	// return_data_size
	checkOpcode(t, result, 3, vm.RETURNDATASIZE)
	checkOpcode(t, result, 4, vm.POP, nil)
}

func TestStylusTraceCall(t *testing.T) {
	const jit = false
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	storage := deployWasm(t, ctx, auth, l2client, rustFile("storage"))
	multicall := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))
	key := testhelpers.RandomHash()
	gas := skipCheck
	innerArgs := argsForStorageRead(key)
	argsLen := intToBytes(len(innerArgs))
	returnLen := intToBytes(32)

	args := argsForMulticall(vm.CALL, storage, nil, innerArgs)
	args = multicallAppend(args, vm.DELEGATECALL, storage, innerArgs)
	args = multicallAppend(args, vm.STATICCALL, storage, innerArgs)
	result := sendAndTraceTransaction(t, builder, multicall, nil, args)

	// call_contract
	checkOpcode(t, result, 3, vm.CALL, gas, storage[:], nil, nil, argsLen, nil, nil)

	// read_return_data
	checkOpcode(t, result, 8, vm.RETURNDATACOPY, nil, nil, returnLen)

	// delegate_call_contract
	checkOpcode(t, result, 9, vm.DELEGATECALL, gas, storage[:], nil, argsLen, nil, nil)

	// static_call_contract
	checkOpcode(t, result, 15, vm.STATICCALL, gas, storage[:], nil, argsLen, nil, nil)
}

func TestStylusTraceCreate(t *testing.T) {
	const jit = false
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	program := deployWasm(t, ctx, auth, l2client, rustFile("create"))

	deployWasm, _ := readWasmFile(t, rustFile("storage"))
	deployCode := deployContractInitCode(deployWasm, false)
	startValue := testhelpers.RandomCallValue(1e5)
	salt := testhelpers.RandomHash()
	create1Addr := crypto.CreateAddress(program, 1)
	create2Addr := crypto.CreateAddress2(program, salt, crypto.Keccak256(deployCode))

	// create1
	create1Args := []byte{0x01}
	create1Args = append(create1Args, common.BigToHash(startValue).Bytes()...)
	create1Args = append(create1Args, deployCode...)
	result := sendAndTraceTransaction(t, builder, program, startValue, create1Args)
	checkOpcode(t, result, 10, vm.CREATE, startValue.Bytes(), nil, intToBytes(len(deployCode)))
	checkOpcode(t, result, 11, vm.POP, create1Addr[:])

	// create2
	create2Args := []byte{0x02}
	create2Args = append(create2Args, common.BigToHash(startValue).Bytes()...)
	create2Args = append(create2Args, salt[:]...)
	create2Args = append(create2Args, deployCode...)
	result = sendAndTraceTransaction(t, builder, program, startValue, create2Args)
	checkOpcode(t, result, 10, vm.CREATE2, startValue.Bytes(), nil, intToBytes(len(deployCode)), salt[:])
	checkOpcode(t, result, 11, vm.POP, create2Addr[:])
}

// TestStylusTraceEquivalence compares a Stylus trace with a equivalent Solidity/EVM trace. Notice
// the Stylus trace does not contain all opcodes from the Solidity/EVM trace. Instead, this test
// only checks that both traces contain the same basic opcodes.
func TestStylusTraceEquivalence(t *testing.T) {
	const jit = false
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	// Args for storing and loading from storage
	const (
		storageKind = 0x10
		storeAction = storageKind | 0x00
		loadAction  = storageKind | 0x01
		logModifier = 0x08
	)
	key := testhelpers.RandomHash()
	value := testhelpers.RandomHash()
	args := []byte{2} // number of actions
	// first action
	args = binary.BigEndian.AppendUint32(args, 1+64) // length
	args = append(args, storeAction|logModifier)
	args = append(args, key.Bytes()...)
	args = append(args, value.Bytes()...)
	// second action
	args = binary.BigEndian.AppendUint32(args, 1+32) // length
	args = append(args, loadAction|logModifier)
	args = append(args, key.Bytes()...)

	// Trace recursive call in wasm
	wasmMulticall := deployWasm(t, ctx, auth, l2client, rustFile("multicall"))
	colors.PrintGrey("wasm multicall deployed at ", wasmMulticall)
	wasmArgs := argsForMulticall(vm.CALL, wasmMulticall, nil, args)
	wasmResult := sendAndTraceTransaction(t, builder, wasmMulticall, nil, wasmArgs)

	// Trace recursive call in evm
	evmMulticall, tx, _, err := mocksgen.DeployMultiCallTest(&auth, builder.L2.Client)
	Require(t, err)
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)
	colors.PrintGrey("evm multicall deployed at ", evmMulticall)
	evmArgs := argsForMulticall(vm.CALL, evmMulticall, nil, args)
	evmResult := sendAndTraceTransaction(t, builder, evmMulticall, nil, evmArgs)

	// Check equivalence of opcodes
	_ = evmResult
	_ = wasmResult
}
