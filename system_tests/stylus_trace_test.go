// Copyright 2022-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package arbtest

import (
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
	"github.com/offchainlabs/nitro/solgen/go/precompilesgen"
	"github.com/offchainlabs/nitro/util/testhelpers"
)

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
	_, err = EnsureTxSucceeded(ctx, l2client, tx)
	Require(t, err)

	var result logger.ExecutionResult
	err = rpcClient.CallContext(ctx, &result, "debug_traceTransaction", tx.Hash(), nil)
	Require(t, err, "failed to trace call")

	for i, log := range result.StructLogs {
		if log.Stack == nil {
			stack := []string{}
			log.Stack = &stack
		}
		t.Log("Trace call: i =", i, "| OpCode =", log.Op, "| Stack =", *log.Stack)
	}

	return result
}

func checkOpcode(t *testing.T, result logger.ExecutionResult, index int, wantOp vm.OpCode, wantStackSize int) {
	CheckEqual(t, wantOp.String(), result.StructLogs[index].Op)
	CheckEqual(t, wantStackSize, len(*result.StructLogs[index].Stack))
}

func checkOpcodeStack(t *testing.T, result logger.ExecutionResult, index int, wantOp vm.OpCode, wantStack ...[]byte) {
	checkOpcode(t, result, index, wantOp, len(wantStack))

	// reverse stack to canonical order
	for i, j := 0, len(wantStack)-1; i < j; i, j = i+1, j-1 {
		wantStack[i], wantStack[j] = wantStack[j], wantStack[i]

	}

	for i, wantBytes := range wantStack {
		wantVal := uint256.NewInt(0).SetBytes(wantBytes).Hex()
		CheckEqual(t, wantVal, (*result.StructLogs[index].Stack)[i])
	}
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
	checkOpcodeStack(t, result, 3, vm.SSTORE, key[:], value[:])

	// storage_load_bytes32
	result = sendAndTraceTransaction(t, builder, program, nil, argsForStorageRead(key))
	checkOpcodeStack(t, result, 3, vm.SLOAD, key[:])
	checkOpcodeStack(t, result, 4, vm.POP, value[:])

	// transient_store_bytes32
	result = sendAndTraceTransaction(t, builder, program, nil, trans(argsForStorageWrite(key, value)))
	checkOpcodeStack(t, result, 3, vm.TSTORE, key[:], value[:])

	// transient_load_bytes32
	result = sendAndTraceTransaction(t, builder, program, nil, trans(argsForStorageRead(key)))
	checkOpcodeStack(t, result, 3, vm.TLOAD, key[:])
	checkOpcodeStack(t, result, 4, vm.POP, nil)
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
	checkOpcodeStack(t, result, 3, vm.KECCAK256, nil, intToBytes(len(args)))
	checkOpcodeStack(t, result, 4, vm.POP, hash[:])
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
	checkOpcodeStack(t, result, 3, vm.MULMOD, value, unknown, ed25519)
	checkOpcodeStack(t, result, 4, vm.POP, results[0])

	// math_add_mod
	checkOpcodeStack(t, result, 5, vm.ADDMOD, results[0], ed25519, unknown)
	checkOpcodeStack(t, result, 6, vm.POP, results[1])

	// math_div
	checkOpcodeStack(t, result, 7, vm.DIV, results[1], value[:8])
	checkOpcodeStack(t, result, 8, vm.POP, results[2])

	// math_pow
	checkOpcodeStack(t, result, 9, vm.EXP, results[2], ed25519[24:32])
	checkOpcodeStack(t, result, 10, vm.POP, results[3])

	// math_mod
	checkOpcodeStack(t, result, 11, vm.MOD, results[3], unknown[:8])
	checkOpcodeStack(t, result, 12, vm.POP, results[4])
}

func TestStylusTraceExitEarly(t *testing.T) {
	const jit = false
	builder, auth, cleanup := setupProgramTest(t, jit)
	ctx := builder.ctx
	l2client := builder.L2.Client
	defer cleanup()

	program := deployWasm(t, ctx, auth, l2client, watFile("exit-early/exit-early"))
	result := sendAndTraceTransaction(t, builder, program, nil, nil)

	// exit_early
	checkOpcodeStack(t, result, 3, vm.RETURN, nil, nil)
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
	checkOpcodeStack(t, result, 2, vm.CALLDATACOPY, nil, nil, intToBytes(len(data)))

	// account_balance
	checkOpcodeStack(t, result, 3, vm.BALANCE, fundedAddr[:])
	checkOpcodeStack(t, result, 4, vm.POP, fundedBalance.Bytes())

	// account_codehash
	checkOpcodeStack(t, result, 9, vm.EXTCODEHASH, program[:])
	checkOpcodeStack(t, result, 10, vm.POP, programCodehash)

	// account_code_size
	checkOpcodeStack(t, result, 11, vm.EXTCODESIZE, program[:])
	checkOpcodeStack(t, result, 12, vm.POP, intToBytes(len(programCode)))

	// account_code
	checkOpcodeStack(t, result, 13, vm.EXTCODECOPY, program[:], nil, nil, intToBytes(len(programCode)))

	// block_basefee
	checkOpcodeStack(t, result, 26, vm.BASEFEE)
	checkOpcode(t, result, 27, vm.POP, 1)

	// chainid
	checkOpcodeStack(t, result, 28, vm.CHAINID)
	checkOpcodeStack(t, result, 29, vm.POP, intToBytes(412346))

	// block_coinbase
	checkOpcodeStack(t, result, 30, vm.COINBASE)
	checkOpcode(t, result, 31, vm.POP, 1)

	// block_gas_limit
	checkOpcodeStack(t, result, 32, vm.GASLIMIT)
	checkOpcode(t, result, 33, vm.POP, 1)

	// block_timestamp
	checkOpcodeStack(t, result, 34, vm.TIMESTAMP)
	checkOpcode(t, result, 35, vm.POP, 1)

	// contract_address
	checkOpcodeStack(t, result, 36, vm.ADDRESS)
	checkOpcodeStack(t, result, 37, vm.POP, program[:])

	// msg_sender
	checkOpcodeStack(t, result, 38, vm.CALLER)
	checkOpcodeStack(t, result, 39, vm.POP, owner[:])

	// msg_value
	checkOpcodeStack(t, result, 40, vm.CALLVALUE)
	checkOpcodeStack(t, result, 41, vm.POP, nil)

	// tx_origin
	checkOpcodeStack(t, result, 42, vm.ORIGIN)
	checkOpcodeStack(t, result, 43, vm.POP, owner[:])

	// tx_gas_price
	checkOpcodeStack(t, result, 44, vm.GASPRICE)
	checkOpcode(t, result, 45, vm.POP, 1)

	// tx_ink_price
	checkOpcodeStack(t, result, 46, vm.GASPRICE)
	checkOpcode(t, result, 47, vm.POP, 1)

	// block_number
	checkOpcodeStack(t, result, 48, vm.NUMBER)
	checkOpcode(t, result, 49, vm.POP, 1)

	// evm_gas_left
	checkOpcodeStack(t, result, 50, vm.GAS)
	checkOpcode(t, result, 51, vm.POP, 1)

	// evm_ink_left
	checkOpcodeStack(t, result, 52, vm.GAS)
	checkOpcode(t, result, 53, vm.POP, 1)
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
	checkOpcodeStack(t, result, 3, vm.LOG4, expectedStack...)
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
	checkOpcodeStack(t, result, 3, vm.RETURNDATASIZE)
	checkOpcodeStack(t, result, 4, vm.POP, nil)
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
	innerArgs := argsForStorageRead(key)
	gas := common.Hex2Bytes("ffffffffffffffff")
	argsLen := intToBytes(len(innerArgs))
	returnLen := intToBytes(32)

	args := argsForMulticall(vm.CALL, storage, nil, innerArgs)
	args = multicallAppend(args, vm.DELEGATECALL, storage, innerArgs)
	args = multicallAppend(args, vm.STATICCALL, storage, innerArgs)
	result := sendAndTraceTransaction(t, builder, multicall, nil, args)

	// call_contract
	checkOpcodeStack(t, result, 6, vm.CALL, gas, storage[:], nil, nil, argsLen, nil, nil)
	checkOpcodeStack(t, result, 7, vm.POP, nil)

	// read_return_data
	checkOpcodeStack(t, result, 8, vm.RETURNDATACOPY, nil, nil, returnLen)

	// delegate_call_contract
	checkOpcodeStack(t, result, 12, vm.DELEGATECALL, gas, storage[:], nil, argsLen, nil, nil)

	// static_call_contract
	checkOpcodeStack(t, result, 18, vm.STATICCALL, gas, storage[:], nil, argsLen, nil, nil)
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
	checkOpcodeStack(t, result, 10, vm.CREATE, startValue.Bytes(), nil, intToBytes(len(deployCode)))
	checkOpcodeStack(t, result, 11, vm.POP, create1Addr[:])

	// create2
	create2Args := []byte{0x02}
	create2Args = append(create2Args, common.BigToHash(startValue).Bytes()...)
	create2Args = append(create2Args, salt[:]...)
	create2Args = append(create2Args, deployCode...)
	result = sendAndTraceTransaction(t, builder, program, startValue, create2Args)
	checkOpcodeStack(t, result, 10, vm.CREATE2, startValue.Bytes(), nil, intToBytes(len(deployCode)), salt[:])
	checkOpcodeStack(t, result, 11, vm.POP, create2Addr[:])
}
