// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/nitro/blob/master/LICENSE

//go:build !js
// +build !js

package programs

/*
#cgo CFLAGS: -g -Wall -I../../target/include/
#cgo LDFLAGS: ${SRCDIR}/../../target/lib/libstylus.a -ldl -lm
#include "arbitrator.h"

typedef uint32_t u32;
typedef uint64_t u64;
typedef size_t usize;

Bytes32     getBytes32Wrap(usize api, Bytes32 key, u64 * cost);
GoApiStatus setBytes32Wrap(usize api, Bytes32 key, Bytes32 value, u64 * cost, RustVec * error);
GoApiStatus contractCallWrap(usize api, Bytes20 contract, RustVec * data, u64 * gas, Bytes32 value, u32 * len);
GoApiStatus delegateCallWrap(usize api, Bytes20 contract, RustVec * data, u64 * gas,                u32 * len);
GoApiStatus staticCallWrap  (usize api, Bytes20 contract, RustVec * data, u64 * gas,                u32 * len);
GoApiStatus create1Wrap(usize api, RustVec * code, Bytes32 endowment,               u64 * gas, u32 * len);
GoApiStatus create2Wrap(usize api, RustVec * code, Bytes32 endowment, Bytes32 salt, u64 * gas, u32 * len);
void        getReturnDataWrap(usize api, RustVec * data);
GoApiStatus emitLogWrap(usize api, RustVec * data, usize topics);
*/
import "C"
import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/arbmath"
	"github.com/offchainlabs/nitro/util/colors"
)

type u8 = C.uint8_t
type u32 = C.uint32_t
type u64 = C.uint64_t
type usize = C.size_t
type bytes20 = C.Bytes20
type bytes32 = C.Bytes32
type rustVec = C.RustVec

func compileUserWasm(db vm.StateDB, program common.Address, wasm []byte, version uint32, debug bool) error {
	debugMode := 0
	if debug {
		debugMode = 1
	}

	output := &rustVec{}
	status := userStatus(C.stylus_compile(
		goSlice(wasm),
		u32(version),
		usize(debugMode),
		output,
	))
	data := output.intoBytes()
	result, err := status.output(data)
	if err == nil {
		db.SetCompiledWasmCode(program, result, version)
	} else {
		log.Debug("program failure", "err", err.Error(), "data", string(data), "program", program)
		colors.PrintPink("ERR: ", err.Error(), " ", string(data))
	}
	return err
}

func callUserWasm(
	scope *vm.ScopeContext,
	db vm.StateDB,
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	msg core.Message,
	calldata []byte,
	stylusParams *goParams,
) ([]byte, error) {
	contract := scope.Contract
	readOnly := interpreter.ReadOnly()
	evm := interpreter.Evm()
	depth := evm.Depth()

	actingAddress := contract.Address() // not necessarily WASM
	program := actingAddress
	if contract.CodeAddr != nil {
		program = *contract.CodeAddr
	}
	if db, ok := db.(*state.StateDB); ok {
		db.RecordProgram(program, stylusParams.version)
	}
	module := db.GetCompiledWasmCode(program, stylusParams.version)

	// closures so Rust can call back into Go
	getBytes32 := func(key common.Hash) (common.Hash, uint64) {
		if tracingInfo != nil {
			tracingInfo.RecordStorageGet(key)
		}
		cost := vm.WasmStateLoadCost(db, actingAddress, key)
		return db.GetState(actingAddress, key), cost
	}
	setBytes32 := func(key, value common.Hash) (uint64, error) {
		if tracingInfo != nil {
			tracingInfo.RecordStorageSet(key, value)
		}
		if readOnly {
			return 0, vm.ErrWriteProtection
		}
		cost := vm.WasmStateStoreCost(db, actingAddress, key, value)
		db.SetState(actingAddress, key, value)
		return cost, nil
	}
	doCall := func(
		contract common.Address, opcode vm.OpCode, input []byte, gas uint64, value *big.Int,
	) (uint32, uint64, error) {
		// This closure can perform each kind of contract call based on the opcode passed in.
		// The implementation for each should match that of the EVM.
		//
		// Note that while the Yellow Paper is authoritative, the following go-ethereum
		// functions provide corresponding implementations in the vm package.
		//     - operations_acl.go makeCallVariantGasCallEIP2929()
		//     - gas_table.go      gasCall() gasDelegateCall() gasStaticCall()
		//     - instructions.go   opCall()  opDelegateCall()  opStaticCall()
		//

		// read-only calls are not payable (opCall)
		if readOnly && value.Sign() != 0 {
			return 0, 0, vm.ErrWriteProtection
		}

		startGas := gas

		// computes makeCallVariantGasCallEIP2929 and gasCall/gasDelegateCall/gasStaticCall
		baseCost, err := vm.WasmCallCost(db, contract, value, startGas)
		if err != nil {
			return 0, gas, err
		}
		gas -= baseCost

		// apply the 63/64ths rule
		one64th := gas / 64
		gas -= one64th

		// Tracing: emit the call (value transfer is done later in evm.Call)
		if tracingInfo != nil {
			tracingInfo.Tracer.CaptureState(0, opcode, startGas, baseCost+gas, scope, []byte{}, depth, nil)
		}

		// EVM rule: calls that pay get a stipend (opCall)
		if value.Sign() != 0 {
			gas = arbmath.SaturatingUAdd(gas, params.CallStipend)
		}

		var ret []byte
		var returnGas uint64

		switch opcode {
		case vm.CALL:
			ret, returnGas, err = evm.Call(scope.Contract, contract, input, gas, value)
		case vm.DELEGATECALL:
			ret, returnGas, err = evm.DelegateCall(scope.Contract, contract, input, gas)
		case vm.STATICCALL:
			ret, returnGas, err = evm.StaticCall(scope.Contract, contract, input, gas)
		default:
			log.Crit("unsupported call type", "opcode", opcode)
		}

		interpreter.SetReturnData(ret)
		cost := arbmath.SaturatingUSub(startGas, returnGas+one64th) // user gets 1/64th back
		return uint32(len(ret)), cost, err
	}
	contractCall := func(contract common.Address, input []byte, gas uint64, value *big.Int) (uint32, uint64, error) {
		return doCall(contract, vm.CALL, input, gas, value)
	}
	delegateCall := func(contract common.Address, input []byte, gas uint64) (uint32, uint64, error) {
		return doCall(contract, vm.DELEGATECALL, input, gas, common.Big0)
	}
	staticCall := func(contract common.Address, input []byte, gas uint64) (uint32, uint64, error) {
		return doCall(contract, vm.STATICCALL, input, gas, common.Big0)
	}
	create := func(code []byte, endowment, salt *big.Int, gas uint64) (common.Address, uint32, uint64, error) {
		// This closure can perform both kinds of contract creation based on the salt passed in.
		// The implementation for each should match that of the EVM.
		//
		// Note that while the Yellow Paper is authoritative, the following go-ethereum
		// functions provide corresponding implementations in the vm package.
		//     - instructions.go opCreate() opCreate2()
		//     - gas_table.go    gasCreate() gasCreate2()
		//

		opcode := vm.CREATE
		if salt != nil {
			opcode = vm.CREATE2
		}
		zeroAddr := common.Address{}
		startGas := gas

		if readOnly {
			return zeroAddr, 0, 0, vm.ErrWriteProtection
		}

		// pay for static and dynamic costs (gasCreate and gasCreate2)
		baseCost := params.CreateGas
		if opcode == vm.CREATE2 {
			keccakWords := arbmath.WordsForBytes(uint64(len(code)))
			keccakCost := arbmath.SaturatingUMul(params.Keccak256WordGas, keccakWords)
			baseCost = arbmath.SaturatingUAdd(baseCost, keccakCost)
		}
		if gas < baseCost {
			return zeroAddr, 0, gas, vm.ErrOutOfGas
		}
		gas -= baseCost

		// apply the 63/64ths rule
		one64th := gas / 64
		gas -= one64th

		// Tracing: emit the create
		if tracingInfo != nil {
			tracingInfo.Tracer.CaptureState(0, opcode, startGas, baseCost+gas, scope, []byte{}, depth, nil)
		}

		var res []byte
		var addr common.Address // zero on failure
		var returnGas uint64
		var suberr error

		if opcode == vm.CREATE {
			res, addr, returnGas, suberr = evm.Create(contract, code, gas, endowment)
		} else {
			salt256, _ := uint256.FromBig(salt)
			res, addr, returnGas, suberr = evm.Create2(contract, code, gas, endowment, salt256)
		}
		if suberr != nil {
			addr = zeroAddr
		}
		if !errors.Is(vm.ErrExecutionReverted, suberr) {
			res = nil // returnData is only provided in the revert case (opCreate)
		}
		interpreter.SetReturnData(res)
		cost := arbmath.SaturatingUSub(startGas, returnGas+one64th) // user gets 1/64th back
		return addr, uint32(len(res)), cost, nil
	}
	create1 := func(code []byte, endowment *big.Int, gas uint64) (common.Address, uint32, uint64, error) {
		return create(code, endowment, nil, gas)
	}
	create2 := func(code []byte, endowment, salt *big.Int, gas uint64) (common.Address, uint32, uint64, error) {
		return create(code, endowment, salt, gas)
	}
	getReturnData := func() []byte {
		data := interpreter.GetReturnData()
		if data == nil {
			return []byte{}
		}
		return data
	}
	emitLog := func(data []byte, topics int) error {
		if readOnly {
			return vm.ErrWriteProtection
		}
		hashes := make([]common.Hash, topics)
		for i := 0; i < topics; i++ {
			hashes[i] = common.BytesToHash(data[:(i+1)*32])
		}
		event := &types.Log{
			Address:     actingAddress,
			Topics:      hashes,
			Data:        data[32*topics:],
			BlockNumber: evm.Context.BlockNumber.Uint64(),
			// Geth will set other fields
		}
		db.AddLog(event)
		return nil
	}

	evmData := C.EvmData{
		origin: addressToBytes20(evm.TxContext.Origin),
	}

	output := &rustVec{}
	status := userStatus(C.stylus_call(
		goSlice(module),
		goSlice(calldata),
		stylusParams.encode(),
		newAPI(
			getBytes32, setBytes32,
			contractCall, delegateCall, staticCall, create1, create2, getReturnData,
			emitLog,
		),
		evmData,
		output,
		(*u64)(&contract.Gas),
	))
	data, err := status.output(output.intoBytes())

	if status == userFailure {
		log.Debug("program failure", "err", string(data), "program", actingAddress)
	}
	return data, err
}

type apiStatus = C.GoApiStatus

const apiSuccess C.GoApiStatus = C.GoApiStatus_Success
const apiFailure C.GoApiStatus = C.GoApiStatus_Failure

//export getBytes32Impl
func getBytes32Impl(api usize, key bytes32, cost *u64) bytes32 {
	closure := getAPI(api)
	value, gas := closure.getBytes32(key.toHash())
	*cost = u64(gas)
	return hashToBytes32(value)
}

//export setBytes32Impl
func setBytes32Impl(api usize, key, value bytes32, cost *u64, errVec *rustVec) apiStatus {
	closure := getAPI(api)

	gas, err := closure.setBytes32(key.toHash(), value.toHash())
	if err != nil {
		errVec.setString(err.Error())
		return apiFailure
	}
	*cost = u64(gas)
	return apiSuccess
}

//export contractCallImpl
func contractCallImpl(api usize, contract bytes20, data *rustVec, evmGas *u64, value bytes32, len *u32) apiStatus {
	closure := getAPI(api)
	defer data.drop()

	ret_len, cost, err := closure.contractCall(contract.toAddress(), data.read(), uint64(*evmGas), value.toBig())
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		return apiFailure
	}
	return apiSuccess
}

//export delegateCallImpl
func delegateCallImpl(api usize, contract bytes20, data *rustVec, evmGas *u64, len *u32) apiStatus {
	closure := getAPI(api)
	defer data.drop()

	ret_len, cost, err := closure.delegateCall(contract.toAddress(), data.read(), uint64(*evmGas))
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		return apiFailure
	}
	return apiSuccess
}

//export staticCallImpl
func staticCallImpl(api usize, contract bytes20, data *rustVec, evmGas *u64, len *u32) apiStatus {
	closure := getAPI(api)
	defer data.drop()

	ret_len, cost, err := closure.staticCall(contract.toAddress(), data.read(), uint64(*evmGas))
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		return apiFailure
	}
	return apiSuccess
}

//export create1Impl
func create1Impl(api usize, code *rustVec, endowment bytes32, evmGas *u64, len *u32) apiStatus {
	closure := getAPI(api)
	addr, ret_len, cost, err := closure.create1(code.read(), endowment.toBig(), uint64(*evmGas))
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		code.setString(err.Error())
		return apiFailure
	}
	code.setBytes(addr.Bytes())
	return apiSuccess
}

//export create2Impl
func create2Impl(api usize, code *rustVec, endowment, salt bytes32, evmGas *u64, len *u32) apiStatus {
	closure := getAPI(api)
	addr, ret_len, cost, err := closure.create2(code.read(), endowment.toBig(), salt.toBig(), uint64(*evmGas))
	*evmGas = u64(cost) // evmGas becomes the call's cost
	*len = u32(ret_len)
	if err != nil {
		code.setString(err.Error())
		return apiFailure
	}
	code.setBytes(addr.Bytes())
	return apiSuccess
}

//export getReturnDataImpl
func getReturnDataImpl(api usize, output *rustVec) {
	closure := getAPI(api)
	return_data := closure.getReturnData()
	output.setBytes(return_data)
}

//export emitLogImpl
func emitLogImpl(api usize, data *rustVec, topics usize) apiStatus {
	closure := getAPI(api)
	err := closure.emitLog(data.read(), int(topics))
	if err != nil {
		data.setString(err.Error())
		return apiFailure
	}
	return apiSuccess
}

func (value bytes20) toAddress() common.Address {
	addr := common.Address{}
	for index, b := range value.bytes {
		addr[index] = byte(b)
	}
	return addr
}

func (value bytes32) toHash() common.Hash {
	hash := common.Hash{}
	for index, b := range value.bytes {
		hash[index] = byte(b)
	}
	return hash
}

func (value bytes32) toBig() *big.Int {
	return value.toHash().Big()
}

func hashToBytes32(hash common.Hash) bytes32 {
	value := bytes32{}
	for index, b := range hash.Bytes() {
		value.bytes[index] = u8(b)
	}
	return value
}

func addressToBytes20(addr common.Address) bytes20 {
	value := bytes20{}
	for index, b := range addr.Bytes() {
		value.bytes[index] = u8(b)
	}
	return value
}

func (vec *rustVec) read() []byte {
	return arbutil.PointerToSlice((*byte)(vec.ptr), int(vec.len))
}

func (vec *rustVec) intoBytes() []byte {
	slice := vec.read()
	C.stylus_drop_vec(*vec)
	return slice
}

func (vec *rustVec) drop() {
	C.stylus_drop_vec(*vec)
}

func (vec *rustVec) setString(data string) {
	vec.setBytes([]byte(data))
}

func (vec *rustVec) setBytes(data []byte) {
	C.stylus_vec_set_bytes(vec, goSlice(data))
}

func goSlice(slice []byte) C.GoSliceData {
	return C.GoSliceData{
		ptr: (*u8)(arbutil.SliceToPointer(slice)),
		len: usize(len(slice)),
	}
}

func (params *goParams) encode() C.GoParams {
	return C.GoParams{
		version:    u32(params.version),
		max_depth:  u32(params.maxDepth),
		ink_price:  u64(params.inkPrice),
		hostio_ink: u64(params.hostioInk),
		debug_mode: u32(params.debugMode),
	}
}
