// Copyright 2023-2024, Offchain Labs, Inc.
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
	am "github.com/offchainlabs/nitro/util/arbmath"
)

type RequestHandler func(req RequestType, input []byte) ([]byte, []byte, uint64)

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
	AccountCode
	AccountCodeHash
	AddPages
	CaptureHostIO
)

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
	depth := evm.Depth()
	db := evm.StateDB

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

		startGas := gas

		// computes makeCallVariantGasCallEIP2929 and gasCall/gasDelegateCall/gasStaticCall
		baseCost, err := vm.WasmCallCost(db, contract, value, startGas)
		if err != nil {
			return nil, gas, err
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
			gas = am.SaturatingUAdd(gas, params.CallStipend)
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
		return ret, cost, err
	}
	create := func(code []byte, endowment, salt *big.Int, gas uint64) (common.Address, []byte, uint64, error) {
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
		cost := vm.WasmAccountTouchCost(evm.StateDB, address, false)
		balance := evm.StateDB.GetBalance(address)
		return common.BigToHash(balance), cost
	}
	accountCode := func(address common.Address, gas uint64) ([]byte, uint64) {
		// In the future it'll be possible to know the size of a contract before loading it.
		// For now, require the worst case before doing the load.

		cost := vm.WasmAccountTouchCost(evm.StateDB, address, true)
		if gas < cost {
			return []byte{}, cost
		}
		return evm.StateDB.GetCode(address), cost
	}
	accountCodehash := func(address common.Address) (common.Hash, uint64) {
		cost := vm.WasmAccountTouchCost(evm.StateDB, address, false)
		return evm.StateDB.GetCodeHash(address), cost
	}
	addPages := func(pages uint16) uint64 {
		open, ever := db.AddStylusPages(pages)
		return memoryModel.GasCost(pages, open, ever)
	}
	captureHostio := func(name string, args, outs []byte, startInk, endInk uint64) {
		tracingInfo.Tracer.CaptureStylusHostio(name, args, outs, startInk, endInk)
	}

	return func(req RequestType, input []byte) ([]byte, []byte, uint64) {
		switch req {
		case GetBytes32:
			if len(input) != 32 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			key := common.BytesToHash(input)

			out, cost := getBytes32(key)

			return out[:], nil, cost
		case SetBytes32:
			if len(input) != 64 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			key := common.BytesToHash(input[:32])
			value := common.BytesToHash(input[32:])

			cost, err := setBytes32(key, value)

			if err != nil {
				return []byte{0}, nil, 0
			}
			return []byte{1}, nil, cost
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
			contract := common.BytesToAddress(input[:20])
			value := common.BytesToHash(input[20:52]).Big()
			gas := binary.BigEndian.Uint64(input[52:60])
			input = input[60:]

			ret, cost, err := doCall(contract, opcode, input, gas, value)

			statusByte := byte(0)
			if err != nil {
				statusByte = 2 //TODO: err value
			}
			return []byte{statusByte}, ret, cost
		case Create1, Create2:
			if len(input) < 40 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			gas := binary.BigEndian.Uint64(input[0:8])
			endowment := common.BytesToHash(input[8:40]).Big()
			var code []byte
			var salt *big.Int
			switch req {
			case Create1:
				code = input[40:]
			case Create2:
				if len(input) < 72 {
					log.Crit("bad API call", "request", req, "len", len(input))
				}
				salt = common.BytesToHash(input[40:72]).Big()
				code = input[72:]
			default:
				log.Crit("unsupported create opcode", "request", req)
			}

			address, retVal, cost, err := create(code, endowment, salt, gas)

			if err != nil {
				res := append([]byte{0}, []byte(err.Error())...)
				return res, nil, gas
			}
			res := append([]byte{1}, address.Bytes()...)
			return res, retVal, cost
		case EmitLog:
			if len(input) < 4 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			topics := binary.BigEndian.Uint32(input[0:4])
			input = input[4:]
			hashes := make([]common.Hash, topics)
			if len(input) < int(topics*32) {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			for i := uint32(0); i < topics; i++ {
				hashes[i] = common.BytesToHash(input[i*32 : (i+1)*32])
			}

			err := emitLog(hashes, input[topics*32:])

			if err != nil {
				return []byte(err.Error()), nil, 0
			}
			return []byte{}, nil, 0
		case AccountBalance:
			if len(input) != 20 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			address := common.BytesToAddress(input)

			balance, cost := accountBalance(address)

			return balance[:], nil, cost
		case AccountCode:
			if len(input) != 28 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			address := common.BytesToAddress(input[0:20])
			gas := binary.BigEndian.Uint64(input[20:28])

			code, cost := accountCode(address, gas)

			return nil, code, cost
		case AccountCodeHash:
			if len(input) != 20 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			address := common.BytesToAddress(input)

			codeHash, cost := accountCodehash(address)

			return codeHash[:], nil, cost
		case AddPages:
			if len(input) != 2 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			pages := binary.BigEndian.Uint16(input)

			cost := addPages(pages)

			return []byte{}, nil, cost
		case CaptureHostIO:
			if len(input) < 22 {
				log.Crit("bad API call", "request", req, "len", len(input))
			}
			if tracingInfo == nil {
				return []byte{}, nil, 0
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

			captureHostio(name, args, outs, startInk, endInk)

			return []byte{}, nil, 0
		default:
			log.Crit("unsupported call type", "req", req)
			return []byte{}, nil, 0
		}
	}
}
