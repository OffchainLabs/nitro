// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package programs

import (
	"strconv"

	"github.com/holiman/uint256"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"

	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
	am "github.com/offchainlabs/nitro/util/arbmath"
)

type RequestHandler func(req RequestType, input []byte) ([]byte, []byte, uint64)

type RequestType int
type u256 = uint256.Int

const (
	GetBytes32 RequestType = iota
	SetTrieSlots
	GetTransientBytes32
	SetTransientBytes32
	ContractCall
	DelegateCall
	StaticCall
	Create1
	Create2
	EmitLog
	AccountBalance
	AccountCode
	AccountCodeHash
	AddPages
	CaptureHostIO
)

type apiStatus uint8

const (
	Success apiStatus = iota
	Failure
	OutOfGas
	WriteProtection
)

func (s apiStatus) to_slice() []byte {
	return []byte{uint8(s)}
}

const EvmApiMethodReqOffset = 0x10000000

func newApiClosures(
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	scope *vm.ScopeContext,
	memoryModel *MemoryModel,
) RequestHandler {
	contract := scope.Contract
	actingAddress := contract.Address() // not necessarily WASM
	readOnly := interpreter.ReadOnly()
	evm := interpreter.Evm()
	db := evm.StateDB
	chainConfig := evm.ChainConfig()

	getBytes32 := func(key common.Hash) (common.Hash, uint64) {
		cost := vm.WasmStateLoadCost(db, actingAddress, key)
		return db.GetState(actingAddress, key), cost
	}
	setTrieSlots := func(data []byte, gasLeft *uint64) apiStatus {
		for len(data) > 0 {
			key := common.BytesToHash(data[:32])
			value := common.BytesToHash(data[32:64])
			data = data[64:]

			if readOnly {
				return WriteProtection
			}

			cost := vm.WasmStateStoreCost(db, actingAddress, key, value)
			if cost > *gasLeft {
				*gasLeft = 0
				return OutOfGas
			}
			*gasLeft -= cost
			db.SetState(actingAddress, key, value)
		}
		return Success
	}
	getTransientBytes32 := func(key common.Hash) common.Hash {
		return db.GetTransientState(actingAddress, key)
	}
	setTransientBytes32 := func(key, value common.Hash) apiStatus {
		if readOnly {
			return WriteProtection
		}
		db.SetTransientState(actingAddress, key, value)
		return Success
	}
	doCall := func(
		contract common.Address, opcode vm.OpCode, input []byte, gasLeft, gasReq uint64, value *u256,
	) ([]byte, uint64, error) {
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
			return nil, 0, vm.ErrWriteProtection
		}

		// computes makeCallVariantGasCallEIP2929 and gasCall/gasDelegateCall/gasStaticCall
		baseCost, err := vm.WasmCallCost(db, contract, value, gasLeft)
		if err != nil {
			return nil, gasLeft, err
		}

		// apply the 63/64ths rule
		startGas := am.SaturatingUSub(gasLeft, baseCost) * 63 / 64
		gas := am.MinInt(startGas, gasReq)

		// EVM rule: calls that pay get a stipend (opCall)
		if value.Sign() != 0 {
			gas = am.SaturatingUAdd(gas, params.CallStipend)
		}

		// Tracing: emit the call (value transfer is done later in evm.Call)
		if tracingInfo != nil {
			tracingInfo.CaptureStylusCall(opcode, contract, value, input, gas, startGas, baseCost)
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
			panic("unsupported call type: " + opcode.String())
		}

		interpreter.SetReturnData(ret)
		cost := am.SaturatingUAdd(baseCost, am.SaturatingUSub(gas, returnGas))
		return ret, cost, err
	}
	create := func(code []byte, endowment, salt *u256, gas uint64) (common.Address, []byte, uint64, error) {
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
			return zeroAddr, nil, 0, vm.ErrWriteProtection
		}

		// pay for static and dynamic costs (gasCreate and gasCreate2)
		baseCost := params.CreateGas
		if opcode == vm.CREATE2 {
			keccakWords := am.WordsForBytes(uint64(len(code)))
			keccakCost := am.SaturatingUMul(params.Keccak256WordGas, keccakWords)
			baseCost = am.SaturatingUAdd(baseCost, keccakCost)
		}
		if gas < baseCost {
			return zeroAddr, nil, gas, vm.ErrOutOfGas
		}
		gas -= baseCost

		// apply the 63/64ths rule
		one64th := gas / 64
		gas -= one64th

		var res []byte
		var addr common.Address // zero on failure
		var returnGas uint64
		var suberr error

		if opcode == vm.CREATE {
			res, addr, returnGas, suberr = evm.Create(contract, code, gas, endowment)
		} else {
			res, addr, returnGas, suberr = evm.Create2(contract, code, gas, endowment, salt)
		}
		if suberr != nil {
			addr = zeroAddr
		}
		// This matches geth behavior of doing an exact error comparison instead of errors.Is
		// See e.g. EVM's create method or the opCreate function for references of how geth checks this
		//nolint:errorlint
		if suberr != vm.ErrExecutionReverted {
			res = nil // returnData is only provided in the revert case (opCreate)
		}
		interpreter.SetReturnData(res)
		cost := arbmath.SaturatingUSub(startGas, returnGas+one64th) // user gets 1/64th back
		return addr, res, cost, nil
	}
	emitLog := func(topics []common.Hash, data []byte) error {
		if readOnly {
			return vm.ErrWriteProtection
		}
		event := &types.Log{
			Address:     actingAddress,
			Topics:      topics,
			Data:        data,
			BlockNumber: evm.Context.BlockNumber.Uint64(),
			// Geth will set other fields
		}
		db.AddLog(event)
		return nil
	}
	accountBalance := func(address common.Address) (common.Hash, uint64) {
		cost := vm.WasmAccountTouchCost(chainConfig, evm.StateDB, address, false)
		balance := evm.StateDB.GetBalance(address)
		return balance.Bytes32(), cost
	}
	accountCode := func(address common.Address, gas uint64) ([]byte, uint64) {
		// In the future it'll be possible to know the size of a contract before loading it.
		// For now, require the worst case before doing the load.

		cost := vm.WasmAccountTouchCost(chainConfig, evm.StateDB, address, true)
		if gas < cost {
			return []byte{}, cost
		}
		return evm.StateDB.GetCode(address), cost
	}
	accountCodehash := func(address common.Address) (common.Hash, uint64) {
		cost := vm.WasmAccountTouchCost(chainConfig, evm.StateDB, address, false)
		return evm.StateDB.GetCodeHash(address), cost
	}
	addPages := func(pages uint16) uint64 {
		open, ever := db.AddStylusPages(pages)
		return memoryModel.GasCost(pages, open, ever)
	}
	captureHostio := func(name string, args, outs []byte, startInk, endInk uint64) {
		if tracingInfo.Tracer != nil && tracingInfo.Tracer.CaptureStylusHostio != nil {
			tracingInfo.Tracer.CaptureStylusHostio(name, args, outs, startInk, endInk)
		}
		tracingInfo.CaptureEVMTraceForHostio(name, args, outs, startInk, endInk)
	}

	return func(req RequestType, input []byte) ([]byte, []byte, uint64) {
		original := input

		crash := func(reason string) {
			panic("bad API call reason: " + reason + " request: " + strconv.Itoa(int(req)) + " len: " + strconv.Itoa(len(original)) + " remaining: " + strconv.Itoa(len(input)))
		}
		takeInput := func(needed int, reason string) []byte {
			if len(input) < needed {
				crash(reason)
			}
			data := input[:needed]
			input = input[needed:]
			return data
		}
		defer func() {
			if len(input) > 0 {
				crash("extra input")
			}
		}()

		takeAddress := func() common.Address {
			return common.BytesToAddress(takeInput(20, "expected address"))
		}
		takeHash := func() common.Hash {
			return common.BytesToHash(takeInput(32, "expected hash"))
		}
		takeU256 := func() *u256 {
			return am.BytesToUint256(takeInput(32, "expected big"))
		}
		takeU64 := func() uint64 {
			return am.BytesToUint(takeInput(8, "expected u64"))
		}
		takeU32 := func() uint32 {
			return am.BytesToUint32(takeInput(4, "expected u32"))
		}
		takeU16 := func() uint16 {
			return am.BytesToUint16(takeInput(2, "expected u16"))
		}
		takeFixed := func(needed int) []byte {
			return takeInput(needed, "expected value with known length")
		}
		takeRest := func() []byte {
			data := input
			input = []byte{}
			return data
		}

		switch req {
		case GetBytes32:
			key := takeHash()
			out, cost := getBytes32(key)
			return out[:], nil, cost
		case SetTrieSlots:
			gasLeft := takeU64()
			gas := gasLeft
			status := setTrieSlots(takeRest(), &gas)
			return status.to_slice(), nil, gasLeft - gas
		case GetTransientBytes32:
			key := takeHash()
			out := getTransientBytes32(key)
			return out[:], nil, 0
		case SetTransientBytes32:
			key := takeHash()
			value := takeHash()
			status := setTransientBytes32(key, value)
			return status.to_slice(), nil, 0
		case ContractCall, DelegateCall, StaticCall:
			var opcode vm.OpCode
			switch req {
			case ContractCall:
				opcode = vm.CALL
			case DelegateCall:
				opcode = vm.DELEGATECALL
			case StaticCall:
				opcode = vm.STATICCALL
			default:
				panic("unsupported call type opcode: " + opcode.String())
			}
			contract := takeAddress()
			value := takeU256()
			gasLeft := takeU64()
			gasReq := takeU64()
			calldata := takeRest()

			ret, cost, err := doCall(contract, opcode, calldata, gasLeft, gasReq, value)
			statusByte := byte(0)
			if err != nil {
				statusByte = 2 // TODO: err value
			}
			return []byte{statusByte}, ret, cost
		case Create1, Create2:
			gas := takeU64()
			endowment := takeU256()
			var salt *u256
			if req == Create2 {
				salt = takeU256()
			}
			code := takeRest()

			address, retVal, cost, err := create(code, endowment, salt, gas)
			if err != nil {
				res := append([]byte{0}, []byte(err.Error())...)
				return res, nil, gas
			}
			res := append([]byte{1}, address.Bytes()...)
			return res, retVal, cost
		case EmitLog:
			topics := takeU32()
			hashes := make([]common.Hash, topics)
			for i := uint32(0); i < topics; i++ {
				hashes[i] = takeHash()
			}

			err := emitLog(hashes, takeRest())
			if err != nil {
				return []byte(err.Error()), nil, 0
			}
			return []byte{}, nil, 0
		case AccountBalance:
			address := takeAddress()
			balance, cost := accountBalance(address)
			return balance[:], nil, cost
		case AccountCode:
			address := takeAddress()
			gas := takeU64()
			code, cost := accountCode(address, gas)
			return nil, code, cost
		case AccountCodeHash:
			address := takeAddress()
			codeHash, cost := accountCodehash(address)
			return codeHash[:], nil, cost
		case AddPages:
			pages := takeU16()
			cost := addPages(pages)
			return []byte{}, nil, cost
		case CaptureHostIO:
			if tracingInfo == nil {
				takeRest() // drop any input
				return []byte{}, nil, 0
			}
			startInk := takeU64()
			endInk := takeU64()
			nameLen := takeU32()
			argsLen := takeU32()
			outsLen := takeU32()
			name := string(takeFixed(int(nameLen)))
			args := takeFixed(int(argsLen))
			outs := takeFixed(int(outsLen))

			captureHostio(name, args, outs, startInk, endInk)
			return []byte{}, nil, 0
		default:
			panic("unsupported call type: " + strconv.Itoa(int(req)))
		}
	}
}
