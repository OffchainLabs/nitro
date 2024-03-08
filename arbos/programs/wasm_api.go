// Copyright 2023, Offchain Labs, Inc.
// For license information, see https://github.com/OffchainLabs/nitro/blob/master/LICENSE

//go:build wasm
// +build wasm

package programs

import (
	"unsafe"

	"github.com/offchainlabs/nitro/arbutil"
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
	msgSender unsafe.Pointer,
	msgValue unsafe.Pointer,
	txGasPrice unsafe.Pointer,
	txOrigin unsafe.Pointer,
	reentrant uint32,
) evmDataHandler

func (params *goParams) createHandler() stylusConfigHandler {
	return createStylusConfig(uint32(params.version), params.maxDepth, params.inkPrice.ToUint32(), params.debugMode)
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
		arbutil.SliceToUnsafePointer(data.msgSender[:]),
		arbutil.SliceToUnsafePointer(data.msgValue[:]),
		arbutil.SliceToUnsafePointer(data.txGasPrice[:]),
		arbutil.SliceToUnsafePointer(data.txOrigin[:]),
		data.reentrant)
}
