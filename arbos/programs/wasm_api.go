// Copyright 2023-2024, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build wasm
// +build wasm

package programs

import (
	"unsafe"

	"github.com/offchainlabs/nitro/arbutil"
	"github.com/offchainlabs/nitro/util/arbmath"
)

type stylusConfigHandler uint64

//go:wasmimport programs create_stylus_config
func createStylusConfig(version uint32, max_depth uint32, ink_price uint32, debug uint32) stylusConfigHandler

type evmDataHandler uint64

//go:wasmimport programs create_evm_data
func createEvmData(
	blockBaseFee unsafe.Pointer,
	chainid uint64,
	blockCoinbase unsafe.Pointer,
	gasLimit uint64,
	blockNumber uint64,
	blockTimestamp uint64,
	contractAddress unsafe.Pointer,
	moduleHash unsafe.Pointer,
	msgSender unsafe.Pointer,
	msgValue unsafe.Pointer,
	txGasPrice unsafe.Pointer,
	txOrigin unsafe.Pointer,
	cached uint32,
	reentrant uint32,
) evmDataHandler

func (params *goParams) createHandler() stylusConfigHandler {
	debug := arbmath.BoolToUint32(params.debugMode)
	return createStylusConfig(uint32(params.version), params.maxDepth, params.inkPrice.ToUint32(), debug)
}

func (data *evmData) createHandler() evmDataHandler {
	return createEvmData(
		arbutil.SliceToUnsafePointer(data.blockBasefee[:]),
		data.chainId,
		arbutil.SliceToUnsafePointer(data.blockCoinbase[:]),
		data.blockGasLimit,
		data.blockNumber,
		data.blockTimestamp,
		arbutil.SliceToUnsafePointer(data.contractAddress[:]),
		arbutil.SliceToUnsafePointer(data.moduleHash[:]),
		arbutil.SliceToUnsafePointer(data.msgSender[:]),
		arbutil.SliceToUnsafePointer(data.msgValue[:]),
		arbutil.SliceToUnsafePointer(data.txGasPrice[:]),
		arbutil.SliceToUnsafePointer(data.txOrigin[:]),
		arbmath.BoolToUint32(data.cached),
		data.reentrant,
	)
}
