// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

package programs

import (
	"encoding/binary"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type RequestHandler func(req RequestType, input []byte) ([]byte, uint64)

type RequestType int

const (
	GetBytes32 RequestType = iota
	SetBytes32
	ContractCall
	DelegateCall
	StaticCall
	Create1
	Create2
	EmitLog
	AccountBalance
	AccountCodeHash
	AddPages
	CaptureHostIO
)

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
	depth := evm.Depth()
	db := evm.StateDB

	return func(req RequestType, input []byte) ([]byte, uint64) {
		switch req {
		case GetBytes32:
			if len(input) != 32 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			key := common.BytesToHash(input)
			if tracingInfo != nil {
				tracingInfo.RecordStorageGet(key)
			}
			cost := vm.WasmStateLoadCost(db, actingAddress, key)
			out := db.GetState(actingAddress, key)
			return out[:], cost
		case SetBytes32:
			if len(input) != 64 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			key := common.BytesToHash(input[:32])
			value := common.BytesToHash(input[32:])
			if tracingInfo != nil {
				tracingInfo.RecordStorageSet(key, value)
			}
			log.Error("API: SetBytes32", "key", key, "value", value, "readonly", readOnly)
			if readOnly {
				return []byte{0}, 0
			}
			cost := vm.WasmStateStoreCost(db, actingAddress, key, value)
			db.SetState(actingAddress, key, value)
			return []byte{1}, cost
		case ContractCall, DelegateCall, StaticCall:
			if len(input) < 60 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			var opcode vm.OpCode
			switch req {
			case ContractCall:
				opcode = vm.CALL
			case DelegateCall:
				opcode = vm.DELEGATECALL
			case StaticCall:
				opcode = vm.STATICCALL
			default:
				log.Crit("unsupported call type", "opcode", opcode)
			}

			// This closure can perform each kind of contract call based on the opcode passed in.
			// The implementation for each should match that of the EVM.
			//
			// Note that while the Yellow Paper is authoritative, the following go-ethereum
			// functions provide corresponding implementations in the vm package.
			//     - operations_acl.go makeCallVariantGasCallEIP2929()
			//     - gas_table.go      gasCall() gasDelegateCall() gasStaticCall()
			//     - instructions.go   opCall()  opDelegateCall()  opStaticCall()
			//
			contract := common.BytesToAddress(input[:20])
			value := common.BytesToHash(input[20:52]).Big()
			gas := binary.BigEndian.Uint64(input[52:60])
			input = input[60:]

			// read-only calls are not payable (opCall)
			if readOnly && value.Sign() != 0 {
				return []byte{2}, 0 //TODO: err value
			}

			startGas := gas

			// computes makeCallVariantGasCallEIP2929 and gasCall/gasDelegateCall/gasStaticCall
			baseCost, err := vm.WasmCallCost(db, contract, value, startGas)
			if err != nil {
				return []byte{2}, 0 //TODO: err value
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

			switch req {
			case ContractCall:
				ret, returnGas, err = evm.Call(scope.Contract, contract, input, gas, value)
			case DelegateCall:
				ret, returnGas, err = evm.DelegateCall(scope.Contract, contract, input, gas)
			case StaticCall:
				ret, returnGas, err = evm.StaticCall(scope.Contract, contract, input, gas)
			default:
				log.Crit("unsupported call type", "opcode", opcode)
			}

			interpreter.SetReturnData(ret)
			cost := arbmath.SaturatingUSub(startGas, returnGas+one64th) // user gets 1/64th back
			statusByte := byte(0)
			if err != nil {
				statusByte = 2 //TODO: err value
			}
			ret = append([]byte{statusByte}, ret...)
			return ret, cost
		case Create1, Create2:
			// This closure can perform both kinds of contract creation based on the salt passed in.
			// The implementation for each should match that of the EVM.
			//
			// Note that while the Yellow Paper is authoritative, the following go-ethereum
			// functions provide corresponding implementations in the vm package.
			//     - instructions.go opCreate() opCreate2()
			//     - gas_table.go    gasCreate() gasCreate2()
			//

			if len(input) < 40 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			gas := binary.BigEndian.Uint64(input[0:8])
			endowment := common.BytesToHash(input[8:40]).Big()
			var code []byte
			var salt *big.Int
			var opcode vm.OpCode
			switch req {
			case Create1:
				opcode = vm.CREATE
				code = input[40:]
			case Create2:
				opcode = vm.CREATE2
				if len(input) < 72 {
					log.Crit("bad API call", "request", req, "len", len(input))
				}
				salt = common.BytesToHash(input[40:72]).Big()
				code = input[72:]
			default:
				log.Crit("unsupported create opcode", "opcode", opcode)
			}

			zeroAddr := common.Address{}
			startGas := gas

			if readOnly {
				res := []byte(vm.ErrWriteProtection.Error())
				res = append([]byte{0}, res...)
				return res, 0
			}

			// pay for static and dynamic costs (gasCreate and gasCreate2)
			baseCost := params.CreateGas
			if opcode == vm.CREATE2 {
				keccakWords := arbmath.WordsForBytes(uint64(len(code)))
				keccakCost := arbmath.SaturatingUMul(params.Keccak256WordGas, keccakWords)
				baseCost = arbmath.SaturatingUAdd(baseCost, keccakCost)
			}
			if gas < baseCost {
				res := []byte(vm.ErrOutOfGas.Error())
				res = append([]byte{0}, res...)
				return res, gas
			}
			gas -= baseCost

			// apply the 63/64ths rule
			one64th := gas / 64
			gas -= one64th

			// Tracing: emit the create
			if tracingInfo != nil {
				tracingInfo.Tracer.CaptureState(0, opcode, startGas, baseCost+gas, scope, []byte{}, depth, nil)
			}

			var result []byte
			var addr common.Address // zero on failure
			var returnGas uint64
			var suberr error

			if opcode == vm.CREATE {
				result, addr, returnGas, suberr = evm.Create(contract, code, gas, endowment)
			} else {
				salt256, _ := uint256.FromBig(salt)
				result, addr, returnGas, suberr = evm.Create2(contract, code, gas, endowment, salt256)
			}
			if suberr != nil {
				addr = zeroAddr
			}
			if !errors.Is(suberr, vm.ErrExecutionReverted) {
				result = nil // returnData is only provided in the revert case (opCreate)
			}

			cost := arbmath.SaturatingUSub(startGas, returnGas+one64th) // user gets 1/64th back
			res := append([]byte{1}, addr.Bytes()...)
			res = append(res, result...)
			return res, cost
		case EmitLog:
			if len(input) < 4 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			if readOnly {
				return []byte(vm.ErrWriteProtection.Error()), 0
			}
			topics := binary.BigEndian.Uint32(input[0:4])
			input = input[4:]
			if len(input) < int(topics*32) {
				log.Crit("bad emitLog", "request", req, "len", len(input)+4, "min expected", topics*32+4)
			}
			hashes := make([]common.Hash, topics)
			for i := uint32(0); i < topics; i++ {
				hashes[i] = common.BytesToHash(input[:(i+1)*32])
			}
			event := &types.Log{
				Address:     actingAddress,
				Topics:      hashes,
				Data:        input[32*topics:],
				BlockNumber: evm.Context.BlockNumber.Uint64(),
				// Geth will set other fields
			}
			db.AddLog(event)
			return []byte{}, 0
		case AccountBalance:
			if len(input) != 20 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			address := common.BytesToAddress(input)
			cost := vm.WasmAccountTouchCost(evm.StateDB, address)
			balance := common.BigToHash(evm.StateDB.GetBalance(address))
			return balance[:], cost
		case AccountCodeHash:
			if len(input) != 20 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			address := common.BytesToAddress(input)
			cost := vm.WasmAccountTouchCost(evm.StateDB, address)
			codeHash := common.Hash{}
			if !evm.StateDB.Empty(address) {
				codeHash = evm.StateDB.GetCodeHash(address)
			}
			return codeHash[:], cost
		case AddPages:
			if len(input) != 2 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			pages := binary.BigEndian.Uint16(input)
			open, ever := db.AddStylusPages(pages)
			cost := memoryModel.GasCost(pages, open, ever)
			return []byte{}, cost
		case CaptureHostIO:
			if len(input) < 22 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			if tracingInfo == nil {
				return []byte{}, 0
			}
			startInk := binary.BigEndian.Uint64(input[:8])
			endInk := binary.BigEndian.Uint64(input[8:16])
			nameLen := binary.BigEndian.Uint16(input[16:18])
			argsLen := binary.BigEndian.Uint16(input[18:20])
			outsLen := binary.BigEndian.Uint16(input[20:22])
			if len(input) != 22+int(nameLen+argsLen+outsLen) {
				log.Error("bad API call", "request", req, "len", len(input), "expected", nameLen+argsLen+outsLen)
			}
			name := string(input[22 : 22+nameLen])
			args := input[22+nameLen : 22+nameLen+argsLen]
			outs := input[22+nameLen+argsLen:]
			tracingInfo.Tracer.CaptureStylusHostio(name, args, outs, startInk, endInk)
			return []byte{}, 0
		default:
			log.Crit("unsupported call type", "req", req)
			return []byte{}, 0
		}
	}
}
