// Copyright 2022-2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build js
// +build js

package programs

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/offchainlabs/nitro/arbos/burn"
	"github.com/offchainlabs/nitro/arbos/util"
	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type addr = common.Address
type hash = common.Hash

// rust types
type u8 = uint8
type u16 = uint16
type u32 = uint32
type u64 = uint64
type usize = uintptr

// opaque types
type rustVec byte
type rustConfig byte
type rustModule byte
type rustEvmData byte

func activateWasmRustImpl(
	wasm []byte, pageLimit, version u16, debugMode u32, moduleHash *hash, gas *u64,
) (footprint u16, err *rustVec)

func callUserWasmRustImpl(
	moduleHash *hash, calldata []byte, params *rustConfig, evmApi []byte, evmData *rustEvmData, gas *u64,
) (status userStatus, out *rustVec)

func readRustVecLenImpl(vec *rustVec) (len u32)
func rustVecIntoSliceImpl(vec *rustVec, ptr *byte)
func rustModuleDropImpl(mach *rustModule)
func rustConfigImpl(version u16, maxDepth, inkPrice, debugMode u32) *rustConfig
func rustEvmDataImpl(
	blockBasefee *hash,
	chainId u64,
	blockCoinbase *addr,
	blockGasLimit u64,
	blockNumber u64,
	blockTimestamp u64,
	contractAddress *addr,
	msgSender *addr,
	msgValue *hash,
	txGasPrice *hash,
	txOrigin *addr,
	reentrant u32,
) *rustEvmData

func activateProgram(
	db vm.StateDB,
	program addr,
	wasm []byte,
	pageLimit u16,
	version u16,
	debug bool,
	burner burn.Burner,
) (common.Hash, u16, error) {
	debugMode := arbmath.BoolToUint32(debug)
	moduleHash := common.Hash{}
	gasPtr := burner.GasLeft()

	footprint, err := activateWasmRustImpl(wasm, pageLimit, version, debugMode, &moduleHash, gasPtr)
	if err != nil {
		_, _, err := userFailure.toResult(err.intoSlice(), debug)
		return moduleHash, footprint, err
	}
	return moduleHash, footprint, nil
}

func callUserWasm(
	address common.Address,
	program Program,
	scope *vm.ScopeContext,
	db vm.StateDB,
	interpreter *vm.EVMInterpreter,
	tracingInfo *util.TracingInfo,
	calldata []byte,
	evmData *evmData,
	params *goParams,
	memoryModel *MemoryModel,
) ([]byte, error) {
	evmApi := newApi(interpreter, tracingInfo, scope, memoryModel)
	defer evmApi.drop()
	debug := arbmath.UintToBool(params.debugMode)

	status, output := callUserWasmRustImpl(
		&program.moduleHash,
		calldata,
		params.encode(),
		evmApi.funcs,
		evmData.encode(),
		&scope.Contract.Gas,
	)
	data, _, err := status.toResult(output.intoSlice(), debug)
	return data, err
}

func (vec *rustVec) intoSlice() []byte {
	len := readRustVecLenImpl(vec)
	slice := make([]byte, len)
	rustVecIntoSliceImpl(vec, arbutil.SliceToPointer(slice))
	return slice
}

func (p *goParams) encode() *rustConfig {
	return rustConfigImpl(p.version, p.maxDepth, p.inkPrice.ToUint32(), p.debugMode)
}

func (d *evmData) encode() *rustEvmData {
	return rustEvmDataImpl(
		&d.blockBasefee,
		u64(d.chainId),
		&d.blockCoinbase,
		u64(d.blockGasLimit),
		u64(d.blockNumber),
		u64(d.blockTimestamp),
		&d.contractAddress,
		&d.msgSender,
		&d.msgValue,
		&d.txGasPrice,
		&d.txOrigin,
		u32(d.reentrant),
	)
}
